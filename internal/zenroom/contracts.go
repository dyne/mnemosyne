package zenroom

import (
	"fmt"
	"os"
	"path/filepath"
)

// Contract is a versioned, named Zencode contract.
type Contract struct {
	Name    string
	Version string
	Script  []byte
}

// LoadContract reads a .zen file from the filesystem and returns a Contract.
func LoadContract(path string) (*Contract, error) {
	script, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("contract %s not found: %w", path, err)
	}
	name := filepath.Base(path)
	return &Contract{
		Name:    name,
		Version: "1.0.0",
		Script:  script,
	}, nil
}
