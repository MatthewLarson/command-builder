# Command Builder (`cb`)

Command Builder is a Linux terminal utility that helps you construct complex commands step-by-step using a filesystem-navigation metaphor.

## Installation

### From Source
Requirements: Go 1.23+

```bash
git clone https://github.com/MatthewLarson/command-builder.git
cd command-builder
make install
```

### Shell Integration (Critical)
To enable history integration (so `cb exec` adds the command to your shell history), add this to your `.bashrc` or `.zshrc`:

```bash
source /usr/share/command-builder/cb.bash
```

## Usage

### Building a Command
Start building a command by adding words to it. You can do this incrementally.

```bash
cb git          # Context: "git"
cb ls           # Lists available subcommands (e.g., commit, push)
cb commit       # Context: "git commit"
cb op           # Lists options (e.g., -m)
cb -m           # Context: "git commit -m"
cb "my message" # Context: "git commit -m 'my message'"
```

### Navigation
- `cb`: Print the current command.
- `cb ..` or `cb back`: Remove the last part of the command.
- `cb clear`: Clear the entire command.

### execution
- `cb exec`: Execute the command you built and add it to your shell history.

## flexible Definitions
`cb` uses YAML definition files to know about subcommands and flags.
1. **Local Cache**: Checks `~/.config/command-builder/definitions/`.
2. **Registry**: Downloads from the [official registry](https://github.com/MatthewLarson/command-builder-definitions) if missing.
3. **Auto-Discovery**: If no definition exists, it runs `command --help` to scrape potential flags.

## Directory Structure
- `cmd/cb`: Main entry point.
- `internal/`: Core logic (state, definitions, registry, scraper).
- `definitions/`: Example YAML definitions.
- `scripts/`: Shell integration scripts.