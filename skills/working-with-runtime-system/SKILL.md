---
name: working-with-runtime-system
description: Guide to understanding and working with the kortex-cli runtime system architecture
argument-hint: ""
---

# Working with the Runtime System

The runtime system provides a pluggable architecture for managing workspaces on different container/VM platforms (Podman, MicroVM, Kubernetes, etc.). This skill provides detailed guidance on understanding and working with the runtime system.

## Overview

The runtime system enables kortex-cli to support multiple backend platforms through a common interface. Each runtime implementation handles the platform-specific details of creating, starting, stopping, and managing workspace instances.

## Key Components

- **Runtime Interface** (`pkg/runtime/runtime.go`): Contract all runtimes must implement
- **Registry** (`pkg/runtime/registry.go`): Manages runtime registration and discovery
- **Runtime Implementations** (`pkg/runtime/<runtime-name>/`): Platform-specific packages (e.g., `fake`)
- **Centralized Registration** (`pkg/runtimesetup/register.go`): Automatically registers all available runtimes

## Runtime Registration in Commands

Commands use `runtimesetup.RegisterAll()` to automatically register all available runtimes:

```go
import "github.com/kortex-hub/kortex-cli/pkg/runtimesetup"

// In command preRun
manager, err := instances.NewManager(storageDir)
if err != nil {
    return err
}

// Register all available runtimes
if err := runtimesetup.RegisterAll(manager); err != nil {
    return err
}
```

This automatically registers all runtimes from `pkg/runtimesetup/register.go` that report as available (e.g., only registers Podman if `podman` CLI is installed).

## Optional Runtime Interfaces

Some runtimes may implement additional optional interfaces to provide extended functionality beyond the base Runtime interface. These are checked at runtime using type assertions, allowing runtimes to opt-in to features they support.

### StorageAware Interface

The StorageAware interface enables runtimes to persist data in a dedicated storage directory managed by the registry.

```go
type StorageAware interface {
    Initialize(storageDir string) error
}
```

**How it works:**

When a runtime implements StorageAware, the registry will:
1. Create a directory at `<registry-storage>/<runtime-type>/`
2. Call `Initialize(storageDir)` with the path
3. The runtime can use this directory to persist instance data, configuration, or other state

**Example implementation:**

```go
type myRuntime struct {
    storageDir  string
    storageFile string
    instances   map[string]Instance
}

// Implement StorageAware
func (r *myRuntime) Initialize(storageDir string) error {
    r.storageDir = storageDir
    r.storageFile = filepath.Join(storageDir, "instances.json")

    // Load existing state from disk
    return r.loadFromDisk()
}

func (r *myRuntime) Create(ctx context.Context, params runtime.CreateParams) (runtime.RuntimeInfo, error) {
    // ... create instance logic ...

    // Persist to storage directory
    if err := r.saveToDisk(); err != nil {
        return runtime.RuntimeInfo{}, fmt.Errorf("failed to persist instance: %w", err)
    }

    return info, nil
}
```

### Terminal Interface

The Terminal interface enables interactive terminal sessions for connecting to running instances. This is used by the `terminal` command.

```go
type Terminal interface {
    // Terminal starts an interactive terminal session inside a running instance.
    // The agent parameter is used to load agent-specific configuration for the terminal session.
    // The command is executed with stdin/stdout/stderr connected directly to the user's terminal.
    Terminal(ctx context.Context, agent string, instanceID string, command []string) error
}
```

**Example implementation (Podman runtime):**

```go
func (p *podmanRuntime) Terminal(ctx context.Context, agent string, instanceID string, command []string) error {
    if instanceID == "" {
        return fmt.Errorf("%w: instance ID is required", runtime.ErrInvalidParams)
    }
    if len(command) == 0 {
        return fmt.Errorf("%w: command is required", runtime.ErrInvalidParams)
    }

    // Build podman exec -it <container> <command...>
    args := []string{"exec", "-it", instanceID}
    args = append(args, command...)

    return p.executor.RunInteractive(ctx, args...)
}
```

**How optional interfaces work:**

The Terminal interface follows the same pattern as `StorageAware` - it's optional, and runtimes that don't support interactive sessions simply don't implement it. The instances manager checks for Terminal support at runtime using type assertion:

```go
if terminalRuntime, ok := runtime.(Terminal); ok {
    return terminalRuntime.Terminal(ctx, agent, instanceID, command)
}
return errors.New("runtime does not support terminal sessions")
```

This pattern allows runtimes to provide additional capabilities without requiring all runtimes to implement every possible feature.

## Adding a New Runtime

Use the `/add-runtime` skill which provides step-by-step instructions for creating a new runtime implementation. The `fake` runtime in `pkg/runtime/fake/` serves as a reference implementation.

## Related Skills

- `/add-runtime` - Step-by-step guide to create a new runtime implementation
- `/working-with-steplogger` - Add progress feedback to runtime operations
- `/working-with-podman-runtime-config` - Configure the Podman runtime

## References

- **Runtime Interface**: `pkg/runtime/runtime.go`
- **Registry**: `pkg/runtime/registry.go`
- **Registration**: `pkg/runtimesetup/register.go`
- **Reference Implementation**: `pkg/runtime/fake/`
- **Podman Implementation**: `pkg/runtime/podman/`
