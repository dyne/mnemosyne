package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dyne/mnemosyne/internal/anchor"
	"github.com/dyne/mnemosyne/internal/domain"
	"github.com/dyne/mnemosyne/internal/ledger"
	"github.com/dyne/mnemosyne/internal/merkle"
	"github.com/dyne/mnemosyne/internal/receipts"
	"github.com/dyne/mnemosyne/internal/storage"
	"github.com/dyne/mnemosyne/internal/verifier"
)

// Server is the HTTP API server for Mnemosyne.
type Server struct {
	store        storage.Store
	tree         *merkle.Tree
	ledger       ledger.Backend
	anchor       anchor.Backend
	mux          *http.ServeMux
	contractsDir string
	version      string
}

// ServerConfig holds optional dependencies for the server.
type ServerConfig struct {
	Store        storage.Store
	Tree         *merkle.Tree
	Ledger       ledger.Backend
	Anchor       anchor.Backend
	WebDir       string
	ContractsDir string
	Version      string
}

// NewServer creates a new API server.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		store:        cfg.Store,
		tree:         cfg.Tree,
		ledger:       cfg.Ledger,
		anchor:       cfg.Anchor,
		mux:          http.NewServeMux(),
		contractsDir: cfg.ContractsDir,
		version:      cfg.Version,
	}
	s.routes()
	if cfg.WebDir != "" {
		s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.WebDir+"/static"))))
		s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, cfg.WebDir+"/index.html")
				return
			}
			http.NotFound(w, r)
		})
	}
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /version", s.handleVersion)
	s.mux.HandleFunc("POST /memories", s.handleRemember)
	s.mux.HandleFunc("GET /memories/{id}", s.handleRecall)
	s.mux.HandleFunc("POST /checkpoints", s.handleAnchorBeacon)
	s.mux.HandleFunc("GET /beacons/{id}", s.handleGetBeacon)
	s.mux.HandleFunc("GET /beacons/{id}/memories", s.handleBeaconMemories)
	s.mux.HandleFunc("POST /beacons/{id}/extend", s.handleExtendBeacon)
	s.mux.HandleFunc("GET /proofs/{memory_id}", s.handleGenerateRoute)
	s.mux.HandleFunc("POST /verify", s.handleWitness)
	s.mux.HandleFunc("POST /verify/full", s.handleFullVerify)
	s.mux.HandleFunc("GET /memories/{id}/receipt", s.handleReceiptExport)
	s.mux.HandleFunc("POST /anchors", s.handleCreateAnchor)
	s.mux.HandleFunc("GET /anchors/{id}", s.handleGetAnchor)
	s.mux.HandleFunc("GET /ledger/events", s.handleLedgerEvents)
	s.mux.HandleFunc("GET /ledger/head", s.handleLedgerHead)
	s.mux.HandleFunc("POST /ledger/verify", s.handleLedgerVerify)
	s.mux.HandleFunc("GET /dashboard", s.handleDashboard)
	s.mux.HandleFunc("GET /openapi.json", handleOpenAPI)
	s.mux.HandleFunc("GET /docs", handleDocs)
	s.mux.HandleFunc("GET /docs/", handleDocs)
	s.mux.HandleFunc("GET /contracts", s.handleListContracts)
	s.mux.HandleFunc("GET /contracts/{name}", s.handleGetContract)
}

// Handler returns the http.Handler for the server.
func (s *Server) Handler() http.Handler {
	return s.mux
}

// ---- System ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{"status": "ok"}
	if s.ledger != nil {
		resp["ledger"] = "available"
	}
	if s.anchor != nil {
		resp["anchor"] = s.anchor.Name()
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": s.version,
		"project": "mnemosyne",
	})
}

// ---- Dashboard ----

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	resp := map[string]any{}
	// Storage stats
	memories, err := s.store.MemoriesByBeacon(r.Context(), "current")
	if err == nil {
		resp["pending_memories"] = len(memories)
	}

	latestBeacon, err := s.store.LatestBeacon(r.Context())
	if err == nil {
		resp["latest_beacon"] = latestBeacon
	}

	resp["storage_backend"] = "sqlite"

	// Ledger info
	if s.ledger != nil {
		head, err := s.ledger.LatestHead(r.Context())
		if err == nil {
			resp["ledger_head"] = head
		}
		resp["ledger_backend"] = "ndjson_hash_chain"

		// Ledger stats
		events, _ := s.ledger.ListEvents(r.Context(), domain.LedgerListOptions{Limit: 1})
		lastSeq := uint64(0)
		if len(events) > 0 {
			lastSeq = events[len(events)-1].Seq
		}
		resp["ledger_total_events"] = lastSeq
	}

	// Anchor info
	if s.anchor != nil {
		resp["anchor_backend"] = s.anchor.Name()
	}

	writeJSON(w, http.StatusOK, resp)
}

