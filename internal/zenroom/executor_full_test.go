package zenroom

import (
	"os"
	"testing"
)

// ---- Run (Zencode) tests ----

func TestRun_InvalidScript(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	e := NewExecutor(bin)
	_, err := e.Run([]byte("not valid zencode at all"), nil, nil)
	if err == nil {
		t.Error("expected error for invalid script")
	}
}

func TestRun_WithKeys(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	e := NewExecutor(bin)
	script := []byte(`rule unknown ignore
Given I have a 'string' named 'input'
When I create the hash of 'input'
Then print the 'hash'`)
	keys := []byte(`{"key":"value"}`)
	data := []byte(`{"input":"hello"}`)
	result, err := e.Run(script, keys, data)
	if err != nil {
		t.Fatalf("Run with keys: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
}

func TestRun_MultipleHashes(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	e := NewExecutor(bin)
	script := []byte(`rule unknown ignore
Given I have a 'string array' named 'inputs'
When I create the key derivations of each object in 'inputs'
Then print the 'key derivations'`)
	data := []byte(`{"inputs":["a","b","c"]}`)
	result, err := e.Run(script, nil, data)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	m, err := result.OutputMap()
	if err != nil {
		t.Fatalf("OutputMap: %v", err)
	}
	if _, ok := m["key_derivations"]; !ok {
		t.Error("expected key_derivations in output")
	}
}

// ---- RunLua tests ----

func TestRunLua(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	e := NewExecutor(bin)
	script := []byte(`print(JSON.encode({status="ok", value=42}))`)
	result, err := e.RunLua(script, nil, nil)
	if err != nil {
		t.Fatalf("RunLua: %v", err)
	}
	m, err := result.OutputMap()
	if err != nil {
		t.Fatalf("OutputMap: %v", err)
	}
	if m["status"] != "ok" {
		t.Errorf("expected ok, got %v", m["status"])
	}
}

func TestRunLua_WithData(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	e := NewExecutor(bin)
	script := []byte(`local d = JSON.decode(DATA); d.from_lua = true; print(JSON.encode(d))`)
	data := []byte(`{"hello":"world"}`)
	result, err := e.RunLua(script, nil, data)
	if err != nil {
		t.Fatalf("RunLua: %v", err)
	}
	m, err := result.OutputMap()
	if err != nil {
		t.Fatalf("OutputMap: %v", err)
	}
	if m["from_lua"] != true {
		t.Error("expected from_lua flag")
	}
}

func TestRunLua_WithKeys(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	e := NewExecutor(bin)
	script := []byte(`print(JSON.encode({got_keys=KEYS ~= nil and #KEYS > 0}))`)
	keys := []byte(`{"secret":"123"}`)
	result, err := e.RunLua(script, keys, nil)
	if err != nil {
		t.Fatalf("RunLua: %v", err)
	}
	t.Logf("output: %s", result.OutputString())
}

func TestRunLua_EmptyOutput(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	e := NewExecutor(bin)
	script := []byte(``)
	_, err := e.RunLua(script, nil, nil)
	if err == nil {
		t.Error("expected error for empty output")
	}
}

// ---- Result tests ----

func TestResult_OutputMap_NotObject(t *testing.T) {
	r := &Result{Output: "just a string", Raw: []byte(`"just a string"`)}
	_, err := r.OutputMap()
	if err == nil {
		t.Error("expected error for non-object output")
	}
}

func TestResult_OutputString(t *testing.T) {
	r := &Result{Raw: []byte("hello")}
	if r.OutputString() != "hello" {
		t.Error("OutputString mismatch")
	}
}

// ---- writeTemp tests ----

func TestWriteTemp(t *testing.T) {
	path, err := writeTemp("mnemosyne-test-*.json", []byte(`{"test":true}`))
	if err != nil {
		t.Fatalf("writeTemp: %v", err)
	}
	defer os.Remove(path)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("temp file does not exist")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read temp: %v", err)
	}
	if string(data) != `{"test":true}` {
		t.Error("content mismatch")
	}
}

// ---- LoadContract tests ----

func TestLoadContract_NotFound(t *testing.T) {
	_, err := LoadContract("/nonexistent/path/to/contract.zen")
	if err == nil {
		t.Error("expected error for missing contract")
	}
}
