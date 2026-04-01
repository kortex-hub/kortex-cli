---
name: working-with-instances-manager
description: Guide to using the instances manager API for workspace management and project detection
argument-hint: ""
---

# Working with the Instances Manager

The instances manager provides the API for managing workspace instances throughout their lifecycle. This skill covers the manager API and project detection functionality.

## Overview

The instances manager handles:
- Adding and removing workspace instances
- Listing and retrieving instance information
- Starting and stopping instances via runtimes
- Project detection and grouping
- Configuration merging (workspace, project, agent configs)
- Interactive terminal sessions with running instances

## Creating the Manager

In command `preRun`, create the manager from the storage flag:

```go
storageDir, _ := cmd.Flags().GetString("storage")
manager, err := instances.NewManager(storageDir)
if err != nil {
    return fmt.Errorf("failed to create manager: %w", err)
}
```

## Manager API

### Add - Create New Instance

Add a new workspace instance to the manager:

```go
instance, err := instances.NewInstance(instances.NewInstanceParams{
    SourceDir: sourceDir,
    ConfigDir: configDir,
})
if err != nil {
    return fmt.Errorf("failed to create instance: %w", err)
}

addedInstance, err := manager.Add(ctx, instances.AddOptions{
    Instance:        instance,
    RuntimeType:     "fake",
    WorkspaceConfig: workspaceConfig,  // From .kortex/workspace.json
    Project:         "custom-project",  // Optional: overrides auto-detection
    Agent:           "claude",          // Optional: agent name for agent-specific config
})
if err != nil {
    return fmt.Errorf("failed to add instance: %w", err)
}
```

The `Add()` method:
1. Detects project ID (or uses custom override)
2. Loads project config (global `""` + project-specific merged)
3. Loads agent config (if agent name provided)
4. Merges configs: workspace → global → project → agent
5. Reads agent settings files from `<storage-dir>/config/<agent>/` into `map[string][]byte`
6. Passes merged config and agent settings to runtime for injection into workspace

### List - Get All Instances

List all registered workspace instances:

```go
instancesList, err := manager.List()
if err != nil {
    return fmt.Errorf("failed to list instances: %w", err)
}

for _, instance := range instancesList {
    fmt.Printf("ID: %s, State: %s, Project: %s\n",
        instance.ID, instance.State, instance.Project)
}
```

### Get - Retrieve Specific Instance

Get a specific instance by ID:

```go
instance, err := manager.Get(id)
if err != nil {
    if errors.Is(err, instances.ErrInstanceNotFound) {
        return fmt.Errorf("workspace not found: %s", id)
    }
    return fmt.Errorf("instance not found: %w", err)
}

fmt.Printf("Found instance: %s (State: %s)\n", instance.ID, instance.State)
```

### Delete - Remove Instance

Delete an instance from the manager:

```go
err := manager.Delete(id)
if err != nil {
    if errors.Is(err, instances.ErrInstanceNotFound) {
        return fmt.Errorf("workspace not found: %s", id)
    }
    return fmt.Errorf("failed to delete instance: %w", err)
}
```

### Start - Start Instance Runtime

Start a stopped instance:

```go
info, err := manager.Start(ctx, id)
if err != nil {
    if errors.Is(err, instances.ErrInstanceNotFound) {
        return fmt.Errorf("workspace not found: %s", id)
    }
    return fmt.Errorf("failed to start instance: %w", err)
}

fmt.Printf("Started instance: %s (State: %s)\n", info.ID, info.State)
```

### Stop - Stop Instance Runtime

Stop a running instance:

```go
err := manager.Stop(ctx, id)
if err != nil {
    if errors.Is(err, instances.ErrInstanceNotFound) {
        return fmt.Errorf("workspace not found: %s", id)
    }
    return fmt.Errorf("failed to stop instance: %w", err)
}
```

### Terminal - Interactive Terminal Session

Connect to a running instance with an interactive terminal:

```go
err := manager.Terminal(cmd.Context(), id, []string{"bash"})
if err != nil {
    if errors.Is(err, instances.ErrInstanceNotFound) {
        return fmt.Errorf("workspace not found: %s\nUse 'workspace list' to see available workspaces", id)
    }
    return err
}
```

**Terminal Method Behavior:**
- Verifies the instance exists and is in a running state
- Checks if the runtime implements the `runtime.Terminal` interface
- Delegates to the runtime's Terminal implementation
- Returns an error if the instance is not running or runtime doesn't support terminals

