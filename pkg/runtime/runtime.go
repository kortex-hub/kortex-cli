// Copyright 2026 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package runtime provides interfaces and types for managing AI agent workspace runtimes.
// A runtime is an execution environment (e.g., container, process) that hosts a workspace instance.
package runtime

import "context"

// Runtime manages the lifecycle of workspace instances in a specific execution environment.
// Implementations might use containers (podman, docker), processes, or other isolation mechanisms.
type Runtime interface {
	// Type returns the runtime type identifier (e.g., "podman", "docker", "process", "fake").
	Type() string

	// Create creates a new runtime instance without starting it.
	// Returns RuntimeInfo with the assigned instance ID and initial state.
	Create(ctx context.Context, params CreateParams) (RuntimeInfo, error)

	// Start starts a previously created runtime instance.
	// Returns updated RuntimeInfo with running state.
	Start(ctx context.Context, id string) (RuntimeInfo, error)

	// Stop stops a running runtime instance without removing it.
	// The instance can be started again later.
	Stop(ctx context.Context, id string) error

	// Remove removes a runtime instance and cleans up all associated resources.
	// The instance must be stopped before removal.
	Remove(ctx context.Context, id string) error

	// Info retrieves current information about a runtime instance.
	Info(ctx context.Context, id string) (RuntimeInfo, error)
}

// CreateParams contains parameters for creating a new runtime instance.
type CreateParams struct {
	// Name is the human-readable name for the instance.
	Name string

	// SourcePath is the absolute path to the workspace source directory.
	SourcePath string

	// ConfigPath is the absolute path to the workspace configuration directory.
	ConfigPath string
}

// RuntimeInfo contains information about a runtime instance.
type RuntimeInfo struct {
	// ID is the runtime-assigned instance identifier.
	ID string

	// State is the current runtime state (e.g., "created", "running", "stopped").
	State string

	// Info contains runtime-specific metadata as key-value pairs.
	// Examples: container_id, pid, created_at, network addresses.
	Info map[string]string
}
