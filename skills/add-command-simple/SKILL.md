---
name: add-command-simple
description: Add a simple CLI command without JSON output support
argument-hint: <command-name>
---

# Add Simple CLI Command

This skill helps you add a simple CLI command that only provides text output (no JSON support).

## Prerequisites

- Command name (e.g., "clean", "validate", "reset")
- Understanding of what the command should do
- Knowledge of required arguments and flags

## Implementation Steps

### 1. Create the Command File

Create `pkg/cmd/<command>.go` with the following structure:

```go
package cmd

import (
    "fmt"
    "path/filepath"

    "github.com/spf13/cobra"
    // Add other imports as needed
)

type <command>Cmd struct {
    // Fields for flags and dependencies
    verbose bool
    manager instances.Manager
}

func New<Command>Cmd() *cobra.Command {
    c := &<command>Cmd{}

    cmd := &cobra.Command{
        Use:   "<command> [args]",
        Short: "Short description of the command",
        Long:  `Long description of what the command does.`,
        Example: `# Basic usage
kortex-cli <command>

# With verbose output
kortex-cli <command> --verbose

# With other flags
kortex-cli <command> --flag value`,
        Args:    cobra.NoArgs,  // Or ExactArgs(1), MinimumNArgs(1), etc.
        PreRunE: c.preRun,
        RunE:    c.run,
    }

    // Bind flags to struct fields using *Var variants
    cmd.Flags().BoolVarP(&c.verbose, "verbose", "v", false, "Show detailed output")

    return cmd
}

func (c *<command>Cmd) preRun(cmd *cobra.Command, args []string) error {
    // Get global flags
    storageDir, err := cmd.Flags().GetString("storage")
    if err != nil {
        return fmt.Errorf("failed to read --storage flag: %w", err)
    }

    // Validate arguments
    // Convert paths to absolute if needed
    // Create dependencies (manager, etc.)

    // Example: Create manager
    manager, err := instances.NewManager(storageDir)
    if err != nil {
        return fmt.Errorf("failed to create manager: %w", err)
    }
    c.manager = manager

    return nil
}

func (c *<command>Cmd) run(cmd *cobra.Command, args []string) error {
    // Perform the command logic

    data, err := c.manager.GetData()
    if err != nil {
        return fmt.Errorf("failed to get data: %w", err)
    }

    // Output results
    cmd.Println("Success message")

    if c.verbose {
        cmd.Printf("Detailed information: %v\n", data)
    }

    return nil
}
```

### 2. Register the Command

In `pkg/cmd/root.go`, add to the `NewRootCmd()` function:

```go
rootCmd.AddCommand(New<Command>Cmd())
```

### 3. Create Tests

Create `pkg/cmd/<command>_test.go`:

```go
package cmd

import (
    "strings"
    "testing"

    "github.com/kortex-labs/kortex-cli/pkg/cmd/testutil"
    "github.com/spf13/cobra"
)

func Test<Command>Cmd_PreRun(t *testing.T) {
    t.Parallel()

    t.Run("sets fields correctly", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        c := &<command>Cmd{
            verbose: true,
        }
        cmd := &cobra.Command{}
        cmd.Flags().String("storage", storageDir, "test storage flag")

        err := c.preRun(cmd, []string{})
        if err != nil {
            t.Fatalf("preRun() failed: %v", err)
        }

        if c.manager == nil {
            t.Error("Expected manager to be created")
        }
    })

    t.Run("validates arguments", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        c := &<command>Cmd{}
        cmd := &cobra.Command{}
        cmd.Flags().String("storage", storageDir, "test storage flag")

        // Test with invalid args if applicable
        args := []string{"invalid"}
        err := c.preRun(cmd, args)

        // Assert on expected errors
        if err == nil {
            t.Fatal("Expected error for invalid arguments")
        }
    })
}

func Test<Command>Cmd_E2E(t *testing.T) {
    t.Parallel()

    t.Run("executes successfully", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        var output strings.Builder
        rootCmd.SetOut(&output)
        rootCmd.SetArgs([]string{"<command>", "--storage", storageDir})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify output contains expected messages
        outputStr := output.String()
        if !strings.Contains(outputStr, "Success message") {
            t.Errorf("Expected success message in output, got: %s", outputStr)
        }
    })

    t.Run("executes with verbose flag", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        var output strings.Builder
        rootCmd.SetOut(&output)
        rootCmd.SetArgs([]string{"<command>", "--storage", storageDir, "--verbose"})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify verbose output
        outputStr := output.String()
        if !strings.Contains(outputStr, "Detailed information") {
            t.Errorf("Expected detailed information in verbose output, got: %s", outputStr)
        }
    })
}

func Test<Command>Cmd_Examples(t *testing.T) {
    t.Parallel()

    cmd := New<Command>Cmd()

    if cmd.Example == "" {
        t.Fatal("Example field should not be empty")
    }

    commands, err := testutil.ParseExampleCommands(cmd.Example)
    if err != nil {
        t.Fatalf("Failed to parse examples: %v", err)
    }

    expectedCount := 3  // Adjust based on your examples
    if len(commands) != expectedCount {
        t.Errorf("Expected %d example commands, got %d", expectedCount, len(commands))
    }

    rootCmd := NewRootCmd()
    err = testutil.ValidateCommandExamples(rootCmd, cmd.Example)
    if err != nil {
        t.Errorf("Example validation failed: %v", err)
    }
}
```

### 4. Run Tests

```bash
# Run tests for the new command
go test ./pkg/cmd -run Test<Command>

# Run all tests
make test
```

### 5. Update Documentation

If the command warrants user-facing documentation, update relevant docs.

## Key Points

- **Flag Binding**: Always use `*Var` variants (StringVarP, BoolVarP, etc.) to bind flags to struct fields
- **Error Messages**: Provide clear, actionable error messages
- **Examples**: Include 3-5 clear examples showing common use cases
- **Testing**: Create both unit tests (preRun) and E2E tests (full execution)
- **Example Validation**: Always add a Test<Command>Cmd_Examples test
- **Parallel Tests**: All test functions should call `t.Parallel()` as the first line
- **Cross-Platform Paths**: Use `filepath.Join()` and `t.TempDir()` for all path operations

## Common Flag Patterns

```go
// Boolean flag
cmd.Flags().BoolVarP(&c.force, "force", "f", false, "Force operation without confirmation")

// String flag
cmd.Flags().StringVarP(&c.format, "format", "o", "text", "Output format")

// Integer flag
cmd.Flags().IntVarP(&c.count, "count", "c", 10, "Number of items")

// String slice flag
cmd.Flags().StringSliceVarP(&c.tags, "tag", "t", nil, "Tags to apply")
```

## Common Argument Validators

```go
// No arguments
Args: cobra.NoArgs,

// Exactly N arguments
Args: cobra.ExactArgs(1),
Args: cobra.ExactArgs(2),

// At least N arguments
Args: cobra.MinimumNArgs(1),

// At most N arguments
Args: cobra.MaximumNArgs(1),

// Range of arguments
Args: cobra.RangeArgs(1, 3),
```

## References

- `pkg/cmd/workspace.go` - Parent command (simpler pattern)
- CLAUDE.md - Full documentation of patterns and best practices