// ---- Memories ----

func (s *Server) handleRemember(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Payload any `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Payload == nil {
		writeError(w, http.StatusBadRequest, "payload is required")
		return
	}

	// Hash the payload via Zenroom
	payloadJSON, _ := json.Marshal(req.Payload)
	hashResult, err := s.tree.HashPayload(r.Context(), string(payloadJSON))
	if err != nil {
		log.Printf("ERROR hashing payload: %v", err)
		writeError(w, http.StatusInternalServerError, "hashing failed")
		return
	}

	// Store the memory
	beaconID := "current"
	m, err := s.store.Remember(r.Context(), req.Payload, hashResult, beaconID)
	if err != nil {
		log.Printf("ERROR storing memory: %v", err)
		writeError(w, http.StatusInternalServerError, "storage failed")
		return
	}

	// Append ledger event
	var ledgerReceipt *domain.LedgerReceipt
	if s.ledger != nil {
		rec, err := s.ledger.Append(r.Context(), domain.EventMemoryRecorded, map[string]any{
			"memory_id": string(m.ID),
			"leaf_hash": m.LeafHash,
		})
		if err != nil {
			log.Printf("ERROR appending ledger: %v", err)
		} else {
			ledgerReceipt = &rec
		}
	}

	resp := map[string]any{
		"memory_id":   m.ID,
		"leaf_hash":   m.LeafHash,
		"payload":     req.Payload,
		"inserted_at": m.CreatedAt,
		"storage": map[string]any{
			"backend":   "sqlite",
			"record_id": string(m.ID),
		},
		"status": "recorded_not_sealed",
	}
	if ledgerReceipt != nil {
		resp["ledger"] = map[string]any{
			"backend":     "ndjson_hash_chain",
			"seq":         ledgerReceipt.Seq,
			"event_hash":  ledgerReceipt.EventHash,
			"ledger_head": ledgerReceipt.Head,
		}
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleRecall(w http.ResponseWriter, r *http.Request) {
	id := domain.MemoryID(r.PathValue("id"))
	m, err := s.store.Recall(r.Context(), id)
	if err != nil {
		if err == domain.ErrMemoryNotFound {
			writeError(w, http.StatusNotFound, "memory not found")
			return
		}
		log.Printf("ERROR recalling memory: %v", err)
		writeError(w, http.StatusInternalServerError, "recall failed")
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// ---- Checkpoints ----

func (s *Server) handleAnchorBeacon(w http.ResponseWriter, r *http.Request) {
	memories, err := s.store.MemoriesByBeacon(r.Context(), domain.BeaconID("current"))
	if err != nil {
		log.Printf("ERROR listing memories: %v", err)
		writeError(w, http.StatusInternalServerError, "listing failed")
		return
	}
	if len(memories) == 0 {
		writeError(w, http.StatusBadRequest, "no memories to anchor")
		return
	}

	leaves := make([]string, len(memories))
	for i, mem := range memories {
		payloadJSON, _ := json.Marshal(mem.Payload)
		leaves[i] = string(payloadJSON)
	}

	root, err := s.tree.CreateRoot(r.Context(), leaves)
	if err != nil {
		log.Printf("ERROR creating merkle root: %v", err)
		writeError(w, http.StatusInternalServerError, "merkle root failed")
		return
	}

	parentBeaconID := ""
	if latest, err := s.store.LatestBeacon(r.Context()); err == nil {
		parentBeaconID = string(latest.ID)
	}

	now := time.Now().UTC()
	beaconID := storage.NewBeaconID()
	beacon := &domain.Beacon{
		ID:             beaconID,
		Root:           root,
		ParentBeaconID: parentBeaconID,
		ProofCount:     len(memories),
		CreatedAt:      now,
	}

	if err := s.store.AnchorBeacon(r.Context(), beacon); err != nil {
		log.Printf("ERROR anchoring beacon: %v", err)
		writeError(w, http.StatusInternalServerError, "anchoring failed")
		return
	}

	if err := s.store.UpdateBeaconID(r.Context(), "current", string(beaconID)); err != nil {
		log.Printf("ERROR updating beacon IDs: %v", err)
		writeError(w, http.StatusInternalServerError, "updating memories failed")
		return
	}

	// Append ledger event
	var ledgerReceipt *domain.LedgerReceipt
	if s.ledger != nil {
		rec, err := s.ledger.Append(r.Context(), domain.EventRootSealed, map[string]any{
			"beacon_id":  string(beaconID),
			"root":       root,
			"leaf_count": len(memories),
		})
		if err != nil {
			log.Printf("ERROR appending ledger: %v", err)
		} else {
			ledgerReceipt = &rec
		}
	}

	resp := map[string]any{
		"beacon_id":  beacon.ID,
		"root":       root,
		"leaf_count": len(memories),
		"storage": map[string]any{
			"backend":   "sqlite",
			"record_id": string(beacon.ID),
		},
		"proofs_created": len(memories),
	}
	if ledgerReceipt != nil {
		resp["ledger"] = map[string]any{
			"backend":    "ndjson_hash_chain",
			"seq":        ledgerReceipt.Seq,
			"event_hash": ledgerReceipt.EventHash,
		}
	}

	log.Printf("beacon %s anchored: root=%s, memories=%d", beaconID, root, len(memories))
	writeJSON(w, http.StatusCreated, resp)
}

// ---- Proofs ----

func (s *Server) handleGenerateRoute(w http.ResponseWriter, r *http.Request) {
	memoryID := r.PathValue("memory_id")
	m, err := s.store.Recall(r.Context(), domain.MemoryID(memoryID))
	if err != nil {
		writeError(w, http.StatusNotFound, "memory not found")
		return
	}

	memories, err := s.store.MemoriesByBeacon(r.Context(), domain.BeaconID(m.BeaconID))
	if err != nil {
		log.Printf("ERROR listing memories: %v", err)
		writeError(w, http.StatusInternalServerError, "listing failed")
		return
	}

	leaves := make([]string, len(memories))
	pos := -1
	for i, mem := range memories {
		payloadJSON, _ := json.Marshal(mem.Payload)
		leaves[i] = string(payloadJSON)
		if string(mem.ID) == memoryID {
			pos = i + 1
		}
	}

	if pos < 0 {
		writeError(w, http.StatusInternalServerError, "memory position not found")
		return
	}

	route, err := s.tree.GenerateRoute(r.Context(), leaves, pos)
	if err != nil {
		log.Printf("ERROR generating route: %v", err)
		writeError(w, http.StatusInternalServerError, "proof generation failed")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"leaf":       route.Leaf,
		"root":       route.Root,
		"path":       route.Path,
		"position":   pos,
		"leaf_count": len(leaves),
	})
}

func (s *Server) handleWitness(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Leaf      string   `json:"leaf"`
		Root      string   `json:"root"`
		Path      []string `json:"path"`
		Position  int      `json:"position"`
		LeafCount int      `json:"leaf_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	route := &domain.Route{
		Leaf: req.Leaf,
		Root: req.Root,
		Path: req.Path,
	}
	result, err := s.tree.Witness(r.Context(), route, req.Leaf, req.Position, req.LeafCount)
	if err != nil {
		log.Printf("ERROR verifying proof: %v", err)
		writeError(w, http.StatusInternalServerError, "verification failed")
		return
	}

	if s.ledger != nil {
		evtType := domain.EventVerifyOK
		if !result.Valid {
			evtType = domain.EventVerifyFailed
		}
		_, _ = s.ledger.Append(r.Context(), evtType, nil)
	}

	writeJSON(w, http.StatusOK, result)
}

