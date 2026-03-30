---
name: working-with-podman-runtime-config
description: Guide to configuring the Podman runtime including image setup, agent configuration, and containerfile generation
argument-hint: ""
---

# Working with Podman Runtime Configuration

The Podman runtime supports configurable image and agent settings through JSON files. This is **runtime-specific configuration** that controls how the Podman container is built and configured.

**What this config system controls:**
- Base container image (Fedora version)
- Packages to install in the container
- Sudo permissions for binaries
- Custom RUN commands during image build
- Agent-specific setup commands
- Terminal command for the agent

**What this does NOT control:**
- Environment variables and mounts injected into workspaces
- See `/working-with-config-system` for workspace configuration (env vars, mounts)

## Overview

The Podman runtime configuration allows customization of the base image, installed packages, sudo permissions, and agent setup through JSON files stored in the runtime's storage directory.

## Key Components

- **Config Interface** (`pkg/runtime/podman/config/config.go`): Interface for managing Podman runtime configuration
- **ImageConfig** (`pkg/runtime/podman/config/types.go`): Base image configuration (Fedora version, packages, sudo binaries, custom RUN commands)
- **AgentConfig** (`pkg/runtime/podman/config/types.go`): Agent-specific configuration (packages, RUN commands, terminal command)
- **Defaults** (`pkg/runtime/podman/config/defaults.go`): Default configurations for image and Claude agent

## Configuration Storage

Configuration files are stored in the runtime's storage directory:

```text
<storage-dir>/runtimes/podman/config/
├── image.json    # Base image configuration
└── claude.json   # Agent-specific configuration (e.g., for Claude Code)
```

## Configuration Files

### image.json - Base Image Configuration

```json
{
  "version": "latest",
  "packages": ["which", "procps-ng", "wget2", "@development-tools", "jq", "gh", "golang", "golangci-lint", "python3", "python3-pip"],
  "sudo": ["/usr/bin/dnf", "/bin/nice", "/bin/kill", "/usr/bin/kill", "/usr/bin/killall"],
  "run_commands": []
}
```

**Fields:**
- `version` (required) - Fedora version tag (e.g., "latest", "40", "41")
- `packages` (optional) - DNF packages to install
- `sudo` (optional) - Absolute paths to binaries the user can run with sudo (creates single `ALLOWED` Cmnd_Alias)
- `run_commands` (optional) - Custom shell commands to execute during image build (before agent setup)

### claude.json - Agent-Specific Configuration

```json
{
  "packages": [],
  "run_commands": [
    "curl -fsSL --proto-redir '-all,https' --tlsv1.3 https://claude.ai/install.sh | bash",
    "mkdir /home/agent/.config"
  ],
  "terminal_command": ["claude"]
}
```

**Fields:**
- `packages` (optional) - Additional packages for the agent (merged with image packages)
- `run_commands` (optional) - Commands to set up the agent (executed after image setup)
- `terminal_command` (required) - Command to launch the agent (must have at least one element)

## Using the Config Interface

```go
import "github.com/kortex-hub/kortex-cli/pkg/runtime/podman/config"

// Create config manager (in Initialize method)
configDir := filepath.Join(storageDir, "config")
cfg, err := config.NewConfig(configDir)
if err != nil {
    return fmt.Errorf("failed to create config: %w", err)
}

// Generate default configs if they don't exist
if err := cfg.GenerateDefaults(); err != nil {
    return fmt.Errorf("failed to generate defaults: %w", err)
}

// Load configurations (in Create method)
imageConfig, err := cfg.LoadImage()
if err != nil {
    return fmt.Errorf("failed to load image config: %w", err)
}

agentConfig, err := cfg.LoadAgent("claude")
if err != nil {
    return fmt.Errorf("failed to load agent config: %w", err)
}
```

## Validation

The config system validates:
- Image version cannot be empty
- Sudo binaries must be absolute paths
- Terminal command must have at least one element
- All fields are optional except `version` (ImageConfig) and `terminal_command` (AgentConfig)

## Default Generation

- Default configs are auto-generated on first runtime initialization
- Existing config files are never overwritten - customizations are preserved
- Default image config includes common development tools and packages
- Default Claude config installs Claude Code from the official install script

## Containerfile Generation

The config system is used to generate Containerfiles dynamically:

```go
import "github.com/kortex-hub/kortex-cli/pkg/runtime/podman"

// Generate Containerfile content from configs
containerfileContent := generateContainerfile(imageConfig, agentConfig)

// Generate sudoers file content from sudo binaries
sudoersContent := generateSudoers(imageConfig.Sudo)
```

The `generateContainerfile` function creates a Containerfile with:
- Base image: `registry.fedoraproject.org/fedora:<version>`
- Merged packages from image and agent configs
- User/group setup (hardcoded as `agent:agent`)
- Sudoers configuration with single `ALLOWED` Cmnd_Alias
- Custom RUN commands from both configs (image commands first, then agent commands)

## Hardcoded Values

These values are not configurable:
- Base image registry: `registry.fedoraproject.org/fedora` (only version tag is configurable)
- Container user: `agent`
- Container group: `agent`
- User UID/GID: Matched to host user's UID/GID at build time

## Design Principles

- Follows interface-based design pattern with unexported implementation
- Uses nested JSON structure for clarity
- Validates all configurations on load to catch errors early
- Separate concerns: base image vs agent-specific settings
- Extensible: easy to add new agent configurations (e.g., `goose.json`, `cursor.json`)

## Related Skills

- `/working-with-config-system` - Workspace configuration (env vars, mounts)
- `/working-with-runtime-system` - Runtime system architecture
- `/add-runtime` - Creating new runtimes

## References

- **Config Interface**: `pkg/runtime/podman/config/config.go`
- **ImageConfig & AgentConfig**: `pkg/runtime/podman/config/types.go`
- **Defaults**: `pkg/runtime/podman/config/defaults.go`
- **Podman Runtime**: `pkg/runtime/podman/`
