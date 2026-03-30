---
name: working-with-steplogger
description: Complete guide to integrating StepLogger for user progress feedback in commands and runtimes
argument-hint: ""
---

# Working with StepLogger

The StepLogger system provides user-facing progress feedback during runtime operations. It displays operational steps with spinners and completion messages in text mode, improving the user experience for long-running operations.

## Overview

StepLogger enables commands and runtimes to show users what's happening during multi-step operations like creating containers, building images, or starting instances. It automatically handles different output modes (text with spinners vs JSON with silence).

## Key Components

- **StepLogger Interface** (`pkg/steplogger/steplogger.go`): Contract for logging operational steps
- **TextLogger** (`pkg/steplogger/text.go`): Implementation with spinner animations for text output
- **NoOpLogger** (`pkg/steplogger/noop.go`): Silent implementation for JSON mode and tests
- **Context Integration** (`pkg/steplogger/context.go`): Attach/retrieve loggers from context

## Injecting StepLogger into Context

Commands are responsible for creating and injecting the appropriate StepLogger into the context before calling runtime methods:

```go
func (c *myCmd) run(cmd *cobra.Command, args []string) error {
    // Create appropriate logger based on output mode
    var logger steplogger.StepLogger
    if c.output == "json" {
        // No step logging in JSON mode
        logger = steplogger.NewNoOpLogger()
    } else {
        // Use text logger with spinners for text output
        logger = steplogger.NewTextLogger(cmd.ErrOrStderr())
    }
    defer logger.Complete()

    // Attach logger to context
    ctx := steplogger.WithLogger(cmd.Context(), logger)

    // Pass context to runtime methods
    info, err := runtime.Create(ctx, params)
    if err != nil {
        return err
    }

    return nil
}
```

## Logger Selection Rules

- **JSON mode** (`--output json`): Use `steplogger.NewNoOpLogger()` - completely silent, no output
- **Text mode** (default): Use `steplogger.NewTextLogger(cmd.ErrOrStderr())` - displays spinners and messages to stderr

## Important Notes

- Always call `defer logger.Complete()` immediately after creating the logger
- Attach the logger to context using `steplogger.WithLogger(cmd.Context(), logger)`
- Pass the context (not `cmd.Context()`) to all runtime methods
- Output to stderr (`cmd.ErrOrStderr()`) so it doesn't interfere with stdout JSON output

## Using StepLogger in Runtime Methods

All runtime methods that accept a `context.Context` should use the StepLogger for user feedback.

**Note for Runtime Implementers:** You don't need to create or inject the StepLogger - it's already in the context. Simply retrieve it using `steplogger.FromContext(ctx)` and use it as shown below:

```go
func (r *myRuntime) Create(ctx context.Context, params runtime.CreateParams) (runtime.RuntimeInfo, error) {
    logger := steplogger.FromContext(ctx)
    defer logger.Complete()

    // Step 1: Create resources
    logger.Start("Creating workspace directory", "Workspace directory created")
    if err := r.createDirectory(params.Name); err != nil {
        logger.Fail(err)
        return runtime.RuntimeInfo{}, err
    }

    // Step 2: Build image
    logger.Start("Building container image", "Container image built")
    if err := r.buildImage(ctx, params.Name); err != nil {
        logger.Fail(err)
        return runtime.RuntimeInfo{}, err
    }

    // Step 3: Create instance
    logger.Start("Creating instance", "Instance created")
    info, err := r.createInstance(ctx, params)
    if err != nil {
        logger.Fail(err)
        return runtime.RuntimeInfo{}, err
    }

    return info, nil
}
```

## StepLogger Methods

### Start(inProgress, completed string)

Begin a new step with progress and completion messages.

- Automatically completes the previous step if one exists
- `inProgress`: Message shown while the step is running (e.g., "Building container image")
- `completed`: Message shown when the step completes (e.g., "Container image built")

### Complete()

Mark the current step as successfully completed.

- Typically called with `defer` at the start of the method to complete the last step

### Fail(err error)

Mark the current step as failed.

