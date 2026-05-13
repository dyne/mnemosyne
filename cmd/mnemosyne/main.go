package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/dyne/mnemosyne/internal/anchor/local"
	"github.com/dyne/mnemosyne/internal/api"
	"github.com/dyne/mnemosyne/internal/ledger/ndjson"
	"github.com/dyne/mnemosyne/internal/merkle"
	"github.com/dyne/mnemosyne/internal/storage"
	"github.com/dyne/mnemosyne/internal/zenroom"
)

// version is set at build time via -ldflags="-X main.version=..."
var version = "dev"

const banner = `
╔══════════════════════════════════════════════════╗
║                                                  ║
║       ███╗   ███╗███╗   ██╗███████╗███╗   ███╗   ║
║       ████╗ ████║████╗  ██║██╔════╝████╗ ████║   ║
║       ██╔████╔██║██╔██╗ ██║█████╗  ██╔████╔██║   ║
║       ██║╚██╔╝██║██║╚██╗██║██╔══╝  ██║╚██╔╝██║   ║
║       ██║ ╚═╝ ██║██║ ╚████║███████╗██║ ╚═╝ ██║   ║
║       ╚═╝     ╚═╝╚═╝  ╚═══╝╚══════╝╚═╝     ╚═╝   ║
║                                                  ║
║       MNEMOSYNE — titaness of memory             ║
║       cryptographic memory archive               ║
║       verifiable append-only truth               ║
║       dyne.org                                   ║
║                                                  ║
╚══════════════════════════════════════════════════╝
`

func main() {
	fmt.Print(banner)
	fmt.Printf("version %s\n\n", version)

	// Check for CLI subcommands
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "serve":
			// default — handled below
		case "help", "-h", "--help":
			printUsage()
			return
		case "version", "-v", "--version":
			fmt.Printf("mnemosyne version %s\n", version)
			return
		default:
			fmt.Printf("unknown command: %s\n", args[0])
			printUsage()
			os.Exit(1)
		}
	}

	if err := run(os.Environ()); err != nil {
		log.Fatal(err)
	}
	log.Println("server stopped")
}

func printUsage() {
	fmt.Print(`
Usage: mnemosyne [command]

Commands:
  serve            Start the HTTP API server (default)
  help             Show this help

Environment:
  MNEMOSYNE_ADDR        Address to listen on (default :8080)
  MNEMOSYNE_DATA_DIR    Data directory (default data/)
  MNEMOSYNE_DB          SQLite database path
  MNEMOSYNE_CONTRACTS   Zenroom contracts directory
  MNEMOSYNE_WEB         Web UI directory
  MNEMOSYNE_KEY_REF     Key reference for signing
  ZENROOM_BIN           Path to zenroom binary (default zenroom)

API endpoints:
  POST /memories           Remember a memory
  GET  /memories/{id}      Recall a memory
  POST /checkpoints        Seal memories into a Merkle root
  GET  /proofs/{id}        Generate inclusion proof
  POST /verify             Verify a Merkle proof
  POST /verify/full        Full trust-chain verification
  POST /anchors            Create an anchor
  GET  /ledger/events      Browse ledger events
  POST /ledger/verify      Verify ledger chain integrity
  GET  /dashboard          Dashboard status
  GET  /docs               API documentation UI
  GET  /                   Web UI
`)
}

type config struct {
	contractsDir string
	dataDir      string
	dbPath       string
	zenroomBin   string
	addr         string
	webDir       string
	ledgerKeyRef string
}

func configFromEnv(environ []string) config {
	lookup := func(key, def string) string {
		prefix := key + "="
		for _, e := range environ {
			if len(e) > len(prefix) && e[:len(prefix)] == prefix {
				return e[len(prefix):]
			}
		}
		return def
	}
	return config{
		contractsDir: lookup("MNEMOSYNE_CONTRACTS", "zenflows"),
		dataDir:      lookup("MNEMOSYNE_DATA_DIR", "data"),
		dbPath:       lookup("MNEMOSYNE_DB", filepath.Join("data", "mnemosyne.db")),
		zenroomBin:   lookup("ZENROOM_BIN", "zenroom"),
		addr:         lookup("MNEMOSYNE_ADDR", ":8080"),
		webDir:       lookup("MNEMOSYNE_WEB", "web"),
		ledgerKeyRef: lookup("MNEMOSYNE_KEY_REF", "mnemosyne-local"),
	}
}

func setupServer(cfg config) (http.Handler, *storage.SQLiteStore, error) {
	executor := zenroom.NewExecutor(cfg.zenroomBin)

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.dataDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("data dir: %w", err)
	}

	// Storage
	store, err := storage.NewSQLiteStore(cfg.dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("storage: %w", err)
	}

	// Merkle tree
	tree := merkle.NewTree(executor, store, cfg.contractsDir)

	// Ledger
	ledgerPath := filepath.Join(cfg.dataDir, "ledger.ndjson")
	ledger, err := ndjson.New(ledgerPath, cfg.contractsDir, cfg.ledgerKeyRef, executor)
	if err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("ledger: %w", err)
	}

	// Anchor
	anchor, err := local.New(cfg.contractsDir, cfg.ledgerKeyRef, executor)
	if err != nil {
		store.Close()
		return nil, nil, fmt.Errorf("anchor: %w", err)
	}

	server := api.NewServer(api.ServerConfig{
		Store:        store,
		Tree:         tree,
		Ledger:       ledger,
		Anchor:       anchor,
		WebDir:       cfg.webDir,
		ContractsDir: cfg.contractsDir,
		Version:      version,
	})

	handler := corsMiddleware(server.Handler())
	return handler, store, nil
}

func run(environ []string) error {
	cfg := configFromEnv(environ)

	handler, store, err := setupServer(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("ERROR closing storage: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:         cfg.addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("ERROR shutting down server: %v", err)
		}
	}()

	fmt.Printf("mnemosyne listening on %s\n", cfg.addr)
	fmt.Printf("  database:  %s\n", cfg.dbPath)
	fmt.Printf("  ledger:    %s\n", filepath.Join(cfg.dataDir, "ledger.ndjson"))
	fmt.Printf("  contracts: %s\n", cfg.contractsDir)
	fmt.Printf("  web:       %s\n", cfg.webDir)
	fmt.Printf("  anchor:    local_signature\n")

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
