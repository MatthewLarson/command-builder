package main

import (
	"command-builder/internal/definitions"
	"command-builder/internal/registry"
	"command-builder/internal/scraper"
	"command-builder/internal/state"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"golang.org/x/term"
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
		printLocalDefinitions(defMgr)
		return
	}

	rootCmdName := st.CommandParts[0]
	def, err := ensureDefinition(rootCmdName, defMgr)
	if err != nil {
		fmt.Printf("No definition found for '%s'.\n", rootCmdName)
		return
	}

	// Navigate to the correct context
	var flags []definitions.Flag
	var subcommands []definitions.Subcommand
	var args []definitions.Argument

	// Default context is root
	flags = def.Flags
	subcommands = def.Subcommands
	args = def.Args

	// Traverse path
	// If the last part is an option (starts with -), use the PARENT context
	// If the last part is a subcommand, use THAT context.

	validPath := st.CommandParts[1:]

	// Check if last part is a flag
	if len(validPath) > 0 {
		last := validPath[len(validPath)-1]
		if strings.HasPrefix(last, "-") {
			// User just typed a flag. They might be looking for a value for it,
			// OR they are done with it and want to see what else they can do.
			// Revert to parent context (remove the flag) to show available options again.
			// But specialized logic: "Can have a string... optional/required"
			// Since we don't have per-flag value definitions yet, we will just show parent context
			// effectively ignoring the dangling flag for 'ls' purposes,
			// UNLESS we implement precise flag arg detection later.
			// For now: pop the flag from path
			validPath = validPath[:len(validPath)-1]
		}
	}

	// Re-traverse with cleaned path
	if len(validPath) > 0 {
		currentSub := def.FindSubcommand(validPath)
		if currentSub != nil {
			flags = currentSub.Flags
			subcommands = currentSub.Subcommands
			args = currentSub.Args
		} else {
			// If we can't find the exact subcommand (maybe it's a positional arg value?),
			// we stick to what we found furthest down or root.
			// Simplified: if exact match fails, maybe it's just args.
			// For now, let's assume if FindSubcommand returns nil for a partial path,
			// it might mean we are "inside" an argument.
			// But FindSubcommand is strict.
			// Let's rely on the previous logic: if exact match fails, we might just be at root or
			// an intermediate point.
			// NOTE: Improvement needed here for complex paths.
		}
	}

	// Calculate max name length to determine description width
	maxNameLen := 0

	// Helper to check length
	updateMax := func(name string) {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}
	}

	for _, arg := range args {
		updateMax(arg.Name)
	}
	for _, sub := range subcommands {
		updateMax(sub.Name)
	}
	for _, f := range flags {
		updateMax(f.Name)
	}

	// Get terminal width
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width <= 0 {
		width = 80 // fallback
	}

	// Calculate available width for description
	// Padding (4 spaces) + Tab + Indent (~2)
	// Let's aim safely: Width - (MaxNameLen + 8)
	descWidth := width - (maxNameLen + 8)
	if descWidth < 20 {
		descWidth = 20 // minimum readable width
	}

	// Setup TabWriter with padding=4 for "extra tab away" feel
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)

	hasOutput := false

	// Helper to print arguments/items with wrapping
	printItem := func(name, desc string, extras ...string) {
		fullDesc := desc
		if len(extras) > 0 {
			fullDesc += " " + extras[0]
		}

		lines := wrapString(fullDesc, descWidth)
		if len(lines) == 0 {
			lines = []string{""}
		}

		// First line
		fmt.Fprintf(w, "  %s\t%s\n", name, lines[0])

		// Subsequent lines
		for _, line := range lines[1:] {
			fmt.Fprintf(w, "  \t%s\n", line)
		}
	}

	// 1. Arguments
	if len(args) > 0 {
		fmt.Fprintln(w, "Arguments:")
		for _, arg := range args {
			reqStr := "(Optional)"
			if arg.Required {
				reqStr = "(Required)"
			}
			printItem(arg.Name, arg.Description, reqStr)
		}
		fmt.Fprintln(w, "")
		hasOutput = true
	}

	// 2. Subcommands
	if len(subcommands) > 0 {
		fmt.Fprintln(w, "Subcommands:")
		for _, sub := range subcommands {
			printItem(sub.Name, sub.Description)
		}
		fmt.Fprintln(w, "")
		hasOutput = true
	}

	// 3. Options
	if len(flags) > 0 {
		fmt.Fprintln(w, "Options:")
		for _, f := range flags {
			printItem(f.Name, f.Description)
		}
		hasOutput = true
	}

	if !hasOutput {
		fmt.Println("No further options or subcommands available.")
	} else {
		w.Flush()
	}
}

func printLocalDefinitions(defMgr *definitions.Manager) {
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
}

// wrapString splits a string into lines no longer than maxWidth, breaking at words.
func wrapString(s string, maxWidth int) []string {
	var lines []string
	words := strings.Fields(s)
	if len(words) == 0 {
		return lines
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	lines = append(lines, currentLine)
	return lines
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
