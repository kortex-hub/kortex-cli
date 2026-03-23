# kortex-cli

[![codecov](https://codecov.io/gh/kortex-hub/kortex-cli/branch/main/graph/badge.svg)](https://codecov.io/gh/kortex-hub/kortex-cli)

## Introduction

kortex-cli is a command-line interface for launching and managing AI agents with custom configurations. It provides a unified way to start different agents with specific settings including skills, MCP (Model Context Protocol) server connections, and LLM integrations.

### Supported Agents

- **Claude Code** - Anthropic's official CLI for Claude
- **Goose** - AI agent for development tasks
- **Cursor** - AI-powered code editor agent

### Key Features

- Configure agents with custom skills and capabilities
- Connect to MCP servers for extended functionality
- Integrate with various LLM providers
- Consistent interface across different agent types

## Glossary

### Agent
An AI assistant that can perform tasks autonomously. In kortex-cli, agents are the different AI tools (Claude Code, Goose, Cursor) that can be launched and configured.

### LLM (Large Language Model)
The underlying AI model that powers the agents. Examples include Claude (by Anthropic), GPT (by OpenAI), and other language models.

### MCP (Model Context Protocol)
A standardized protocol for connecting AI agents to external data sources and tools. MCP servers provide agents with additional capabilities like database access, API integrations, or file system operations.

### Skills
Pre-configured capabilities or specialized functions that can be enabled for an agent. Skills extend what an agent can do, such as code review, testing, or specific domain knowledge.

### Workspace
A registered directory containing your project source code and its configuration. Each workspace is tracked by kortex-cli with a unique ID and name for easy management.

## Scenarios

### Managing Workspaces from a UI or Programmatically

This scenario demonstrates how to manage workspaces programmatically using JSON output, which is ideal for UIs, scripts, or automation tools. All commands support the `--output json` (or `-o json`) flag for machine-readable output.

**Step 1: Check existing workspaces**

```bash
$ kortex-cli workspace list -o json
```

```json
{
  "items": []
}
```

Exit code: `0` (success, but no workspaces registered)

**Step 2: Register a new workspace**

```bash
$ kortex-cli init /path/to/project --runtime fake -o json
```

```json
{
  "id": "2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea"
}
```

Exit code: `0` (success)

**Step 3: Register with verbose output to get full details**

```bash
$ kortex-cli init /path/to/another-project --runtime fake -o json -v
```

```json
{
  "id": "f6e5d4c3b2a1098765432109876543210987654321098765432109876543210a",
  "name": "another-project",
  "paths": {
    "source": "/absolute/path/to/another-project",
    "configuration": "/absolute/path/to/another-project/.kortex"
  }
}
```

Exit code: `0` (success)

**Step 4: List all workspaces**

```bash
$ kortex-cli workspace list -o json
```

```json
{
  "items": [
    {
      "id": "2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea",
      "name": "project",
      "paths": {
        "source": "/absolute/path/to/project",
        "configuration": "/absolute/path/to/project/.kortex"
      }
    },
    {
      "id": "f6e5d4c3b2a1098765432109876543210987654321098765432109876543210a",
      "name": "another-project",
      "paths": {
        "source": "/absolute/path/to/another-project",
        "configuration": "/absolute/path/to/another-project/.kortex"
      }
    }
  ]
}
```

Exit code: `0` (success)

**Step 5: Start a workspace**

```bash
$ kortex-cli workspace start 2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea -o json
```

```json
{
  "id": "2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea"
}
```

Exit code: `0` (success)

**Step 6: Stop a workspace**

```bash
$ kortex-cli workspace stop 2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea -o json
```

```json
{
  "id": "2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea"
}
```

Exit code: `0` (success)

**Step 7: Remove a workspace**

```bash
$ kortex-cli workspace remove 2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea -o json
```

```json
{
  "id": "2c5f16046476be368fcada501ac6cdc6bbd34ea80eb9ceb635530c0af64681ea"
}
```

Exit code: `0` (success)

**Step 8: Verify removal**

```bash
$ kortex-cli workspace list -o json
```

```json
{
  "items": [
    {
      "id": "f6e5d4c3b2a1098765432109876543210987654321098765432109876543210a",
      "name": "another-project",
      "paths": {
        "source": "/absolute/path/to/another-project",
        "configuration": "/absolute/path/to/another-project/.kortex"
      }
    }
  ]
}
```

Exit code: `0` (success)

#### Error Handling

All errors are returned in JSON format when using `--output json`, with the error written to **stdout** (not stderr) and a non-zero exit code.

**Error: Non-existent directory**

```bash
$ kortex-cli init /tmp/no-exist --runtime fake -o json
```

```json
{
  "error": "sources directory does not exist: /tmp/no-exist"
}
```

Exit code: `1` (error)

**Error: Workspace not found**

```bash
$ kortex-cli workspace remove unknown-id -o json
```

```json
{
  "error": "workspace not found: unknown-id"
}
```

Exit code: `1` (error)

#### Best Practices for Programmatic Usage

1. **Always check the exit code** to determine success (0) or failure (non-zero)
2. **Parse stdout** for JSON output in both success and error cases
3. **Use verbose mode** with init (`-v`) when you need full workspace details immediately after creation
4. **Handle both success and error JSON structures** in your code:
   - Success responses have specific fields (e.g., `id`, `items`, `name`, `paths`)
   - Error responses always have an `error` field

**Example script pattern:**

```bash
#!/bin/bash

# Register a workspace
output=$(kortex-cli init /path/to/project --runtime fake -o json)
exit_code=$?

if [ $exit_code -eq 0 ]; then
    workspace_id=$(echo "$output" | jq -r '.id')
    echo "Workspace created: $workspace_id"
else
    error_msg=$(echo "$output" | jq -r '.error')
    echo "Error: $error_msg"
    exit 1
fi
```

## Environment Variables

kortex-cli supports environment variables for configuring default behavior.

### `KORTEX_CLI_DEFAULT_RUNTIME`

Sets the default runtime to use when registering a workspace with the `init` command.

**Usage:**

```bash
export KORTEX_CLI_DEFAULT_RUNTIME=fake
kortex-cli init /path/to/project
```

**Priority:**

The runtime is determined in the following order (highest to lowest priority):

1. `--runtime` flag (if specified)
2. `KORTEX_CLI_DEFAULT_RUNTIME` environment variable (if set)
3. Error if neither is set (runtime is required)

**Example:**

```bash
# Set the default runtime for the current shell session
export KORTEX_CLI_DEFAULT_RUNTIME=fake

# Register a workspace using the environment variable
kortex-cli init /path/to/project

# Override the environment variable with the flag
kortex-cli init /path/to/another-project --runtime podman
```

**Notes:**

- The runtime parameter is mandatory when registering workspaces
- If neither the flag nor the environment variable is set, the `init` command will fail with an error
- Supported runtime types depend on the available runtime implementations
- Setting this environment variable is useful for automation scripts or when you consistently use the same runtime

### `KORTEX_CLI_STORAGE`

Sets the default storage directory where kortex-cli stores its data files.

**Usage:**

```bash
export KORTEX_CLI_STORAGE=/custom/path/to/storage
kortex-cli init /path/to/project --runtime fake
```

**Priority:**

The storage directory is determined in the following order (highest to lowest priority):

1. `--storage` flag (if specified)
2. `KORTEX_CLI_STORAGE` environment variable (if set)
3. Default: `$HOME/.kortex-cli`

**Example:**

```bash
# Set a custom storage directory
export KORTEX_CLI_STORAGE=/var/lib/kortex

# All commands will use this storage directory
kortex-cli init /path/to/project --runtime fake
kortex-cli list

# Override the environment variable with the flag
kortex-cli list --storage /tmp/kortex-storage
```

## Runtimes

### Podman Runtime

The Podman runtime provides a container-based development environment for workspaces. It creates an isolated environment with all necessary tools pre-installed and configured.

#### Container Image

**Base Image:** `registry.fedoraproject.org/fedora:latest`

The Podman runtime builds a custom container image based on Fedora Linux, providing a stable and up-to-date foundation for development work.

#### Installed Packages

The runtime includes a comprehensive development toolchain:

- **Core Utilities:**
  - `which` - Command location utility
  - `procps-ng` - Process management utilities
  - `wget2` - Advanced file downloader

- **Development Tools:**
  - `@development-tools` - Complete development toolchain (gcc, make, etc.)
  - `jq` - JSON processor
  - `gh` - GitHub CLI

- **Language Support:**
  - `golang` - Go programming language
  - `golangci-lint` - Go linter
  - `python3` - Python 3 interpreter
  - `python3-pip` - Python package manager

#### User and Permissions

The container runs as a non-root user named `claude` with the following configuration:

- **User:** `claude`
- **UID/GID:** Matches the host user's UID and GID for seamless file permissions
- **Home Directory:** `/home/claude`

**Sudo Permissions:**

The `claude` user has limited sudo access with no password required (`NOPASSWD`) for:

- **Package Management:**
  - `/usr/bin/dnf` - Install, update, and manage packages

- **Process Management:**
  - `/bin/nice` - Run programs with modified scheduling priority
  - `/bin/kill`, `/usr/bin/kill` - Send signals to processes
  - `/usr/bin/killall` - Kill processes by name

All other sudo commands are explicitly denied for security.

#### AI Agent

**Claude Code** is installed as the default AI agent using the official installation script from `claude.ai/install.sh`. This provides:

- Full Claude Code CLI capabilities
- Integrated development assistance
- Access to Claude's latest features

The agent runs within the container environment and has access to the mounted workspace sources and dependencies.

#### Working Directory

The container's working directory is set to `/workspace/sources`, which is where your project source code is mounted. This ensures that the agent and all tools operate within your project context.

#### Example Usage

```bash
# Register a workspace with the Podman runtime
kortex-cli init /path/to/project --runtime podman

# Start the workspace (builds image and starts container)
kortex-cli start <workspace-id>
```

The first time you start a workspace, the Podman runtime will:
1. Create a Containerfile with the configuration above
2. Build a custom image (tagged as `kortex-cli-<workspace-name>`)
3. Create a container with your source code mounted
4. Start the container and make it ready for use

## Workspace Configuration

Each workspace can optionally include a configuration file that customizes the environment and mount behavior for that specific workspace. The configuration is stored in a `workspace.json` file within the workspace's configuration directory (typically `.kortex` in the sources directory).

### Configuration File Location

By default, workspace configuration is stored at:
```text
<sources-directory>/.kortex/workspace.json
```

The configuration directory (containing `workspace.json`) can be customized using the `--workspace-configuration` flag when registering a workspace with `init`. The flag accepts a directory path, not the file path itself.

### Configuration Structure

The `workspace.json` file uses a nested JSON structure:

```json
{
  "environment": [
    {
      "name": "DEBUG",
      "value": "true"
    },
    {
      "name": "API_KEY",
      "secret": "github-token"
    }
  ],
  "mounts": {
    "dependencies": ["../main", "../../lib"],
    "configs": [".ssh", ".gitconfig"]
  }
}
```

### Environment Variables

Define environment variables that will be set in the workspace runtime environment.

**Structure:**
```json
{
  "environment": [
    {
      "name": "VAR_NAME",
      "value": "hardcoded-value"
    },
    {
      "name": "SECRET_VAR",
      "secret": "secret-reference"
    }
  ]
}
```

**Fields:**
- `name` (required) - Environment variable name
  - Must be a valid Unix environment variable name
  - Must start with a letter or underscore
  - Can contain letters, digits, and underscores
- `value` (optional) - Hardcoded value for the variable
  - Mutually exclusive with `secret`
  - Empty strings are allowed
- `secret` (optional) - Reference to a secret containing the value
  - Mutually exclusive with `value`
  - Cannot be empty

**Validation Rules:**
- Variable name cannot be empty
- Exactly one of `value` or `secret` must be defined
- Variable names must follow Unix conventions (e.g., `DEBUG`, `API_KEY`, `MY_VAR_123`)
- Invalid names include those starting with digits (`1INVALID`) or containing special characters (`INVALID-NAME`, `INVALID@NAME`)

### Mount Paths

Configure additional directories to mount in the workspace runtime.

**Structure:**
```json
{
  "mounts": {
    "dependencies": ["../main"],
    "configs": [".claude", ".gitconfig"]
  }
}
```

**Fields:**
- `dependencies` (optional) - Additional source directories to mount
  - Paths are relative to the workspace sources directory
  - Useful for git worktrees
- `configs` (optional) - Configuration directories to mount from the user's home directory
  - Paths are relative to `$HOME`
  - Useful for sharing Git configs, or tool configurations

**Validation Rules:**
- All paths must be relative (not absolute)
- Paths cannot be empty
- Absolute paths like `/absolute/path` are rejected

### Configuration Validation

When you register a workspace with `kortex-cli init`, the configuration is automatically validated. If `workspace.json` exists and contains invalid data, the registration will fail with a descriptive error message.

**Example - Invalid configuration (both value and secret set):**
```bash
$ kortex-cli init /path/to/project --runtime fake
```
```text
Error: workspace configuration validation failed: invalid workspace configuration:
environment variable "API_KEY" (index 0) has both value and secret set
```

**Example - Invalid configuration (absolute path in mounts):**
```bash
$ kortex-cli init /path/to/project --runtime fake
```
```text
Error: workspace configuration validation failed: invalid workspace configuration:
dependency mount "/absolute/path" (index 0) must be a relative path
```

### Configuration Examples

**Basic environment variables:**
```json
{
  "environment": [
    {
      "name": "NODE_ENV",
      "value": "development"
    },
    {
      "name": "DEBUG",
      "value": "true"
    }
  ]
}
```

**Using secrets:**
```json
{
  "environment": [
    {
      "name": "API_TOKEN",
      "secret": "github-api-token"
    }
  ]
}
```

**git worktree:**
```json
{
  "mounts": {
    "dependencies": [
      "../main"
    ]
  }
}
```

**Sharing user configurations:**
```json
{
  "mounts": {
    "configs": [
      ".claude",
      ".gitconfig",
      ".kube/config"
    ]
  }
}
```

**Complete configuration:**
```json
{
  "environment": [
    {
      "name": "NODE_ENV",
      "value": "development"
    },
    {
      "name": "DATABASE_URL",
      "secret": "local-db-url"
    }
  ],
  "mounts": {
    "dependencies": ["../main"],
    "configs": [".claude", ".gitconfig"]
  }
}
```

### Notes

- Configuration is **optional** - workspaces can be registered without a `workspace.json` file
- The configuration file is validated only when it exists
- Validation errors are caught early during workspace registration (`init` command)
- All validation rules are enforced to prevent runtime errors
- The configuration model is imported from the `github.com/kortex-hub/kortex-cli-api/workspace-configuration/go` package for consistency across tools

## Commands

### `init` - Register a New Workspace

Registers a new workspace with kortex-cli, making it available for agent launch and configuration.

#### Usage

```bash
kortex-cli init [sources-directory] [flags]
```

#### Arguments

- `sources-directory` - Path to the directory containing your project source files (optional, defaults to current directory `.`)

#### Flags

- `--runtime, -r <type>` - Runtime to use for the workspace (required if `KORTEX_CLI_DEFAULT_RUNTIME` is not set)
- `--workspace-configuration <path>` - Directory for workspace configuration files (default: `<sources-directory>/.kortex`)
- `--name, -n <name>` - Human-readable name for the workspace (default: generated from sources directory)
- `--project, -p <identifier>` - Custom project identifier to override auto-detection (default: auto-detected from git repository or source directory)
- `--verbose, -v` - Show detailed output including all workspace information
- `--output, -o <format>` - Output format (supported: `json`)
- `--storage <path>` - Storage directory for kortex-cli data (default: `$HOME/.kortex-cli`)

#### Examples

**Register the current directory:**
```bash
kortex-cli init --runtime fake
```
Output: `a1b2c3d4e5f6...` (workspace ID)

**Register a specific directory:**
```bash
kortex-cli init /path/to/myproject --runtime fake
```

**Register with a custom name:**
```bash
kortex-cli init /path/to/myproject --runtime fake --name "my-awesome-project"
```

**Register with a custom project identifier:**
```bash
kortex-cli init /path/to/myproject --runtime fake --project "my project"
```

**Register with custom configuration location:**
```bash
kortex-cli init /path/to/myproject --runtime fake --workspace-configuration /path/to/config
```

**View detailed output:**
```bash
kortex-cli init --runtime fake --verbose
```
Output:
```text
Registered workspace:
  ID: a1b2c3d4e5f6...
  Name: myproject
  Sources directory: /absolute/path/to/myproject
  Configuration directory: /absolute/path/to/myproject/.kortex
```

**JSON output (default - ID only):**
```bash
kortex-cli init /path/to/myproject --runtime fake --output json
```
Output:
```json
{
  "id": "a1b2c3d4e5f6..."
}
```

**JSON output with verbose flag (full workspace details):**
```bash
kortex-cli init /path/to/myproject --runtime fake --output json --verbose
```
Output:
```json
{
  "id": "a1b2c3d4e5f6...",
  "name": "myproject",
  "paths": {
    "source": "/absolute/path/to/myproject",
    "configuration": "/absolute/path/to/myproject/.kortex"
  }
}
```

**JSON output with short flags:**
```bash
kortex-cli init -r fake -o json -v
```

#### Workspace Naming

- If `--name` is not provided, the name is automatically generated from the last component of the sources directory path
- If a workspace with the same name already exists, kortex-cli automatically appends an increment (`-2`, `-3`, etc.) to ensure uniqueness

**Examples:**
```bash
# First workspace in /home/user/project
kortex-cli init /home/user/project --runtime fake
# Name: "project"

# Second workspace with the same directory name
kortex-cli init /home/user/another-location/project --runtime fake --name "project"
# Name: "project-2"

# Third workspace with the same name
kortex-cli init /tmp/project --runtime fake --name "project"
# Name: "project-3"
```

#### Project Detection

When registering a workspace, kortex-cli automatically detects and stores a project identifier. This allows grouping workspaces that belong to the same project, even across different branches, forks, or subdirectories.

**The project is determined using the following rules:**

**1. Git repository with remote URL**

The project is the repository remote URL (without `.git` suffix) plus the workspace's relative path within the repository:

- **At repository root**: `https://github.com/user/repo/`
- **In subdirectory**: `https://github.com/user/repo/sub/path`

**Remote priority:**
1. `upstream` remote is checked first (useful for forks)
2. `origin` remote is used if `upstream` doesn't exist
3. If neither exists, falls back to local repository path (see below)

**Example - Fork with upstream:**
```bash
# Repository setup:
# upstream: https://github.com/kortex-hub/kortex-cli.git
# origin:   https://github.com/myuser/kortex-cli.git (fork)

# Workspace at repository root
kortex-cli init /home/user/kortex-cli --runtime fake
# Project: https://github.com/kortex-hub/kortex-cli/

# Workspace in subdirectory
kortex-cli init /home/user/kortex-cli/pkg/git --runtime fake
# Project: https://github.com/kortex-hub/kortex-cli/pkg/git
```

This ensures all forks and branches of the same upstream repository are grouped together.

**2. Git repository without remote**

The project is the repository root directory path plus the workspace's relative path:

- **At repository root**: `/home/user/my-local-repo`
- **In subdirectory**: `/home/user/my-local-repo/sub/path`

**Example - Local repository:**
```bash
# Workspace at repository root
kortex-cli init /home/user/local-repo --runtime fake
# Project: /home/user/local-repo

# Workspace in subdirectory
kortex-cli init /home/user/local-repo/pkg/utils --runtime fake
# Project: /home/user/local-repo/pkg/utils
```

**3. Non-git directory**

The project is the workspace source directory path:

**Example - Regular directory:**
```bash
kortex-cli init /tmp/workspace --runtime fake
# Project: /tmp/workspace
```

**Benefits:**

- **Cross-branch grouping**: Workspaces in different git worktrees or branches of the same repository share the same project
- **Fork grouping**: Forks reference the upstream repository, grouping all contributors working on the same project
- **Subdirectory support**: Monorepo subdirectories are tracked with their full path for precise identification
- **Custom override**: Use `--project` flag to manually group workspaces under a custom identifier (e.g., "client-project")
- **Future filtering**: The project field enables filtering and grouping commands (e.g., list all workspaces for a specific project)

#### Notes

- **Runtime is required**: You must specify a runtime using either the `--runtime` flag or the `KORTEX_CLI_DEFAULT_RUNTIME` environment variable
- **Project auto-detection**: The project identifier is automatically detected from git repository information or source directory path. Use `--project` flag to override with a custom identifier
- All directory paths are converted to absolute paths for consistency
- The workspace ID is a unique identifier generated automatically
- Workspaces can be listed using the `workspace list` command
- The default configuration directory (`.kortex`) is created inside the sources directory unless specified otherwise
- JSON output format is useful for scripting and automation
- Without `--verbose`, JSON output returns only the workspace ID
- With `--verbose`, JSON output includes full workspace details (ID, name, paths)
- **JSON error handling**: When `--output json` is used, errors are written to stdout (not stderr) in JSON format, and the CLI exits with code 1. Always check the exit code to determine success/failure

### `workspace list` - List All Registered Workspaces

Lists all workspaces that have been registered with kortex-cli. Also available as the shorter alias `list`.

#### Usage

```bash
kortex-cli workspace list [flags]
kortex-cli list [flags]
```

#### Flags

- `--output, -o <format>` - Output format (supported: `json`)
- `--storage <path>` - Storage directory for kortex-cli data (default: `$HOME/.kortex-cli`)

#### Examples

**List all workspaces (human-readable format):**
```bash
kortex-cli workspace list
```
Output:
```text
ID: a1b2c3d4e5f6...
  Name: myproject
  Sources: /absolute/path/to/myproject
  Configuration: /absolute/path/to/myproject/.kortex

ID: f6e5d4c3b2a1...
  Name: another-project
  Sources: /absolute/path/to/another-project
  Configuration: /absolute/path/to/another-project/.kortex
```

**Use the short alias:**
```bash
kortex-cli list
```

**List workspaces in JSON format:**
```bash
kortex-cli workspace list --output json
```
Output:
```json
{
  "items": [
    {
      "id": "a1b2c3d4e5f6...",
      "name": "myproject",
      "paths": {
        "source": "/absolute/path/to/myproject",
        "configuration": "/absolute/path/to/myproject/.kortex"
      }
    },
    {
      "id": "f6e5d4c3b2a1...",
      "name": "another-project",
      "paths": {
        "source": "/absolute/path/to/another-project",
        "configuration": "/absolute/path/to/another-project/.kortex"
      }
    }
  ]
}
```

**List with short flag:**
```bash
kortex-cli list -o json
```

#### Notes

- When no workspaces are registered, the command displays "No workspaces registered"
- The JSON output format is useful for scripting and automation
- All paths are displayed as absolute paths for consistency
- **JSON error handling**: When `--output json` is used, errors are written to stdout (not stderr) in JSON format, and the CLI exits with code 1. Always check the exit code to determine success/failure

### `workspace start` - Start a Workspace

Starts a registered workspace by its ID. Also available as the shorter alias `start`.

#### Usage

```bash
kortex-cli workspace start ID [flags]
kortex-cli start ID [flags]
```

#### Arguments

- `ID` - The unique identifier of the workspace to start (required)

#### Flags

- `--output, -o <format>` - Output format (supported: `json`)
- `--storage <path>` - Storage directory for kortex-cli data (default: `$HOME/.kortex-cli`)

#### Examples

**Start a workspace by ID:**
```bash
kortex-cli workspace start a1b2c3d4e5f6...
```
Output: `a1b2c3d4e5f6...` (ID of started workspace)

**Use the short alias:**
```bash
kortex-cli start a1b2c3d4e5f6...
```

**View workspace IDs before starting:**
```bash
# First, list all workspaces to find the ID
kortex-cli list

# Then start the desired workspace
kortex-cli start a1b2c3d4e5f6...
```

**JSON output:**
```bash
kortex-cli workspace start a1b2c3d4e5f6... --output json
```
Output:
```json
{
  "id": "a1b2c3d4e5f6..."
}
```

**JSON output with short flag:**
```bash
kortex-cli start a1b2c3d4e5f6... -o json
```

#### Error Handling

**Workspace not found (text format):**
```bash
kortex-cli start invalid-id
```
Output:
```text
Error: workspace not found: invalid-id
Use 'workspace list' to see available workspaces
```

**Workspace not found (JSON format):**
```bash
kortex-cli start invalid-id --output json
```
Output:
```json
{
  "error": "workspace not found: invalid-id"
}
```

#### Notes

- The workspace ID is required and can be obtained using the `workspace list` or `list` command
- Starting a workspace launches its associated runtime instance
- Upon successful start, the command outputs the ID of the started workspace
- The workspace runtime state is updated to reflect that it's running
- JSON output format is useful for scripting and automation
- When using `--output json`, errors are also returned in JSON format for consistent parsing
- **JSON error handling**: When `--output json` is used, errors are written to stdout (not stderr) in JSON format, and the CLI exits with code 1. Always check the exit code to determine success/failure

### `workspace stop` - Stop a Workspace

Stops a running workspace by its ID. Also available as the shorter alias `stop`.

#### Usage

```bash
kortex-cli workspace stop ID [flags]
kortex-cli stop ID [flags]
```

#### Arguments

- `ID` - The unique identifier of the workspace to stop (required)

#### Flags

- `--output, -o <format>` - Output format (supported: `json`)
- `--storage <path>` - Storage directory for kortex-cli data (default: `$HOME/.kortex-cli`)

#### Examples

**Stop a workspace by ID:**
```bash
kortex-cli workspace stop a1b2c3d4e5f6...
```
Output: `a1b2c3d4e5f6...` (ID of stopped workspace)

**Use the short alias:**
```bash
kortex-cli stop a1b2c3d4e5f6...
```

**View workspace IDs before stopping:**
```bash
# First, list all workspaces to find the ID
kortex-cli list

# Then stop the desired workspace
kortex-cli stop a1b2c3d4e5f6...
```

**JSON output:**
```bash
kortex-cli workspace stop a1b2c3d4e5f6... --output json
```
Output:
```json
{
  "id": "a1b2c3d4e5f6..."
}
```

**JSON output with short flag:**
```bash
kortex-cli stop a1b2c3d4e5f6... -o json
```

#### Error Handling

**Workspace not found (text format):**
```bash
kortex-cli stop invalid-id
```
Output:
```text
Error: workspace not found: invalid-id
Use 'workspace list' to see available workspaces
```

**Workspace not found (JSON format):**
```bash
kortex-cli stop invalid-id --output json
```
Output:
```json
{
  "error": "workspace not found: invalid-id"
}
```

#### Notes

- The workspace ID is required and can be obtained using the `workspace list` or `list` command
- Stopping a workspace stops its associated runtime instance
- Upon successful stop, the command outputs the ID of the stopped workspace
- The workspace runtime state is updated to reflect that it's stopped
- JSON output format is useful for scripting and automation
- When using `--output json`, errors are also returned in JSON format for consistent parsing
- **JSON error handling**: When `--output json` is used, errors are written to stdout (not stderr) in JSON format, and the CLI exits with code 1. Always check the exit code to determine success/failure

### `workspace remove` - Remove a Workspace

Removes a registered workspace by its ID. Also available as the shorter alias `remove`.

#### Usage

```bash
kortex-cli workspace remove ID [flags]
kortex-cli remove ID [flags]
```

#### Arguments

- `ID` - The unique identifier of the workspace to remove (required)

#### Flags

- `--output, -o <format>` - Output format (supported: `json`)
- `--storage <path>` - Storage directory for kortex-cli data (default: `$HOME/.kortex-cli`)

#### Examples

**Remove a workspace by ID:**
```bash
kortex-cli workspace remove a1b2c3d4e5f6...
```
Output: `a1b2c3d4e5f6...` (ID of removed workspace)

**Use the short alias:**
```bash
kortex-cli remove a1b2c3d4e5f6...
```

**View workspace IDs before removing:**
```bash
# First, list all workspaces to find the ID
kortex-cli list

# Then remove the desired workspace
kortex-cli remove a1b2c3d4e5f6...
```

**JSON output:**
```bash
kortex-cli workspace remove a1b2c3d4e5f6... --output json
```
Output:
```json
{
  "id": "a1b2c3d4e5f6..."
}
```

**JSON output with short flag:**
```bash
kortex-cli remove a1b2c3d4e5f6... -o json
```

#### Error Handling

**Workspace not found (text format):**
```bash
kortex-cli remove invalid-id
```
Output:
```text
Error: workspace not found: invalid-id
Use 'workspace list' to see available workspaces
```

**Workspace not found (JSON format):**
```bash
kortex-cli remove invalid-id --output json
```
Output:
```json
{
  "error": "workspace not found: invalid-id"
}
```

#### Notes

- The workspace ID is required and can be obtained using the `workspace list` or `list` command
- Removing a workspace only unregisters it from kortex-cli; it does not delete any files from the sources or configuration directories
- If the workspace ID is not found, the command will fail with a helpful error message
- Upon successful removal, the command outputs the ID of the removed workspace
- JSON output format is useful for scripting and automation
- When using `--output json`, errors are also returned in JSON format for consistent parsing
- **JSON error handling**: When `--output json` is used, errors are written to stdout (not stderr) in JSON format, and the CLI exits with code 1. Always check the exit code to determine success/failure
