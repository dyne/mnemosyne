package zenroom

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Executor runs Zencode contracts via the Zenroom VM subprocess.
// It is the sole cryptographic boundary in the application.
// No hashing, signing, or proof logic lives outside this package.
type Executor struct {
	bin string
}

// NewExecutor returns an Executor that invokes zenroom at the given binary path.
func NewExecutor(zenroomBin string) *Executor {
	return &Executor{bin: zenroomBin}
}

// Run executes a Zencode contract script with the given keys and data.
// Uses the -z flag to enable Zencode mode.
func (e *Executor) Run(script, keys, data []byte) (*Result, error) {
	return e.exec(true, script, keys, data)
}

// RunLua executes a Lua script with the given keys and data.
// Runs without the -z flag (raw Lua mode).
func (e *Executor) RunLua(script, keys, data []byte) (*Result, error) {
	return e.exec(false, script, keys, data)
}

func (e *Executor) exec(zencode bool, script, keys, data []byte) (*Result, error) {
	args := make([]string, 0, 6)
	if zencode {
		args = append(args, "-z")
	}

	var cleanups []func()
	defer func() {
		for _, c := range cleanups {
			c()
		}
	}()

	if len(keys) > 0 {
		f, err := writeTemp("mnemosyne-keys-*.json", keys)
		if err != nil {
			return nil, err
		}
		cleanups = append(cleanups, func() { _ = os.Remove(f) })
		args = append(args, "-k", f)
	}
	if len(data) > 0 {
		f, err := writeTemp("mnemosyne-data-*.json", data)
		if err != nil {
			return nil, err
		}
		cleanups = append(cleanups, func() { _ = os.Remove(f) })
		args = append(args, "-a", f)
	}

	cmd := exec.Command(e.bin, args...)
	cmd.Stdin = bytes.NewReader(script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("zenroom execution failed: %w\nstderr: %s", err, stderr.String())
	}

	output := stdout.Bytes()
	if len(output) == 0 {
		return nil, fmt.Errorf("zenroom produced no output")
	}

	var result Result
	if err := json.Unmarshal(output, &result.Output); err != nil {
		result.Output = string(output)
	}
	result.Raw = output
	result.Logs = stderr.String()
	return &result, nil
}

func writeTemp(pattern string, content []byte) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("temp file: %w", err)
	}
	if _, err := f.Write(content); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// Result holds the output from a Zenroom execution.
type Result struct {
	Output any    `json:"output"`
	Raw    []byte `json:"-"`
	Logs   string `json:"-"`
}

// OutputString returns the raw output as a string.
func (r *Result) OutputString() string {
	return string(r.Raw)
}

// OutputMap returns the parsed output as a map, or an error if it is not a map.
func (r *Result) OutputMap() (map[string]any, error) {
	m, ok := r.Output.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("zenroom output is not a JSON object")
	}
	return m, nil
}
