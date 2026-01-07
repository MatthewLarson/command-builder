package definitions

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Flag represents a command-line flag/option.
type Flag struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Type        string `yaml:"type,omitempty"` // e.g., "string", "bool"
}

// Subcommand represents a nested command.
type Subcommand struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Subcommands []Subcommand `yaml:"subcommands,omitempty"`
	Flags       []Flag       `yaml:"flags,omitempty"`
}

// CommandDefinition represents the top-level command definition.
type CommandDefinition struct {
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Subcommands []Subcommand `yaml:"subcommands,omitempty"`
	Flags       []Flag       `yaml:"flags,omitempty"`
}

// Manager handles loading definitions.
type Manager struct {
	defDir string
}

// NewManager creates a new definitions manager.
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	defDir := filepath.Join(home, ".config", "command-builder", "definitions")
	if err := os.MkdirAll(defDir, 0755); err != nil {
		return nil, err
	}

	return &Manager{
		defDir: defDir,
	}, nil
}

// LoadDefinition attempts to load a YAML definition for the given command name.
// It looks in the local definition directory.
func (m *Manager) LoadDefinition(commandName string) (*CommandDefinition, error) {
	filename := filepath.Join(m.defDir, commandName+".yaml")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var def CommandDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// ListDefinitions returns a list of available command definitions in the local cache.
func (m *Manager) ListDefinitions() ([]string, error) {
	entries, err := os.ReadDir(m.defDir)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".yaml" {
			names = append(names, entry.Name()[:len(entry.Name())-5])
		}
	}
	return names, nil
}

// FindSubcommand traverses the definition based on the path of subcommands.
func (def *CommandDefinition) FindSubcommand(path []string) *Subcommand {
	if len(path) == 0 {
		return nil
	}

	var current *Subcommand
	// Find the first part in top-level subcommands
	for i := range def.Subcommands {
		if def.Subcommands[i].Name == path[0] {
			current = &def.Subcommands[i]
			break
		}
	}

	if current == nil {
		return nil
	}

	// Traverse the rest
	for _, part := range path[1:] {
		found := false
		for i := range current.Subcommands {
			if current.Subcommands[i].Name == part {
				current = &current.Subcommands[i]
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	return current
}
