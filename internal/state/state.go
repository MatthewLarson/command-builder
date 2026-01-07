package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State represents the current persistence state of the command builder.
type State struct {
	// CommandParts is the list of tokens currently built.
	// e.g., ["git", "commit", "-m"]
	CommandParts []string `json:"command_parts"`
}

// Manager handles saving and loading state.
type Manager struct {
	configDir string
	stateFile string
}

// NewManager creates a new state manager.
// It ensures the config directory exists.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(home, ".config", "command-builder")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	return &Manager{
		configDir: configDir,
		stateFile: filepath.Join(configDir, "state.json"),
	}, nil
}

// Load reads the state from disk.
func (m *Manager) Load() (*State, error) {
	data, err := os.ReadFile(m.stateFile)
	if os.IsNotExist(err) {
		return &State{CommandParts: []string{}}, nil
	}
	if err != nil {
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Save writes the state to disk.
func (m *Manager) Save(s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.stateFile, data, 0644)
}

// Clear resets the state.
func (m *Manager) Clear() error {
	return m.Save(&State{CommandParts: []string{}})
}