// ---- Anchors ----

func (s *Server) handleCreateAnchor(w http.ResponseWriter, r *http.Request) {
	if s.anchor == nil {
		writeError(w, http.StatusServiceUnavailable, "anchor backend not configured")
		return
	}

	var req struct {
		AnchoredType string `json:"anchored_type"`
		AnchoredID   string `json:"anchored_id"`
		Hash         string `json:"hash"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Hash == "" {
		writeError(w, http.StatusBadRequest, "hash is required")
		return
	}
	if req.AnchoredType == "" {
		req.AnchoredType = "checkpoint"
	}

	receipt, err := s.anchor.Anchor(r.Context(), req.Hash, req.AnchoredType, req.AnchoredID)
	if err != nil {
		log.Printf("ERROR anchoring: %v", err)
		writeError(w, http.StatusInternalServerError, "anchoring failed")
		return
	}

	if s.ledger != nil {
		_, _ = s.ledger.Append(r.Context(), domain.EventAnchorCreated, map[string]any{
			"anchor_id":   receipt.AnchorID,
			"backend":     receipt.Backend,
			"hash":        req.Hash,
			"type":        req.AnchoredType,
			"anchored_id": req.AnchoredID,
		})
	}

	writeJSON(w, http.StatusCreated, receipt)
}

func (s *Server) handleGetAnchor(w http.ResponseWriter, r *http.Request) {
	if s.anchor == nil {
		writeError(w, http.StatusServiceUnavailable, "anchor backend not configured")
		return
	}
	// For local anchor, reconstruct info
	id := r.PathValue("id")
	writeJSON(w, http.StatusOK, map[string]any{
		"anchor_id": id,
		"backend":   s.anchor.Name(),
		"status":    "confirmed",
	})
}

// ---- Ledger ----

func (s *Server) handleLedgerEvents(w http.ResponseWriter, r *http.Request) {
	if s.ledger == nil {
		writeError(w, http.StatusServiceUnavailable, "ledger backend not configured")
		return
	}

	opts := domain.LedgerListOptions{Limit: 100}
	events, err := s.ledger.ListEvents(r.Context(), opts)
	if err != nil {
		log.Printf("ERROR listing ledger events: %v", err)
		writeError(w, http.StatusInternalServerError, "listing events failed")
		return
	}
	if events == nil {
		events = []domain.LedgerEvent{}
	}

	head, _ := s.ledger.LatestHead(r.Context())

	writeJSON(w, http.StatusOK, map[string]any{
		"events":      events,
		"total":       len(events),
		"ledger_head": head,
	})
}

func (s *Server) handleLedgerHead(w http.ResponseWriter, r *http.Request) {
	if s.ledger == nil {
		writeError(w, http.StatusServiceUnavailable, "ledger backend not configured")
		return
	}

	head, err := s.ledger.LatestHead(r.Context())
	if err != nil {
		log.Printf("ERROR getting ledger head: %v", err)
		writeError(w, http.StatusInternalServerError, "getting head failed")
		return
	}
	writeJSON(w, http.StatusOK, head)
}

func (s *Server) handleLedgerVerify(w http.ResponseWriter, r *http.Request) {
	if s.ledger == nil {
		writeError(w, http.StatusServiceUnavailable, "ledger backend not configured")
		return
	}

	verification, err := s.ledger.Verify(r.Context())
	if err != nil {
		log.Printf("ERROR verifying ledger: %v", err)
		writeError(w, http.StatusInternalServerError, "verification failed")
		return
	}
	writeJSON(w, http.StatusOK, verification)
}

// ---- Beacons ----

func (s *Server) handleGetBeacon(w http.ResponseWriter, r *http.Request) {
	id := domain.BeaconID(r.PathValue("id"))
	beacon, err := s.store.BeaconByID(r.Context(), id)
	if err != nil {
		if err == domain.ErrBeaconNotFound {
			writeError(w, http.StatusNotFound, "beacon not found")
			return
		}
		log.Printf("ERROR getting beacon: %v", err)
		writeError(w, http.StatusInternalServerError, "retrieving beacon failed")
		return
	}
	writeJSON(w, http.StatusOK, beacon)
}

func (s *Server) handleBeaconMemories(w http.ResponseWriter, r *http.Request) {
	id := domain.BeaconID(r.PathValue("id"))
	memories, err := s.store.MemoriesByBeacon(r.Context(), id)
	if err != nil {
		log.Printf("ERROR listing beacon memories: %v", err)
		writeError(w, http.StatusInternalServerError, "listing memories failed")
		return
	}
	if memories == nil {
		memories = []*domain.Memory{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"beacon_id": string(id),
		"memories":  memories,
	})
}

func (s *Server) handleExtendBeacon(w http.ResponseWriter, r *http.Request) {
	parentID := domain.BeaconID(r.PathValue("id"))

	_, err := s.store.BeaconByID(r.Context(), parentID)
	if err != nil {
		if err == domain.ErrBeaconNotFound {
			writeError(w, http.StatusNotFound, "parent beacon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "looking up parent failed")
		return
	}

	var req struct {
		Payload any `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if req.Payload == nil {
		writeError(w, http.StatusBadRequest, "payload is required")
		return
	}

	existing, err := s.store.MemoriesByBeacon(r.Context(), parentID)
	if err != nil {
		log.Printf("ERROR listing parent memories: %v", err)
		writeError(w, http.StatusInternalServerError, "listing parent memories failed")
		return
	}

	leaves := make([]string, 0, len(existing)+1)
	for _, mem := range existing {
		payloadJSON, _ := json.Marshal(mem.Payload)
		leaves = append(leaves, string(payloadJSON))
	}
	newPayloadJSON, _ := json.Marshal(req.Payload)
	leaves = append(leaves, string(newPayloadJSON))

	root, err := s.tree.CreateRoot(r.Context(), leaves)
	if err != nil {
		log.Printf("ERROR creating merkle root: %v", err)
		writeError(w, http.StatusInternalServerError, "merkle root failed")
		return
	}

	now := time.Now().UTC()
	childID := storage.NewBeaconID()
	child := &domain.Beacon{
		ID:             childID,
		Root:           root,
		ParentBeaconID: string(parentID),
		ProofCount:     len(leaves),
		CreatedAt:      now,
	}

	if err := s.store.AnchorBeacon(r.Context(), child); err != nil {
		log.Printf("ERROR anchoring child beacon: %v", err)
		writeError(w, http.StatusInternalServerError, "anchoring failed")
		return
	}

	if len(existing) > 0 {
		if err := s.store.UpdateBeaconID(r.Context(), string(parentID), string(childID)); err != nil {
			log.Printf("ERROR reassigning memories: %v", err)
			writeError(w, http.StatusInternalServerError, "reassigning memories failed")
			return
		}
	}

	hashResult, err := s.tree.HashPayload(r.Context(), string(newPayloadJSON))
	if err != nil {
		log.Printf("ERROR hashing payload: %v", err)
		writeError(w, http.StatusInternalServerError, "hashing failed")
		return
	}

	newMem, err := s.store.Remember(r.Context(), req.Payload, hashResult, string(childID))
	if err != nil {
		log.Printf("ERROR storing new memory: %v", err)
		writeError(w, http.StatusInternalServerError, "storing memory failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"beacon":  child,
		"memory":  newMem,
		"extends": string(parentID),
		"leaves":  len(leaves),
		"root":    root,
	})
}

// ---- Receipts ----

func (s *Server) handleReceiptExport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	exp := receipts.NewExporter(s.store, s.tree, s.ledger, s.anchor)
	receipt, err := exp.ExportMemory(r.Context(), id)
	if err != nil {
		log.Printf("ERROR exporting receipt: %v", err)
		writeError(w, http.StatusInternalServerError, "receipt export failed")
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}

// ---- Full Verification ----

func (s *Server) handleFullVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MemoryID string `json:"memory_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.MemoryID == "" {
		writeError(w, http.StatusBadRequest, "memory_id is required")
		return
	}

	v := verifier.New(s.store, s.tree, s.ledger, s.anchor)
	result, err := v.VerifyMemory(r.Context(), req.MemoryID)
	if err != nil {
		log.Printf("ERROR verifying memory: %v", err)
		writeError(w, http.StatusInternalServerError, "verification failed")
		return
	}

	if s.ledger != nil {
		evtType := domain.EventVerifyOK
		if result.Status != "valid" {
			evtType = domain.EventVerifyFailed
		}
		_, _ = s.ledger.Append(r.Context(), evtType, nil)
	}

	writeJSON(w, http.StatusOK, result)
}

// ---- Contracts ----

func (s *Server) handleListContracts(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(s.contractsDir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "cannot list contracts")
		return
	}

	contracts := make([]ContractInfo, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		ext := filepath.Ext(e.Name())
		lang := "zencode"
		if ext == ".lua" {
			lang = "lua"
		}
		contracts = append(contracts, ContractInfo{
			Name:     e.Name(),
			Size:     info.Size(),
			Language: lang,
		})
	}
	sort.Slice(contracts, func(i, j int) bool { return contracts[i].Name < contracts[j].Name })

	writeJSON(w, http.StatusOK, map[string]any{
		"contracts": contracts,
		"directory": s.contractsDir,
	})
}

func (s *Server) handleGetContract(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if strings.Contains(name, "..") || strings.Contains(name, "/") {
		writeError(w, http.StatusBadRequest, "invalid contract name")
		return
	}

	path := filepath.Join(s.contractsDir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		writeError(w, http.StatusNotFound, "contract not found")
		return
	}

	contentType := "text/plain; charset=utf-8"
	switch ext := filepath.Ext(name); ext {
	case ".lua":
		contentType = "text/x-lua; charset=utf-8"
	case ".zen":
		contentType = "text/plain; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)
	if _, err := w.Write(data); err != nil {
		log.Printf("ERROR writing contract response: %v", err)
	}
}

// ---- Helpers ----

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("ERROR writing JSON response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ContractInfo describes a single Zenroom contract.
type ContractInfo struct {
	Name     string `json:"name"`
	Size     int64  `json:"size"`
	Language string `json:"language"`
}
