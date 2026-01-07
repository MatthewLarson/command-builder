package scraper

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"

	"os"
	"regexp"
	"strings"

	"command-builder/internal/definitions"

	"gopkg.in/yaml.v3"
)

// Scraper handles generating definitions from help text.
type Scraper struct {
	defDir string
}

func NewScraper() (*Scraper, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Scraper{
		defDir: filepath.Join(home, ".config", "command-builder", "definitions"),
	}, nil
}

// Scrape attempts to run `command --help` and parse it into a basic definition.
func (s *Scraper) Scrape(commandName string) (*definitions.CommandDefinition, error) {
	// 1. Run command --help
	cmd := exec.Command(commandName, "--help")
	var out bytes.Buffer
	cmd.Stdout = &out
	// Some commands print help to stderr
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run %s --help: %w", commandName, err)
	}

	output := out.String()

	// 2. Parse the output
	def := parseHelpOutput(commandName, output)

	// 3. Save to file
	if err := s.saveDefinition(def); err != nil {
		return nil, err
	}

	return def, nil
}

func parseHelpOutput(name, text string) *definitions.CommandDefinition {
	def := &definitions.CommandDefinition{
		Name:        name,
		Description: "Auto-generated from --help",
		Flags:       []definitions.Flag{},
	}

	scanner := bufio.NewScanner(strings.NewReader(text))
	// Naive regex for flags: "  -f, --flag    Description"
	// This is very specific to standard GNU style help
	flagRegex := regexp.MustCompile(`^\s+(-[a-zA-Z0-9],? )?(--[a-zA-Z0-9-]+)\s+(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := flagRegex.FindStringSubmatch(line)
		if len(matches) > 3 {
			// matches[2] is --flag
			// matches[3] is description
			f := definitions.Flag{
				Name:        matches[2],
				Description: matches[3],
				Type:        "string", // Default assumption
			}
			def.Flags = append(def.Flags, f)
		}
	}

	return def
}

func (s *Scraper) saveDefinition(def *definitions.CommandDefinition) error {
	data, err := yaml.Marshal(def)
	if err != nil {
		return err
	}
	filename := filepath.Join(s.defDir, def.Name+".yaml")
	return os.WriteFile(filename, data, 0644)
}
