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

	"github.com/dyne/mnemosyne/internal/api"
	"github.com/dyne/mnemosyne/internal/merkle"
	"github.com/dyne/mnemosyne/internal/storage"
	"github.com/dyne/mnemosyne/internal/zenroom"
)

// version is set at build time via -ldflags="-X main.version=..."
var version = "dev"

const banner = `
╔══════════════════════════════════════════════════╗
║                                                  ║
║       ███╗   ███╗███╗   ██╗███████╗███╗   ███╗  ║
║       ████╗ ████║████╗  ██║██╔════╝████╗ ████║  ║
║       ██╔████╔██║██╔██╗ ██║█████╗  ██╔████╔██║  ║
║       ██║╚██╔╝██║██║╚██╗██║██╔══╝  ██║╚██╔╝██║  ║
║       ██║ ╚═╝ ██║██║ ╚████║███████╗██║ ╚═╝ ██║  ║
║       ╚═╝     ╚═╝╚═╝  ╚═══╝╚══════╝╚═╝     ╚═╝  ║
║                                                  ║
║       MNEMOSYNE — titaness of memory              ║
║       cryptographic memory archive                 ║
║       verifiable append-only truth                 ║
║       dyne.org                                     ║
║                                                  ║
╚══════════════════════════════════════════════════╝
`

func main() {
	fmt.Print(banner)
	fmt.Printf("version %s\n\n", version)
	if err := run(os.Environ()); err != nil {
		log.Fatal(err)
	}
	log.Println("server stopped")
}

type config struct {
	contractsDir string
	dbPath       string
	zenroomBin   string
	addr         string
	webDir       string
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
		dbPath:       lookup("MNEMOSYNE_DB", filepath.Join(os.TempDir(), "mnemosyne.db")),
		zenroomBin:   lookup("ZENROOM_BIN", "zenroom"),
		addr:         lookup("MNEMOSYNE_ADDR", ":8080"),
		webDir:       lookup("MNEMOSYNE_WEB", "web"),
	}
}

func setupServer(cfg config) (http.Handler, *storage.SQLiteStore, error) {
	store, err := storage.NewSQLiteStore(cfg.dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("storage: %w", err)
	}

	executor := zenroom.NewExecutor(cfg.zenroomBin)
	tree := merkle.NewTree(executor, store, cfg.contractsDir)
	server := api.NewServer(store, tree, cfg.webDir, cfg.contractsDir, version)
	handler := corsMiddleware(server.Handler())

	return handler, store, nil
}

func run(environ []string) error {
	cfg := configFromEnv(environ)

	handler, store, err := setupServer(cfg)
	if err != nil {
		return err
	}
	defer store.Close()

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
		srv.Shutdown(ctx)
	}()

	fmt.Printf("mnemosyne listening on %s\n", cfg.addr)
	fmt.Printf("  database: %s\n", cfg.dbPath)
	fmt.Printf("  contracts: %s\n", cfg.contractsDir)

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
