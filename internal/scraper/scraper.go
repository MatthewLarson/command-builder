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
	// 1. Run root command --help
	out, err := runHelp(commandName)
	if err != nil {
		return nil, err
	}

	// 2. Parse the output
	def := parseHelpOutput(commandName, out)

	// 3. One level deep: scrape subcommands
	// Limit to avoid infinite loops or massive execution time
	for i := range def.Subcommands {
		// e.g. "git clone"
		subName := def.Subcommands[i].Name

		fmt.Printf("Scraping subcommand: %s %s...\n", commandName, subName)

		subOut, err := runHelp(commandName, subName)
		if err == nil {
			// Reuse parsing logic
			subDef := parseHelpOutput(subName, subOut)
			// Merge flags from parsed output into the subcommand struct
			def.Subcommands[i].Flags = subDef.Flags
			// Optionally merge nested sub-subcommands if we wanted to
			// def.Subcommands[i].Subcommands = subDef.Subcommands
		}
	}

	// 4. Save to file
	if err := s.saveDefinition(def); err != nil {
		return nil, err
	}

	return def, nil
}

func runHelp(args ...string) (string, error) {
	args = append(args, "--help")
	cmd := exec.Command(args[0], args[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out // Some commands print help to stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to run %v: %w", args, err)
	}
	return out.String(), nil
}

func parseHelpOutput(name, text string) *definitions.CommandDefinition {
	def := &definitions.CommandDefinition{
		Name:        name,
		Description: "Auto-generated from --help",
		Flags:       []definitions.Flag{},
		Subcommands: []definitions.Subcommand{},
	}

	scanner := bufio.NewScanner(strings.NewReader(text))

	// Regex for flags: "  -f, --flag    Description" or "  --flag    Description"
	flagRegex := regexp.MustCompile(`^\s+(-[a-zA-Z0-9],? )?(--[a-zA-Z0-9-]+)\s+(.*)$`)

	// Regex for subcommands: "   command   Description"
	// Assumes indented, starts with specific chars, followed by >1 space and description
	subcmdRegex := regexp.MustCompile(`^\s+([a-zA-Z][a-zA-Z0-9-_]+)\s{2,}(.*)$`)

	for scanner.Scan() {
		line := scanner.Text()

		// Try matching flags
		if matches := flagRegex.FindStringSubmatch(line); len(matches) > 3 {
			// matches[2] is --flag
			// matches[3] is description
			f := definitions.Flag{
				Name:        matches[2],
				Description: matches[3],
				Type:        "string",
			}
			def.Flags = append(def.Flags, f)
			continue
		}

		// Try matching subcommands
		// Filter out lines that look like flags or header text if possible
		if matches := subcmdRegex.FindStringSubmatch(line); len(matches) > 2 {
			cmdName := matches[1]
			// Avoid common false positives in help text (e.g. "Usage:", "Options:")
			low := strings.ToLower(cmdName)
			if low == "usage:" || low == "options:" || low == "commands:" || low == "arguments:" {
				continue
			}

			s := definitions.Subcommand{
				Name:        cmdName,
				Description: matches[2],
			}
			def.Subcommands = append(def.Subcommands, s)
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
