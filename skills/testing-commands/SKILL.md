---
name: testing-commands
description: Comprehensive guide to testing CLI commands with unit tests, E2E tests, and best practices
argument-hint: ""
---

# Testing Commands

This skill covers comprehensive testing patterns for CLI commands, including unit tests for `preRun` validation and E2E tests for full command execution.

## Overview

Commands should have two types of tests:
1. **Unit Tests** - Test the `preRun` method directly to verify validation logic
2. **E2E Tests** - Test the full command execution including Cobra wiring

## Unit Tests - Testing preRun Directly

Unit tests focus on validating the `preRun` method logic without executing the full command flow.

### Pattern

- **IMPORTANT**: Create an instance of the command struct (e.g., `c := &initCmd{}`)
- **IMPORTANT**: Create a mock `*cobra.Command` and set up required flags
- **IMPORTANT**: Call `c.preRun(cmd, args)` directly - DO NOT call `rootCmd.Execute()`
- Use `t.Run()` for subtests within a parent test function
- Test with different argument/flag combinations
- Verify struct fields are set correctly after `preRun()` executes
- Use `t.TempDir()` for temporary directories (automatic cleanup)

### Example

```go
func TestMyCmd_PreRun(t *testing.T) {
    t.Run("sets fields correctly", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        c := &myCmd{}  // Create command struct instance
        cmd := &cobra.Command{}  // Create mock cobra command
        cmd.Flags().String("storage", storageDir, "test storage flag")

        args := []string{"arg1"}

        err := c.preRun(cmd, args)  // Call preRun directly
        if err != nil {
            t.Fatalf("preRun() failed: %v", err)
        }

        // Assert on struct fields
        if c.manager == nil {
            t.Error("Expected manager to be created")
        }
    })
}
```

## E2E Tests - Testing Full Command Execution

E2E tests verify the complete command flow including Cobra argument parsing, flag handling, and persistence.

### Pattern

- Execute via `rootCmd.Execute()` to test the complete flow
- Use real temp directories with `t.TempDir()`
- Verify output messages
- Verify persistence (check storage/database)
- Verify all field values from `manager.List()` or similar
- Test multiple scenarios (default args, custom args, edge cases)
- Test Cobra's argument validation (e.g., required args, arg counts)

### Example

```go
func TestMyCmd_E2E(t *testing.T) {
    t.Run("executes successfully", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()  // Use full command construction
        rootCmd.SetArgs([]string{"mycommand", "arg1", "--storage", storageDir})

        err := rootCmd.Execute()  // Execute the full command
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify results in storage
        manager, _ := instances.NewManager(storageDir)
        instancesList, _ := manager.List()
        // ... assert on results
    })
}
```

## Testing with Captured Output

Capture command output for verification:

```go
func TestMyCmd_E2E_Output(t *testing.T) {
    t.Run("displays correct output", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"mycommand", "arg1", "--storage", storageDir})

        // Capture output
        var buf bytes.Buffer
        rootCmd.SetOut(&buf)

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        output := buf.String()
        if !strings.Contains(output, "expected message") {
            t.Errorf("Expected output to contain 'expected message', got: %s", output)
        }
    })
}
```

## Testing JSON Output

Test both text and JSON output modes:

```go
func TestMyCmd_E2E_JSON(t *testing.T) {
    t.Run("outputs valid JSON", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"mycommand", "arg1", "--storage", storageDir, "--output", "json"})

        var buf bytes.Buffer
        rootCmd.SetOut(&buf)

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify JSON is valid
        var result MyResultType
        if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
            t.Fatalf("Failed to parse JSON output: %v", err)
        }

        // Verify JSON fields
        if result.Field != "expected value" {
            t.Errorf("Expected field to be 'expected value', got: %s", result.Field)
        }
    })
}
```

## Testing Error Cases

Test that errors are handled correctly:

```go
func TestMyCmd_E2E_Errors(t *testing.T) {
    t.Run("returns error for invalid argument", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"mycommand", "invalid-arg", "--storage", storageDir})

        err := rootCmd.Execute()
        if err == nil {
            t.Fatal("Expected error for invalid argument")
        }

        expectedMsg := "expected error message"
        if !strings.Contains(err.Error(), expectedMsg) {
            t.Errorf("Expected error to contain '%s', got: %s", expectedMsg, err.Error())
        }
    })
}
```