**Key Points:**
- Uses a read lock (doesn't modify instance state)
- Command is a slice of strings: `[]string{"bash"}` or `[]string{"claude-code", "--debug"}`
- Returns `ErrInstanceNotFound` if instance doesn't exist
- Returns an error if instance state is not "running"
- Returns an error if the runtime doesn't implement `runtime.Terminal` interface

**Example usage in a command:**

```go
func (w *workspaceTerminalCmd) run(cmd *cobra.Command, args []string) error {
    // Start terminal session with the command extracted in preRun
    err := w.manager.Terminal(cmd.Context(), w.id, w.command)
    if err != nil {
        if errors.Is(err, instances.ErrInstanceNotFound) {
            return fmt.Errorf("workspace not found: %s\nUse 'workspace list' to see available workspaces", w.id)
        }
        return err
    }
    return nil
}
```

## Project Detection and Grouping

Each workspace has a `project` field that enables grouping workspaces belonging to the same project across branches, forks, or subdirectories.

### Project Identifier Detection

The manager automatically detects the project identifier when adding instances:

1. **Git repository with remote**: Uses repository remote URL (without `.git`) plus relative path
   - Checks `upstream` remote first (useful for forks)
   - Falls back to `origin` remote if `upstream` doesn't exist
   - Example: `https://github.com/kortex-hub/kortex-cli/` (at root) or `https://github.com/kortex-hub/kortex-cli/pkg/git` (in subdirectory)

2. **Git repository without remote**: Uses repository root directory plus relative path
   - Example: `/home/user/local-repo` (at root) or `/home/user/local-repo/pkg/utils` (in subdirectory)

3. **Non-git directory**: Uses the source directory path
   - Example: `/tmp/workspace`

### Custom Project Override

Users can override auto-detection with the `--project` flag:

```go
// Add instance with custom project
addedInstance, err := manager.Add(ctx, instances.AddOptions{
    Instance:        instance,
    RuntimeType:     "fake",
    WorkspaceConfig: workspaceConfig,
    Project:         "custom-project-id", // Optional: overrides auto-detection
})
```

### Implementation Details

- **Package**: `pkg/git` provides git repository detection with testable abstractions
- **Detector Interface**: `git.Detector` with `DetectRepository(ctx, dir)` method
- **Executor Pattern**: `git.Executor` abstracts git command execution for testing
- **Manager Integration**: `manager.detectProject()` is called during `Add()` if no custom project is provided

### Testing with Fake Git Detector

```go
// Use fake git detector in tests
gitDetector := newFakeGitDetectorWithRepo(
    "/repo/root",
    "https://github.com/user/repo",
    "pkg/subdir", // relative path
)

manager, _ := newManagerWithFactory(
    storageDir,
    fakeInstanceFactory,
    newFakeGenerator(),
    newTestRegistry(tmpDir),
    gitDetector,
)
```

See `pkg/instances/manager_project_test.go` for comprehensive test examples.

## Error Handling

Common errors from the manager:

```go
// Instance not found
if errors.Is(err, instances.ErrInstanceNotFound) {
    return fmt.Errorf("workspace not found: %s", id)
}

// Runtime not found
if errors.Is(err, runtime.ErrRuntimeNotFound) {
    return fmt.Errorf("runtime not found: %s", runtimeType)
}

// Instance already exists
if errors.Is(err, instances.ErrInstanceExists) {
    return fmt.Errorf("workspace already exists: %s", id)
}
```

## Example: Complete Command Implementation

```go
type myCmd struct {
    manager instances.Manager
}

func (c *myCmd) preRun(cmd *cobra.Command, args []string) error {
    storageDir, _ := cmd.Flags().GetString("storage")

    manager, err := instances.NewManager(storageDir)
    if err != nil {
        return fmt.Errorf("failed to create manager: %w", err)
    }

    // Register runtimes
    if err := runtimesetup.RegisterAll(manager); err != nil {
        return fmt.Errorf("failed to register runtimes: %w", err)
    }

    c.manager = manager
    return nil
}

func (c *myCmd) run(cmd *cobra.Command, args []string) error {
    // Use manager to list instances
    instances, err := c.manager.List()
    if err != nil {
        return fmt.Errorf("failed to list instances: %w", err)
    }

    for _, instance := range instances {
        cmd.Printf("ID: %s, State: %s, Project: %s\n",
            instance.ID, instance.State, instance.Project)
    }

    return nil
}
```

## Related Skills

- `/working-with-config-system` - Configuration merging and multi-level configs
- `/working-with-runtime-system` - Runtime system architecture
- `/implementing-command-patterns` - Command implementation patterns

## References

- **Manager Interface**: `pkg/instances/manager.go`
- **Git Detection**: `pkg/git/`
- **Project Tests**: `pkg/instances/manager_project_test.go`
- **Example Commands**: `pkg/cmd/init.go`, `pkg/cmd/workspace_list.go`, `pkg/cmd/workspace_terminal.go`
