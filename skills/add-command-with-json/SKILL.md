---
name: add-command-with-json
description: Add a new CLI command with JSON output support
argument-hint: <command-name>
---

# Add Command with JSON Output Support

This skill helps you add a new CLI command that supports both text and JSON output formats.

## Prerequisites

- Command name (e.g., "status", "validate", "config")
- Understanding of what the command should do
- Knowledge of required arguments and flags

## Implementation Steps

### 1. Create the Command File

Create `pkg/cmd/<command>.go` with the following structure:

```go
package cmd

import (
    "encoding/json"
    "fmt"
    "io"
    "path/filepath"

    "github.com/spf13/cobra"
    // Add other imports as needed
)

type <command>Cmd struct {
    output  string  // Bound to --output flag
    // Add other fields for flags and dependencies
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

# With JSON output
kortex-cli <command> --output json

# With other flags
kortex-cli <command> --flag value`,
        Args:    cobra.NoArgs,  // Or ExactArgs(1), MinimumNArgs(1), etc.
        PreRunE: c.preRun,
        RunE:    c.run,
    }

    // Bind flags to struct fields using *Var variants
    cmd.Flags().StringVarP(&c.output, "output", "o", "", "Output format (supported: json)")

    return cmd
}

func (c *<command>Cmd) preRun(cmd *cobra.Command, args []string) error {
    // 1. FIRST: Validate output format
    if c.output != "" && c.output != "json" {
        return fmt.Errorf("unsupported output format: %s (supported: json)", c.output)
    }

    // 2. EARLY: Silence Cobra's error output in JSON mode
    if c.output == "json" {
        cmd.SilenceErrors = true
        cmd.SilenceUsage = true
    }

    // 3. ALL subsequent errors use outputErrorIfJSON
    storageDir, err := cmd.Flags().GetString("storage")
    if err != nil {
        return outputErrorIfJSON(cmd, c.output, fmt.Errorf("failed to read --storage flag: %w", err))
    }

    // Validate arguments
    // Convert paths to absolute if needed
    // Create dependencies (manager, etc.)
    // Store in struct fields

    return nil
}

func (c *<command>Cmd) run(cmd *cobra.Command, args []string) error {
    // Perform the command logic

    // ALL errors use outputErrorIfJSON
    data, err := c.manager.GetData()
    if err != nil {
        return outputErrorIfJSON(cmd, c.output, fmt.Errorf("failed to get data: %w", err))
    }

    if c.output == "json" {
        return c.outputJSON(cmd, data)
    }

    // Text output
    cmd.Println("Success message")
    return nil
}

func (c *<command>Cmd) outputJSON(cmd *cobra.Command, data interface{}) error {
    jsonData, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return outputErrorIfJSON(cmd, c.output, fmt.Errorf("failed to marshal to JSON: %w", err))
    }

    fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
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
    "encoding/json"
    "path/filepath"
    "strings"
    "testing"

    "github.com/kortex-labs/kortex-cli/pkg/cmd/testutil"
)

func Test<Command>Cmd_PreRun(t *testing.T) {
    t.Parallel()

    t.Run("validates output format", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        c := &<command>Cmd{
            output: "xml",  // Invalid format
        }
        cmd := &cobra.Command{}
        cmd.Flags().String("storage", storageDir, "test storage flag")

        err := c.preRun(cmd, []string{})
        if err == nil {
            t.Fatal("Expected error for invalid output format")
        }
        if !strings.Contains(err.Error(), "unsupported output format") {
            t.Errorf("Expected 'unsupported output format' error, got: %v", err)
        }
    })

    t.Run("sets fields correctly", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        c := &<command>Cmd{
            output: "json",
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
}

func Test<Command>Cmd_E2E(t *testing.T) {
    t.Parallel()

    t.Run("executes with text output", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"<command>", "--storage", storageDir})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }
    })

    t.Run("executes with JSON output", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        var output strings.Builder
        rootCmd.SetOut(&output)
        rootCmd.SetArgs([]string{"<command>", "--storage", storageDir, "--output", "json"})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify JSON structure
        var result map[string]interface{}
        if err := json.Unmarshal([]byte(output.String()), &result); err != nil {
            t.Fatalf("Failed to parse JSON output: %v", err)
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
- **Output Validation**: Check output format FIRST in preRun
- **Error Handling**: Always use `outputErrorIfJSON()` for ALL errors after setting up JSON mode
- **JSON Mode Setup**: Set `cmd.SilenceErrors = true` and `cmd.SilenceUsage = true` early in preRun
- **Examples**: Include 3-5 clear examples showing common use cases
- **Testing**: Create both unit tests (preRun) and E2E tests (full execution)
- **Example Validation**: Always add a Test<Command>Cmd_Examples test

## References

- `pkg/cmd/init.go` - Complete implementation with JSON output
- `pkg/cmd/workspace_list.go` - List command with JSON output
- `pkg/cmd/workspace_remove.go` - Remove command with JSON output
- `pkg/cmd/conversion.go` - Helper functions for JSON error handling
- CLAUDE.md - Full documentation of patterns and best practices