## Testing Cobra Argument Validation

Test that Cobra's `Args` validation works:

```go
func TestMyCmd_E2E_ArgValidation(t *testing.T) {
    t.Run("requires argument", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"mycommand", "--storage", storageDir})  // Missing required arg

        err := rootCmd.Execute()
        if err == nil {
            t.Fatal("Expected error for missing required argument")
        }
    })

    t.Run("rejects too many arguments", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"mycommand", "arg1", "arg2", "--storage", storageDir})  // Too many args

        err := rootCmd.Execute()
        if err == nil {
            t.Fatal("Expected error for too many arguments")
        }
    })
}
```

## Testing with Temp Directories

Always use `t.TempDir()` for temporary directories:

```go
func TestMyCmd_E2E_Persistence(t *testing.T) {
    t.Run("persists data correctly", func(t *testing.T) {
        t.Parallel()

        // Create temp directories - automatically cleaned up
        storageDir := t.TempDir()
        sourcesDir := t.TempDir()

        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"init", sourcesDir, "--storage", storageDir})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }

        // Verify persistence
        manager, err := instances.NewManager(storageDir)
        if err != nil {
            t.Fatalf("Failed to create manager: %v", err)
        }

        instances, err := manager.List()
        if err != nil {
            t.Fatalf("Failed to list instances: %v", err)
        }

        if len(instances) != 1 {
            t.Errorf("Expected 1 instance, got %d", len(instances))
        }
    })
}
```

## Test Organization

Organize tests into logical groups:

```go
// TestMyCmd_PreRun tests the preRun validation logic
func TestMyCmd_PreRun(t *testing.T) {
    // Unit tests for preRun method
}

// TestMyCmd_E2E tests the full command execution
func TestMyCmd_E2E(t *testing.T) {
    // E2E tests for complete flow
}

// TestMyCmd_Examples validates example commands
func TestMyCmd_Examples(t *testing.T) {
    // Example validation (see CLAUDE.md - Adding a New Command)
}
```

## Common Patterns

### Setting Up Test Data

```go
func TestMyCmd_E2E(t *testing.T) {
    t.Run("works with existing data", func(t *testing.T) {
        t.Parallel()

        storageDir := t.TempDir()

        // Set up test data
        manager, _ := instances.NewManager(storageDir)
        instance, _ := instances.NewInstance(instances.NewInstanceParams{
            SourceDir: t.TempDir(),
            ConfigDir: t.TempDir(),
        })
        manager.Add(context.Background(), instances.AddOptions{
            Instance:    instance,
            RuntimeType: "fake",
        })

        // Run command
        rootCmd := NewRootCmd()
        rootCmd.SetArgs([]string{"workspace", "list", "--storage", storageDir})

        err := rootCmd.Execute()
        if err != nil {
            t.Fatalf("Execute() failed: %v", err)
        }
    })
}
```

### Testing Multiple Scenarios

```go
func TestMyCmd_E2E(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr bool
        errMsg  string
    }{
        {
            name:    "valid input",
            args:    []string{"mycommand", "valid-arg"},
            wantErr: false,
        },
        {
            name:    "invalid input",
            args:    []string{"mycommand", "invalid-arg"},
            wantErr: true,
            errMsg:  "expected error",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()

            storageDir := t.TempDir()

            rootCmd := NewRootCmd()
            rootCmd.SetArgs(append(tt.args, "--storage", storageDir))

            err := rootCmd.Execute()
            if (err != nil) != tt.wantErr {
                t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
            }

            if tt.wantErr && !strings.Contains(err.Error(), tt.errMsg) {
                t.Errorf("Expected error to contain '%s', got: %s", tt.errMsg, err.Error())
            }
        })
    }
}
```

## Related Skills

- `/implementing-command-patterns` - Command implementation patterns
- `/testing-best-practices` - General testing best practices
- `/cross-platform-development` - Cross-platform testing patterns

## References

- **Example Tests**: `pkg/cmd/init_test.go`, `pkg/cmd/workspace_list_test.go`, `pkg/cmd/workspace_remove_test.go`
- **Command Pattern**: `pkg/cmd/init.go`
- **Test Utilities**: `pkg/cmd/testutil/`