- Displays the error message to the user
- Should be called before returning the error

## Best Practices

1. **Always call `Complete()` with defer** at the start of the method
2. **Use descriptive messages** that inform users about what's happening
3. **Call `Fail()` before returning errors** to show which step failed
4. **Retrieve logger from context** using `steplogger.FromContext(ctx)`
5. **Don't worry about JSON mode** - the NoOpLogger is automatically used when `--output json` is set

## Example Pattern for All Runtime Methods

```go
// Create
func (r *myRuntime) Create(ctx context.Context, params runtime.CreateParams) (runtime.RuntimeInfo, error) {
    logger := steplogger.FromContext(ctx)
    defer logger.Complete()

    logger.Start("Creating resource", "Resource created")
    // ... implementation ...
}

// Start
func (r *myRuntime) Start(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
    logger := steplogger.FromContext(ctx)
    defer logger.Complete()

    logger.Start("Starting instance", "Instance started")
    // ... implementation ...
}

// Stop
func (r *myRuntime) Stop(ctx context.Context, id string) error {
    logger := steplogger.FromContext(ctx)
    defer logger.Complete()

    logger.Start("Stopping instance", "Instance stopped")
    // ... implementation ...
}

// Remove
func (r *myRuntime) Remove(ctx context.Context, id string) error {
    logger := steplogger.FromContext(ctx)
    defer logger.Complete()

    logger.Start("Removing instance", "Instance removed")
    // ... implementation ...
}
```

## Automatic Behavior

- **Text Mode**: Displays animated spinners during operations and completion checkmarks
- **JSON Mode**: Silent - no output to avoid polluting JSON responses
- **Testing**: Use `steplogger.NewNoOpLogger()` or don't attach a logger (defaults to NoOp)

## Testing StepLogger

Create a fake step logger to verify step behavior in tests:

```go
// Create fake logger
type fakeStepLogger struct {
    startCalls    []stepCall
    failCalls     []error
    completeCalls int
}

type stepCall struct {
    inProgress string
    completed  string
}

func (f *fakeStepLogger) Start(inProgress, completed string) {
    f.startCalls = append(f.startCalls, stepCall{inProgress, completed})
}

func (f *fakeStepLogger) Fail(err error) {
    f.failCalls = append(f.failCalls, err)
}

func (f *fakeStepLogger) Complete() {
    f.completeCalls++
}

// Use in tests
func TestCreate_StepLogger(t *testing.T) {
    fakeLogger := &fakeStepLogger{}
    ctx := steplogger.WithLogger(context.Background(), fakeLogger)

    _, err := runtime.Create(ctx, params)

    // Verify step calls
    if len(fakeLogger.startCalls) != 3 {
        t.Errorf("Expected 3 Start() calls, got %d", len(fakeLogger.startCalls))
    }
    if fakeLogger.completeCalls != 1 {
        t.Errorf("Expected 1 Complete() call, got %d", fakeLogger.completeCalls)
    }
}
```

## Reference Implementation

See `pkg/runtime/podman/` for complete examples:
- `create.go` - Multi-step Create operation with 4 steps
- `start.go` - Start operation with verification step
- `stop.go` - Simple single-step Stop operation
- `remove.go` - Remove operation with state checking

See `pkg/runtime/podman/steplogger_test.go` and step logger tests in `create_test.go`, `start_test.go`, `stop_test.go`, `remove_test.go`.

## Related Skills

- `/working-with-runtime-system` - Runtime architecture overview
- `/add-runtime` - Creating new runtimes with StepLogger
- `/implementing-command-patterns` - Command patterns including StepLogger integration

## References

- **StepLogger Interface**: `pkg/steplogger/steplogger.go`
- **TextLogger**: `pkg/steplogger/text.go`
- **NoOpLogger**: `pkg/steplogger/noop.go`
- **Context Integration**: `pkg/steplogger/context.go`
- **Example Commands**: `pkg/cmd/init.go`, `pkg/cmd/workspace_start.go`, `pkg/cmd/workspace_stop.go`, `pkg/cmd/workspace_remove.go`
