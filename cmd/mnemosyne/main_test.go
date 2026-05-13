package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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
	bin, err := exec.LookPath("zenroom")
	if err != nil {
		t.Skip("zenroom not found")
	}
	tmpDir := t.TempDir()

	// Copy real contracts from the repo
	contractsDir := tmpDir + "/contracts"
	if err := os.MkdirAll(contractsDir, 0755); err != nil {
		t.Fatalf("create contracts dir: %v", err)
	}
	for _, name := range []string{"hash.zen", "keygen.zen", "sign.zen", "verify_signature.zen"} {
		src := "../../zenflows/" + name
		if _, err := os.Stat(src); err == nil {
			data, err := os.ReadFile(src)
			if err != nil {
				t.Fatalf("read contract %s: %v", name, err)
			}
			if err := os.WriteFile(filepath.Join(contractsDir, name), data, 0644); err != nil {
				t.Fatalf("write contract %s: %v", name, err)
			}
		}
	}

	if err := os.MkdirAll(tmpDir+"/web/static", 0755); err != nil {
		t.Fatalf("create web dir: %v", err)
	}
	if err := os.WriteFile(tmpDir+"/web/index.html", []byte("<html></html>"), 0644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	handler, store, err := setupServer(config{
		dataDir:      tmpDir + "/data",
		dbPath:       tmpDir + "/data/test.db",
		contractsDir: contractsDir,
		webDir:       tmpDir + "/web",
		zenroomBin:   bin,
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
	// Run in a subprocess to avoid SIGTERM killing the test runner
	if os.Getenv("MNEMOSYNE_TEST_SHUTDOWN") == "1" {
		bin, err := exec.LookPath("zenroom")
		if err != nil {
			t.Skip("zenroom not found")
		}
		tmpDir := os.Getenv("MNEMOSYNE_TEST_DIR")
		contractsDir := tmpDir + "/contracts"
		webDir := tmpDir + "/web"
		if err := run([]string{
			"ZENROOM_BIN=" + bin,
			"MNEMOSYNE_ADDR=127.0.0.1:0",
			"MNEMOSYNE_DATA_DIR=" + tmpDir + "/data",
			"MNEMOSYNE_DB=" + tmpDir + "/data/run.db",
			"MNEMOSYNE_CONTRACTS=" + contractsDir,
			"MNEMOSYNE_WEB=" + webDir,
		}); err != nil {
			t.Error(err)
		}
		return
	}

	// Set up test directory
	tmpDir, err := os.MkdirTemp("", "mnemosyne-graceful-*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	contractsDir := tmpDir + "/contracts"
	_ = os.MkdirAll(contractsDir, 0755)
	for _, name := range []string{"hash.zen", "keygen.zen", "sign.zen", "verify_signature.zen"} {
		src := "../../zenflows/" + name
		if data, err := os.ReadFile(src); err == nil {
			_ = os.WriteFile(filepath.Join(contractsDir, name), data, 0644)
		}
	}
	webDir := tmpDir + "/web"
	_ = os.MkdirAll(webDir+"/static", 0755)
	_ = os.WriteFile(webDir+"/index.html", []byte("<html></html>"), 0644)

	cmd := exec.Command(os.Args[0], "-test.run=TestRun_GracefulShutdown", "-test.v")
	cmd.Env = append(os.Environ(), "MNEMOSYNE_TEST_SHUTDOWN=1", "MNEMOSYNE_TEST_DIR="+tmpDir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start subprocess: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	_ = cmd.Process.Signal(syscall.SIGTERM)

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Logf("stderr: %s", stderr.String())
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("subprocess did not shut down")
	}
}
