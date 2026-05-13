package zenroom

import (
	"os/exec"
	"testing"
)

func zenroomBin() string {
	for _, p := range []string{"zenroom", "/usr/bin/zenroom", "/usr/local/bin/zenroom"} {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return ""
}

func TestNewExecutor(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}
	e := NewExecutor(bin)
	if e.bin != bin {
		t.Errorf("expected bin %s, got %s", bin, e.bin)
	}
}

func TestRunHashing(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}
	e := NewExecutor(bin)

	script := []byte(`rule unknown ignore
Given I have a 'string' named 'input'
When I create the hash of 'input'
Then print the 'hash'`)

	data := []byte(`{"input":"hello world"}`)

	result, err := e.Run(script, nil, data)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Raw) == 0 {
		t.Error("expected non-empty raw output")
	}
	t.Logf("output: %s", result.OutputString())
}

func TestRunWithoutData(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}
	e := NewExecutor(bin)

	script := []byte(`rule unknown ignore
Given nothing
When I create the random object of '256' bits
Then print the 'random object'`)

	result, err := e.Run(script, nil, nil)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	t.Logf("output: %s", result.OutputString())
}
