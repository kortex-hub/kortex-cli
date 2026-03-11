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

package runtime

import (
	"fmt"
	"sync"
)

// Registry manages available runtime implementations.
type Registry interface {
	// Register adds a runtime to the registry.
	// Returns an error if a runtime with the same type is already registered.
	Register(runtime Runtime) error

	// Get retrieves a runtime by type.
	// Returns ErrRuntimeNotFound if the runtime type is not registered.
	Get(runtimeType string) (Runtime, error)

	// List returns all registered runtime types.
	List() []string
}

// registry is the concrete implementation of Registry.
type registry struct {
	mu       sync.RWMutex
	runtimes map[string]Runtime
}

// Ensure registry implements Registry interface at compile time.
var _ Registry = (*registry)(nil)

// NewRegistry creates a new empty runtime registry.
func NewRegistry() Registry {
	return &registry{
		runtimes: make(map[string]Runtime),
	}
}

// Register adds a runtime to the registry.
func (r *registry) Register(runtime Runtime) error {
	if runtime == nil {
		return fmt.Errorf("runtime cannot be nil")
	}

	runtimeType := runtime.Type()
	if runtimeType == "" {
		return fmt.Errorf("runtime type cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.runtimes[runtimeType]; exists {
		return fmt.Errorf("runtime already registered: %s", runtimeType)
	}

	r.runtimes[runtimeType] = runtime
	return nil
}

// Get retrieves a runtime by type.
func (r *registry) Get(runtimeType string) (Runtime, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runtime, exists := r.runtimes[runtimeType]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrRuntimeNotFound, runtimeType)
	}

	return runtime, nil
}

// List returns all registered runtime types.
func (r *registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.runtimes))
	for runtimeType := range r.runtimes {
		types = append(types, runtimeType)
	}

	return types
}
