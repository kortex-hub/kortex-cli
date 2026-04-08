---
name: add-alias-command
description: Add an alias command that delegates to an existing command
argument-hint: <alias-name> <target-command>
---

# Add Alias Command

This skill helps you create an alias command that provides a shortcut to an existing command (e.g., `list` as an alias for `workspace list`).

## Prerequisites

- Alias name (e.g., "list", "remove", "add")
- Target command that the alias delegates to (e.g., "workspace list")
- The target command must already exist

## Implementation Steps

### 1. Create the Alias Command File

Create `pkg/cmd/<alias>.go` with the following structure:

```go
package cmd

import (
    "github.com/spf13/cobra"
)

func New<Alias>Cmd() *cobra.Command {
    // Create the target command
    targetCmd := New<Target><SubCommand>Cmd()

    // Create an alias command that delegates to the target
    cmd := &cobra.Command{
        Use:     "<alias>",
        Short:   targetCmd.Short,
        Long:    targetCmd.Long,
        Example: AdaptExampleForAlias(targetCmd.Example, "<target> <subcommand>", "<alias>"),
        Args:    targetCmd.Args,
        PreRunE: targetCmd.PreRunE,
        RunE:    targetCmd.RunE,
    }

    // Copy flags from target command
    cmd.Flags().AddFlagSet(targetCmd.Flags())

    return cmd
}
```

**Example: Creating `list` as an alias for `workspace list`**

```go
package cmd

import (
    "github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
    // Create the workspace list command
    workspaceListCmd := NewWorkspaceListCmd()

    // Create an alias command that delegates to workspace list
    cmd := &cobra.Command{
        Use:     "list",
        Short:   workspaceListCmd.Short,
        Long:    workspaceListCmd.Long,
        Example: AdaptExampleForAlias(workspaceListCmd.Example, "workspace list", "list"),
        Args:    workspaceListCmd.Args,
        PreRunE: workspaceListCmd.PreRunE,
        RunE:    workspaceListCmd.RunE,
    }

    // Copy flags from workspace list command
    cmd.Flags().AddFlagSet(workspaceListCmd.Flags())

    return cmd
}
```

### 2. Understanding AdaptExampleForAlias

The `AdaptExampleForAlias()` helper function (from `pkg/cmd/helpers.go`) automatically adapts examples for the alias:

**What it does:**
- Replaces the target command with the alias **only in command lines** (lines starting with `kdn`)
- **Preserves comments unchanged** (lines starting with `#`)
- Maintains formatting and indentation

**Example transformation:**
```bash
# Original (from workspace list):
# List all workspaces
kdn workspace list

# List in JSON format
kdn workspace list --output json

# After AdaptExampleForAlias(..., "workspace list", "list"):
# List all workspaces
kdn list

# List in JSON format
kdn list --output json
```

### 3. Register the Alias Command

In `pkg/cmd/root.go`, add to the `NewRootCmd()` function:

```go
rootCmd.AddCommand(New<Alias>Cmd())
```

### 4. Create Tests

Create `pkg/cmd/<alias>_test.go`:

```go
package cmd

import (
    "strings"
    "testing"
)

func Test<Alias>Cmd_E2E(t *testing.T) {
    t.Parallel()

    t.Run("executes as alias", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        var output strings.Builder
        rootCmd.SetOut(&output)
        rootCmd.SetArgs([]string{"<alias>", "--storage", storageDir})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify output is the same as target command
        outputStr := output.String()
        if !strings.Contains(outputStr, "expected output") {
            t.Errorf("Expected output, got: %s", outputStr)
        }
    })

    t.Run("supports same flags as target", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        var output strings.Builder
        rootCmd.SetOut(&output)
        rootCmd.SetArgs([]string{"<alias>", "--storage", storageDir, "--output", "json"})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify JSON output works
        outputStr := output.String()
        if !strings.HasPrefix(strings.TrimSpace(outputStr), "{") {
            t.Errorf("Expected JSON output, got: %s", outputStr)
        }
    })
}
```

**IMPORTANT: Do NOT create a Test<Alias>Cmd_Examples test for alias commands.** Aliases use the same validation as the target command, so creating a separate validation test would be redundant.

### 5. Run Tests

```bash
# Run tests for the alias command
go test ./pkg/cmd -run Test<Alias>

# Run all tests
make test
```

### 6. Update Documentation

Update relevant documentation to mention the alias as an alternative way to invoke the command.

## Key Points

- **Delegation**: Alias commands delegate ALL behavior to the target command (Args, PreRunE, RunE, flags)
- **Examples**: Use `AdaptExampleForAlias()` to automatically adapt examples
- **No Validation Tests**: Do NOT create Test<Alias>Cmd_Examples tests for aliases
- **Flag Copying**: Use `cmd.Flags().AddFlagSet(targetCmd.Flags())` to copy all flags
- **Same Behavior**: The alias should behave identically to the target command
- **Documentation**: Mention the alias in help text and user documentation

## Common Aliases

- `list` → `workspace list`
- `remove` → `workspace remove`
- `add` → `workspace add`
- `init` → `workspace init`

## Complete Example

**File: `pkg/cmd/remove.go`**
```go
package cmd

import (
    "github.com/spf13/cobra"
)

func NewRemoveCmd() *cobra.Command {
    // Create the workspace remove command
    workspaceRemoveCmd := NewWorkspaceRemoveCmd()

    // Create an alias command that delegates to workspace remove
    cmd := &cobra.Command{
        Use:     "remove <workspace-id>",
        Short:   workspaceRemoveCmd.Short,
        Long:    workspaceRemoveCmd.Long,
        Example: AdaptExampleForAlias(workspaceRemoveCmd.Example, "workspace remove", "remove"),
        Args:    workspaceRemoveCmd.Args,
        PreRunE: workspaceRemoveCmd.PreRunE,
        RunE:    workspaceRemoveCmd.RunE,
    }

    // Copy flags from workspace remove command
    cmd.Flags().AddFlagSet(workspaceRemoveCmd.Flags())

    return cmd
}
```

**File: `pkg/cmd/remove_test.go`**
```go
package cmd

import (
    "strings"
    "testing"
)

func TestRemoveCmd_E2E(t *testing.T) {
    t.Parallel()

    t.Run("removes workspace by ID", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()
        sourceDir := t.TempDir()

        // Setup: Add a workspace first
        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"init", sourceDir, "--storage", storageDir})
        _ = rootCmd.Execute()

        // Get the workspace ID
        rootCmd = NewRootCmd()
        var listOutput strings.Builder
        rootCmd.SetOut(&listOutput)
        rootCmd.SetArgs([]string{"list", "--storage", storageDir, "--output", "json"})
        _ = rootCmd.Execute()

        // Extract ID from JSON output
        // ... (parsing logic)

        // Test: Remove the workspace using alias
        rootCmd = NewRootCmd()
        var output strings.Builder
        rootCmd.SetOut(&output)
        rootCmd.SetArgs([]string{"remove", workspaceID, "--storage", storageDir})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify success message
        outputStr := output.String()
        if !strings.Contains(outputStr, "Successfully removed workspace") {
            t.Errorf("Expected success message, got: %s", outputStr)
        }
    })
}

// No Test_RemoveCmd_Examples - aliases don't need separate validation
```

## References

- `pkg/cmd/list.go` - List command alias
- `pkg/cmd/remove.go` - Remove command alias
- `pkg/cmd/helpers.go` - AdaptExampleForAlias implementation
- CLAUDE.md - Alias Commands section
