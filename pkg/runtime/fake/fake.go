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

// Package fake provides a fake runtime implementation for testing.
package fake

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kortex-hub/kortex-cli/pkg/runtime"
)

// fakeRuntime is an in-memory fake runtime for testing.
type fakeRuntime struct {
	mu        sync.RWMutex
	instances map[string]*instanceState
	nextID    int
}

// instanceState tracks the state of a fake runtime instance.
type instanceState struct {
	id     string
	name   string
	state  string
	info   map[string]string
	source string
	config string
}

// Ensure fakeRuntime implements runtime.Runtime at compile time.
var _ runtime.Runtime = (*fakeRuntime)(nil)

// New creates a new fake runtime instance.
func New() runtime.Runtime {
	return &fakeRuntime{
		instances: make(map[string]*instanceState),
		nextID:    1,
	}
}

// Type returns the runtime type identifier.
func (f *fakeRuntime) Type() string {
	return "fake"
}

// Create creates a new fake runtime instance.
func (f *fakeRuntime) Create(ctx context.Context, params runtime.CreateParams) (runtime.RuntimeInfo, error) {
	if params.Name == "" {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: name is required", runtime.ErrInvalidParams)
	}
	if params.SourcePath == "" {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: source path is required", runtime.ErrInvalidParams)
	}
	if params.ConfigPath == "" {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: config path is required", runtime.ErrInvalidParams)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if instance already exists with same name
	for _, inst := range f.instances {
		if inst.name == params.Name {
			return runtime.RuntimeInfo{}, fmt.Errorf("instance with name %s already exists", params.Name)
		}
	}

	// Generate sequential ID
	id := fmt.Sprintf("fake-%03d", f.nextID)
	f.nextID++

	// Create instance state
	state := &instanceState{
		id:     id,
		name:   params.Name,
		state:  "created",
		source: params.SourcePath,
		config: params.ConfigPath,
		info: map[string]string{
			"created_at": time.Now().Format(time.RFC3339),
			"source":     params.SourcePath,
			"config":     params.ConfigPath,
		},
	}

	f.instances[id] = state

	return runtime.RuntimeInfo{
		ID:    id,
		State: state.state,
		Info:  copyMap(state.info),
	}, nil
}

// Start starts a fake runtime instance.
func (f *fakeRuntime) Start(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	inst, exists := f.instances[id]
	if !exists {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: %s", runtime.ErrInstanceNotFound, id)
	}

	if inst.state == "running" {
		return runtime.RuntimeInfo{}, fmt.Errorf("instance %s is already running", id)
	}

	inst.state = "running"
	inst.info["started_at"] = time.Now().Format(time.RFC3339)

	return runtime.RuntimeInfo{
		ID:    inst.id,
		State: inst.state,
		Info:  copyMap(inst.info),
	}, nil
}

// Stop stops a fake runtime instance.
func (f *fakeRuntime) Stop(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	inst, exists := f.instances[id]
	if !exists {
		return fmt.Errorf("%w: %s", runtime.ErrInstanceNotFound, id)
	}

	if inst.state != "running" {
		return fmt.Errorf("instance %s is not running", id)
	}

	inst.state = "stopped"
	inst.info["stopped_at"] = time.Now().Format(time.RFC3339)

	return nil
}

// Remove removes a fake runtime instance.
func (f *fakeRuntime) Remove(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	inst, exists := f.instances[id]
	if !exists {
		// TODO: The fake runtime is not persistent - each New() creates a separate
		// in-memory instance that doesn't share state. This causes issues in tests
		// where one manager creates instances and another manager (with a new fake
		// runtime) tries to remove them. Consider implementing persistent storage
		// for the fake runtime (e.g., file-based or shared in-memory registry) to
		// better simulate real runtimes (Docker/Podman) which maintain state externally.
		// For now, treat missing instances as already removed (idempotent operation).
		return nil
	}

	if inst.state == "running" {
		return fmt.Errorf("instance %s is still running, stop it first", id)
	}

	delete(f.instances, id)
	return nil
}

// Info retrieves information about a fake runtime instance.
func (f *fakeRuntime) Info(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	inst, exists := f.instances[id]
	if !exists {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: %s", runtime.ErrInstanceNotFound, id)
	}

	return runtime.RuntimeInfo{
		ID:    inst.id,
		State: inst.state,
		Info:  copyMap(inst.info),
	}, nil
}

// copyMap creates a shallow copy of a string map.
func copyMap(m map[string]string) map[string]string {
	result := make(map[string]string, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
