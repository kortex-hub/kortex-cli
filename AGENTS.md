# AGENTS.md

This file provides guidance to AI agents when working with code in this repository.

## Project Overview

kortex-cli is a command-line interface for launching and managing AI agents (Claude Code, Goose, Cursor) with custom configurations. It provides a unified way to start different agents with specific settings including skills, MCP server connections, and LLM integrations.

## Build and Test Commands

All build and test commands are available through the Makefile. Run `make help` to see all available commands.

### Build
```bash
make build
```

### Execute
After building, the `kortex-cli` binary will be created in the current directory:

```bash
# Display help and available commands
./kortex-cli --help

# Execute a specific command
./kortex-cli <command> [flags]
```

### Run Tests
```bash
# Run all tests
make test

# Run tests with coverage report
make test-coverage
```

For more granular testing (specific packages or tests), use Go directly:
```bash
# Run tests in a specific package
go test ./pkg/cmd

# Run a specific test
go test -run TestName ./pkg/cmd
```

### Format Code
```bash
# Format all Go files in the project
make fmt

# Check if code is formatted (without modifying files)
make check-fmt
```

Code should be formatted before committing. Run `make fmt` to ensure consistent style across the codebase.

### Additional Commands
```bash
# Run go vet
make vet

# Run all CI checks (format check, vet, tests)
make ci-checks

# Clean build artifacts
make clean

# Install binary to GOPATH/bin
make install
```

## Architecture

### Command Structure (Cobra-based)
- Entry point: `cmd/kortex-cli/main.go` → calls `cmd.NewRootCmd().Execute()` and handles errors with `os.Exit(1)`
- Root command: `pkg/cmd/root.go` exports `NewRootCmd()` which creates and configures the root command
- Subcommands: Each command is in `pkg/cmd/<command>.go` with a `New<Command>Cmd()` factory function
- Commands use a factory pattern: each command exports a `New<Command>Cmd()` function that returns `*cobra.Command`
- Command registration: `NewRootCmd()` calls `rootCmd.AddCommand(New<Command>Cmd())` for each subcommand
- No global variables or `init()` functions - all configuration is explicit through factory functions

### Global Flags
Global flags are defined as persistent flags in `pkg/cmd/root.go` and are available to all commands.

#### Accessing the --storage Flag
The `--storage` flag specifies the directory where kortex-cli stores all its files. The default path is computed at runtime using `os.UserHomeDir()` and `filepath.Join()` to ensure cross-platform compatibility (Linux, macOS, Windows). The default is `$HOME/.kortex-cli` with a fallback to `.kortex-cli` in the current directory if the home directory cannot be determined.

**Environment Variable**: The `KORTEX_CLI_STORAGE` environment variable can be used to set the storage directory path. The flag `--storage` will override the environment variable if both are specified.

**Priority order** (highest to lowest):
1. `--storage` flag (if specified)
2. `KORTEX_CLI_STORAGE` environment variable (if set)
3. Default: `$HOME/.kortex-cli`

To access this value in any command:

```go
func NewExampleCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "example",
        Short: "An example command",
        Run: func(cmd *cobra.Command, args []string) {
            storagePath, _ := cmd.Flags().GetString("storage")
            // Use storagePath...
        },
    }
}
```

**Important**: Never hardcode paths with `~` as it's not cross-platform. Always use `os.UserHomeDir()` and `filepath.Join()` for path construction.

### Module Design Pattern

All modules (packages outside of `cmd/`) MUST follow the interface-based design pattern to ensure proper encapsulation, testability, and API safety.

**Required Pattern:**
1. **Public types are interfaces** - All public types must be declared as interfaces
2. **Implementations are unexported** - Concrete struct implementations must be unexported (lowercase names)
3. **Compile-time interface checks** - Add unnamed variable declarations to verify interface implementation at compile time
4. **Factory functions** - Provide `New*()` functions that return the interface type

**Benefits:**
- Prevents direct struct instantiation (compile-time enforcement)
- Forces usage of factory functions for proper validation and initialization
- Enables easy mocking in tests
- Clear API boundaries
- Better encapsulation

**This pattern is MANDATORY for all new modules in `pkg/`.**

### Skills System
Skills are reusable capabilities that can be discovered and executed by AI agents:
- **Location**: `skills/<skill-name>/SKILL.md`
- **Claude support**: Skills are symlinked in `.claude/skills/` for Claude Code
- **Format**: Each SKILL.md contains:
  - YAML frontmatter with `name`, `description`, `argument-hint`
  - Detailed instructions for execution
  - Usage examples

### Adding a New Skill
1. Create directory: `skills/<skill-name>/`
2. Create SKILL.md with frontmatter and instructions
3. Symlink in `.claude/skills/`: `ln -s ../../skills/<skill-name> .claude/skills/<skill-name>`

### Adding a New Command
1. Create `pkg/cmd/<command>.go` with a `New<Command>Cmd()` function that returns `*cobra.Command`
2. In the `New<Command>Cmd()` function:
   - Create and configure the `cobra.Command`
   - Set up any flags or subcommands
   - Return the configured command
3. Register the command in `pkg/cmd/root.go` by adding `rootCmd.AddCommand(New<Command>Cmd())` in the `NewRootCmd()` function
4. Create corresponding test file `pkg/cmd/<command>_test.go`
5. In tests, create command instances using `NewRootCmd()` or `New<Command>Cmd()` as needed

Example:
```go
// pkg/cmd/example.go
func NewExampleCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "example",
        Short: "An example command",
        Run: func(cmd *cobra.Command, args []string) {
            // Command logic here
        },
    }
}

// In pkg/cmd/root.go, add to NewRootCmd():
rootCmd.AddCommand(NewExampleCmd())
```

## Copyright Headers

All source files must include Apache License 2.0 copyright headers with Red Hat copyright. Use the `/copyright-headers` skill to add or update headers automatically. The current year is 2026.

## Dependencies

- Cobra (github.com/spf13/cobra): CLI framework
- Go 1.25+

## Testing

Tests follow Go conventions with `*_test.go` files alongside source files. Tests use the standard `testing` package and should cover command initialization, execution, and error cases.

### Parallel Test Execution

**All tests MUST call `t.Parallel()` as the first line of the test function.**

This ensures faster test execution and better resource utilization. Every test function should start with:

```go
func TestExample(t *testing.T) {
    t.Parallel()

    // Test code here...
}
```

### Testing with Fake Objects

When testing code that uses interfaces (following the Module Design Pattern), **use fake implementations instead of real implementations or mocks**.

**Pattern:**
1. Create unexported fake structs that implement the interface
2. Use factory injection to provide fakes to the code under test
3. Control fake behavior through constructor parameters or fields

**Benefits:**
- **No external dependencies** - Fakes are simple structs with no framework requirements
- **Full control** - Control exact behavior through fields/parameters
- **Type-safe** - Compile-time verification that fakes implement interfaces
- **Easy to understand** - Fakes are just plain Go code
- **Flexible** - Can create different factories for different test scenarios

**Reference:** See `pkg/instances/manager_test.go` for a complete implementation of this pattern with factory injection.

## GitHub Actions

GitHub Actions workflows are stored in `.github/workflows/`. All workflows must use commit SHA1 hashes instead of version tags for security reasons (to prevent supply chain attacks from tag manipulation).

Example:
```yaml
- uses: actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd # v6.0.2
```

Always include the version as a comment for readability.
