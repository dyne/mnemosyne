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

	"github.com/dyne/mnemosyne/internal/domain"
	"github.com/dyne/mnemosyne/internal/merkle"
	"github.com/dyne/mnemosyne/internal/storage"
)

// Server is the HTTP API server for Mnemosyne.
type Server struct {
	store        storage.Store
	tree         *merkle.Tree
	mux          *http.ServeMux
	contractsDir string
	version      string
}

// NewServer creates a new API server.
// If webDir is non-empty, static files from that directory are served at /static/ and /.
// contractsDir is the path to the Zenroom contracts directory.
func NewServer(store storage.Store, tree *merkle.Tree, webDir, contractsDir, version string) *Server {
	s := &Server{store: store, tree: tree, mux: http.NewServeMux(), contractsDir: contractsDir, version: version}
	s.routes()
	if webDir != "" {
		s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir(webDir+"/static"))))
		s.mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.ServeFile(w, r, webDir+"/index.html")
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

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"version": s.version,
		"project": "mnemosyne",
	})
}

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
	beaconID := "current" // Will be replaced with actual beacon on checkpoint
	m, err := s.store.Remember(r.Context(), req.Payload, hashResult, beaconID)
	if err != nil {
		log.Printf("ERROR storing memory: %v", err)
		writeError(w, http.StatusInternalServerError, "storage failed")
		return
	}

	writeJSON(w, http.StatusCreated, m)
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

func (s *Server) handleAnchorBeacon(w http.ResponseWriter, r *http.Request) {
	// Collect all memories with the default beacon ID
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

	// Build leaves from payloads
	leaves := make([]string, len(memories))
	for i, mem := range memories {
		payloadJSON, _ := json.Marshal(mem.Payload)
		leaves[i] = string(payloadJSON)
	}

	// Compute Merkle root via Zenroom
	root, err := s.tree.CreateRoot(r.Context(), leaves)
	if err != nil {
		log.Printf("ERROR creating merkle root: %v", err)
		writeError(w, http.StatusInternalServerError, "merkle root failed")
		return
	}

	// Find parent beacon (the latest one)
	parentBeaconID := ""
	if latest, err := s.store.LatestBeacon(r.Context()); err == nil {
		parentBeaconID = string(latest.ID)
	}

	// Create the beacon
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

	// Assign all memories to this beacon
	if err := s.store.UpdateBeaconID(r.Context(), "current", string(beaconID)); err != nil {
		log.Printf("ERROR updating beacon IDs: %v", err)
		writeError(w, http.StatusInternalServerError, "updating memories failed")
		return
	}

	log.Printf("beacon %s anchored: root=%s, memories=%d", beaconID, root, len(memories))
	writeJSON(w, http.StatusCreated, beacon)
}

func (s *Server) handleGenerateRoute(w http.ResponseWriter, r *http.Request) {
	memoryID := r.PathValue("memory_id")
	m, err := s.store.Recall(r.Context(), domain.MemoryID(memoryID))
	if err != nil {
		writeError(w, http.StatusNotFound, "memory not found")
		return
	}

	// Get all memories for this beacon
	memories, err := s.store.MemoriesByBeacon(r.Context(), domain.BeaconID(m.BeaconID))
	if err != nil {
		log.Printf("ERROR listing memories: %v", err)
		writeError(w, http.StatusInternalServerError, "listing failed")
		return
	}

	// Build leaves from memories
	leaves := make([]string, len(memories))
	pos := -1
	for i, mem := range memories {
		payloadJSON, _ := json.Marshal(mem.Payload)
		leaves[i] = string(payloadJSON)
		if string(mem.ID) == memoryID {
			pos = i + 1 // 1-indexed
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

	// Include position and leaf count for verification
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

	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
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

	ext := filepath.Ext(name)
	contentType := "text/plain; charset=utf-8"
	if ext == ".lua" {
		contentType = "text/x-lua; charset=utf-8"
	} else if ext == ".zen" {
		contentType = "text/plain; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

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

	// Verify parent exists
	_, err := s.store.BeaconByID(r.Context(), parentID)
	if err != nil {
		if err == domain.ErrBeaconNotFound {
			writeError(w, http.StatusNotFound, "parent beacon not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "looking up parent failed")
		return
	}

	// Parse new payload
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

	// Get existing leaves from parent beacon
	existing, err := s.store.MemoriesByBeacon(r.Context(), parentID)
	if err != nil {
		log.Printf("ERROR listing parent memories: %v", err)
		writeError(w, http.StatusInternalServerError, "listing parent memories failed")
		return
	}

	// Build the combined leaf set: existing leaves + new leaf
	leaves := make([]string, 0, len(existing)+1)
	for _, mem := range existing {
		payloadJSON, _ := json.Marshal(mem.Payload)
		leaves = append(leaves, string(payloadJSON))
	}
	newPayloadJSON, _ := json.Marshal(req.Payload)
	leaves = append(leaves, string(newPayloadJSON))

	// Compute new Merkle root via Zenroom
	root, err := s.tree.CreateRoot(r.Context(), leaves)
	if err != nil {
		log.Printf("ERROR creating merkle root: %v", err)
		writeError(w, http.StatusInternalServerError, "merkle root failed")
		return
	}

	// Create the child beacon
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

	// Re-assign parent's memories to the child beacon
	if len(existing) > 0 {
		if err := s.store.UpdateBeaconID(r.Context(), string(parentID), string(childID)); err != nil {
			log.Printf("ERROR reassigning memories: %v", err)
			writeError(w, http.StatusInternalServerError, "reassigning memories failed")
			return
		}
	}

	// Store the new memory with the child beacon ID
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
