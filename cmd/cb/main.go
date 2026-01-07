package main

import (
	"fmt"
	"os"
	"strings"

	"command-builder/internal/definitions"
	"command-builder/internal/registry"
	"command-builder/internal/scraper"
	"command-builder/internal/state"
)

func main() {
	stateMgr, err := state.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing state manager: %v\n", err)
		os.Exit(1)
	}

	st, err := stateMgr.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading state: %v\n", err)
		os.Exit(1)
	}

	args := os.Args[1:]

	if len(args) == 0 {
		printCommand(st)
		return
	}

	command := args[0]
	stateModified := false

	switch command {
	case "ls":
		handleLs(st)
		return // Don't print command status after ls
	case "op":
		handleOp(st)
		return // Don't print command status after op
	case "add", "cd":
		if len(args) < 2 {
			fmt.Println("Usage: cb add <subcommand> [args...]")
			return
		}
		st.CommandParts = append(st.CommandParts, args[1:]...)
		stateModified = true
	case "..", "back":
		if len(st.CommandParts) > 0 {
			st.CommandParts = st.CommandParts[:len(st.CommandParts)-1]
			stateModified = true
		}
	case "clear":
		st.CommandParts = []string{}
		stateModified = true
	case "exec":
		handleExec(stateMgr, st)
		return
	default:
		// Implicit Add: Interpret all arguments as command parts
		st.CommandParts = append(st.CommandParts, args...)
		stateModified = true
	}

	if stateModified {
		if err := stateMgr.Save(st); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving state: %v\n", err)
			os.Exit(1)
		}
	}

	printCommand(st)
}

func printCommand(st *state.State) {
	if len(st.CommandParts) == 0 {
		fmt.Println("(empty)")
		return
	}
	fmt.Println(strings.Join(st.CommandParts, " "))
}

func handleExec(mgr *state.Manager, st *state.State) {
	if len(st.CommandParts) == 0 {
		return
	}
	// Print the command to stdout so the shell wrapper can capture it
	fmt.Println(strings.Join(st.CommandParts, " "))

	// Clear the state after execution
	if err := mgr.Clear(); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing state: %v\n", err)
	}
}

func handleLs(st *state.State) {
	defMgr, err := definitions.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing definition manager: %v\n", err)
		return
	}

	if len(st.CommandParts) == 0 {
		fmt.Println("Available commands (definitions found):")
		names, err := defMgr.ListDefinitions()
		if err != nil {
			fmt.Printf("  Error listing definitions: %v\n", err)
			return
		}
		if len(names) == 0 {
			fmt.Println("  (none found in local cache)")
		}
		for _, name := range names {
			fmt.Printf("  %s\n", name)
		}
		return
	}

	rootCmdName := st.CommandParts[0]
	def, err := ensureDefinition(rootCmdName, defMgr)
	if err != nil {
		fmt.Printf("No definition found for '%s' (checked local, registry, and scraper).\n", rootCmdName)
		return
	}

	var subcommands []definitions.Subcommand

	if len(st.CommandParts) == 1 {
		subcommands = def.Subcommands
	} else {
		path := st.CommandParts[1:]
		currentSub := def.FindSubcommand(path)
		if currentSub == nil {
			fmt.Println("Current context not found in definition.")
			return
		}
		subcommands = currentSub.Subcommands
	}

	if len(subcommands) == 0 {
		fmt.Println("No subcommands available.")
		return
	}

	for _, sub := range subcommands {
		fmt.Printf("  %s\t%s\n", sub.Name, sub.Description)
	}
}

func handleOp(st *state.State) {
	defMgr, err := definitions.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing definition manager: %v\n", err)
		return
	}

	if len(st.CommandParts) == 0 {
		fmt.Println("No command context.")
		return
	}

	rootCmdName := st.CommandParts[0]
	def, err := ensureDefinition(rootCmdName, defMgr)
	if err != nil {
		fmt.Printf("No definition found for '%s'.\n", rootCmdName)
		return
	}

	var flags []definitions.Flag

	if len(st.CommandParts) == 1 {
		flags = def.Flags
	} else {
		path := st.CommandParts[1:]
		currentSub := def.FindSubcommand(path)
		if currentSub == nil {
			fmt.Println("Current context not found definition.")
			return
		}
		flags = currentSub.Flags
	}

	if len(flags) == 0 {
		fmt.Println("No options available.")
		return
	}

	for _, f := range flags {
		fmt.Printf("  %s\t%s\n", f.Name, f.Description)
	}
}

// ensureDefinition tries to load from local, then registry, then scraper.
func ensureDefinition(name string, defMgr *definitions.Manager) (*definitions.CommandDefinition, error) {
	// 1. Try Local
	def, err := defMgr.LoadDefinition(name)
	if err == nil {
		return def, nil
	}

	fmt.Printf("Definition for '%s' not found locally. Searching registry...\n", name)

	// 2. Try Registry
	regClient := registry.NewClient(defMgr)
	if err := regClient.FetchDefinition(name); err == nil {
		fmt.Println("Downloaded definition from registry.")
		return defMgr.LoadDefinition(name)
	}

	fmt.Printf("Definition not found in registry. Attempting to scrape '--help'...\n", name)

	// 3. Try Scraper
	scr, err := scraper.NewScraper()
	if err != nil {
		return nil, fmt.Errorf("failed to init scraper: %w", err)
	}
	def, err = scr.Scrape(name)
	if err == nil {
		fmt.Println("Generated definition from help output.")
		return def, nil
	}

	return nil, fmt.Errorf("could not find or generate definition")
}
