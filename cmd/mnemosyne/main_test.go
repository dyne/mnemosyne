package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestBanner(t *testing.T) {
	if len(banner) < 100 {
		t.Error("banner should be substantial")
	}
}

func TestCORSMiddleware_AddsHeaders(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := corsMiddleware(next)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	handler.ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("missing CORS origin")
	}
	if w.Header().Get("Access-Control-Allow-Methods") != "GET, POST, OPTIONS" {
		t.Error("missing CORS methods")
	}
	if w.Header().Get("Access-Control-Allow-Headers") != "Content-Type" {
		t.Error("missing CORS headers")
	}
}

func TestCORSMiddleware_OPTIONS(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for OPTIONS")
	})
	handler := corsMiddleware(next)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("OPTIONS", "/test", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestCORSMiddleware_POST(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})
	handler := corsMiddleware(next)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/test", nil)
	handler.ServeHTTP(w, r)
	if !called {
		t.Error("handler should be called for POST")
	}
	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	cfg := configFromEnv([]string{})
	if cfg.contractsDir != "zenflows" {
		t.Errorf("expected zenflows, got %s", cfg.contractsDir)
	}
	if cfg.zenroomBin != "zenroom" {
		t.Errorf("expected zenroom, got %s", cfg.zenroomBin)
	}
	if cfg.addr != ":8080" {
		t.Errorf("expected :8080, got %s", cfg.addr)
	}
	if cfg.webDir != "web" {
		t.Errorf("expected web, got %s", cfg.webDir)
	}
}

func TestConfigFromEnv_Custom(t *testing.T) {
	cfg := configFromEnv([]string{
		"MNEMOSYNE_ADDR=:9999",
		"MNEMOSYNE_DB=/custom/path.db",
		"MNEMOSYNE_CONTRACTS=/custom/contracts",
	})
	if cfg.addr != ":9999" {
		t.Errorf("expected :9999, got %s", cfg.addr)
	}
	if cfg.dbPath != "/custom/path.db" {
		t.Errorf("expected /custom/path.db, got %s", cfg.dbPath)
	}
	if cfg.contractsDir != "/custom/contracts" {
		t.Errorf("expected /custom/contracts, got %s", cfg.contractsDir)
	}
}

func TestConfigFromEnv_Partial(t *testing.T) {
	cfg := configFromEnv([]string{"MNEMOSYNE_ADDR=:7777"})
	if cfg.addr != ":7777" {
		t.Errorf("expected :7777, got %s", cfg.addr)
	}
	if cfg.webDir != "web" {
		t.Errorf("default webDir should be 'web', got %s", cfg.webDir)
	}
}

func TestRun_InvalidDBPath(t *testing.T) {
	err := run([]string{"MNEMOSYNE_DB=/dev/null/nonexistent/db.db"})
	if err == nil {
		t.Error("expected error for invalid db path")
	}
}

func TestSetupServer_InvalidDB(t *testing.T) {
	_, _, err := setupServer(config{dbPath: "/dev/null/nonexistent/db.db"})
	if err == nil {
		t.Error("expected error for invalid db path")
	}
}

func TestSetupServer_Success(t *testing.T) {
	tmpDir := t.TempDir()
	contractsDir := tmpDir + "/contracts"
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("create contracts dir: %v", err)
	}
	if err := os.WriteFile(contractsDir+"/hash.zen", []byte("Scenario 'simple': hash\nGiven nothing\nWhen I create the random object of '256' bits\nThen print the 'random object'"), 0644); err != nil {
		t.Fatalf("write contract: %v", err)
	}
	if err := os.MkdirAll(tmpDir+"/web/static", 0755); err != nil {
		t.Fatalf("create web dir: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/web/index.html", []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	handler, store, err := setupServer(config{
		dbPath:       tmpDir + "/test.db",
		contractsDir: contractsDir,
		webDir:       tmpDir + "/web",
	})
	if err != nil {
		t.Fatalf("setupServer: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	}()
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestRun_GracefulShutdown(t *testing.T) {
	if _, err := exec.LookPath("zenroom"); err != nil {
		t.Skip("zenroom not found")
	}

	tmpDir := t.TempDir()
	contractsDir := tmpDir + "/contracts"
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("create contracts dir: %v", err)
	}
	webDir := tmpDir + "/web"
	if err := os.MkdirAll(webDir+"/static", 0755); err != nil {
		t.Fatalf("create web dir: %v", err)
	}
	if err := os.WriteFile(webDir+"/index.html", []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- run([]string{
			"MNEMOSYNE_ADDR=127.0.0.1:0",
			"MNEMOSYNE_DB=" + tmpDir + "/run.db",
			"MNEMOSYNE_CONTRACTS=" + contractsDir,
			"MNEMOSYNE_WEB=" + webDir,
		})
	}()

	time.Sleep(50 * time.Millisecond)
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("send signal: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not shut down")
	}
}
