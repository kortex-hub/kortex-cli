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
	"context"
	"errors"
	"fmt"
	"testing"
)

// fakeRuntime is a minimal Runtime implementation for testing.
type fakeRuntime struct {
	typeID string
}

func (f *fakeRuntime) Type() string {
	return f.typeID
}

func (f *fakeRuntime) Create(ctx context.Context, params CreateParams) (RuntimeInfo, error) {
	return RuntimeInfo{}, nil
}

func (f *fakeRuntime) Start(ctx context.Context, id string) (RuntimeInfo, error) {
	return RuntimeInfo{}, nil
}

func (f *fakeRuntime) Stop(ctx context.Context, id string) error {
	return nil
}

func (f *fakeRuntime) Remove(ctx context.Context, id string) error {
	return nil
}

func (f *fakeRuntime) Info(ctx context.Context, id string) (RuntimeInfo, error) {
	return RuntimeInfo{}, nil
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	rt := &fakeRuntime{typeID: "test-runtime"}

	// Register the runtime
	err := reg.Register(rt)
	if err != nil {
		t.Fatalf("Failed to register runtime: %v", err)
	}

	// Retrieve the runtime
	retrieved, err := reg.Get("test-runtime")
	if err != nil {
		t.Fatalf("Failed to get runtime: %v", err)
	}

	if retrieved.Type() != "test-runtime" {
		t.Errorf("Expected runtime type 'test-runtime', got '%s'", retrieved.Type())
	}
}

func TestRegistry_DuplicateRegistration(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	rt1 := &fakeRuntime{typeID: "test-runtime"}
	rt2 := &fakeRuntime{typeID: "test-runtime"}

	// Register first runtime
	err := reg.Register(rt1)
	if err != nil {
		t.Fatalf("Failed to register first runtime: %v", err)
	}

	// Try to register duplicate
	err = reg.Register(rt2)
	if err == nil {
		t.Fatal("Expected error when registering duplicate runtime, got nil")
	}

	expectedMsg := "runtime already registered: test-runtime"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRegistry_GetUnknownRuntime(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	// Try to get non-existent runtime
	_, err := reg.Get("unknown-runtime")
	if err == nil {
		t.Fatal("Expected error when getting unknown runtime, got nil")
	}

	if !errors.Is(err, ErrRuntimeNotFound) {
		t.Errorf("Expected ErrRuntimeNotFound, got %v", err)
	}
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	// Empty registry
	types := reg.List()
	if len(types) != 0 {
		t.Errorf("Expected empty list, got %d types", len(types))
	}

	// Register multiple runtimes
	rt1 := &fakeRuntime{typeID: "runtime-1"}
	rt2 := &fakeRuntime{typeID: "runtime-2"}

	if err := reg.Register(rt1); err != nil {
		t.Fatalf("Failed to register runtime-1: %v", err)
	}
	if err := reg.Register(rt2); err != nil {
		t.Fatalf("Failed to register runtime-2: %v", err)
	}

	// List should contain both
	types = reg.List()
	if len(types) != 2 {
		t.Errorf("Expected 2 types, got %d", len(types))
	}

	// Check both types are present (order not guaranteed)
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	if !typeMap["runtime-1"] || !typeMap["runtime-2"] {
		t.Errorf("Expected both runtime-1 and runtime-2 in list, got %v", types)
	}
}

func TestRegistry_RegisterNil(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	err := reg.Register(nil)
	if err == nil {
		t.Fatal("Expected error when registering nil runtime, got nil")
	}

	expectedMsg := "runtime cannot be nil"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRegistry_RegisterEmptyType(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	rt := &fakeRuntime{typeID: ""}

	err := reg.Register(rt)
	if err == nil {
		t.Fatal("Expected error when registering runtime with empty type, got nil")
	}

	expectedMsg := "runtime type cannot be empty"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestRegistry_ThreadSafety(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	// Spawn multiple goroutines that register, get, and list concurrently
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		i := i // capture loop variable
		go func() {
			defer func() { done <- true }()

			// Register a unique runtime
			rt := &fakeRuntime{typeID: fmt.Sprintf("runtime-%d", i)}
			if err := reg.Register(rt); err != nil {
				t.Errorf("Failed to register runtime-%d: %v", i, err)
				return
			}

			// Try to get it
			retrieved, err := reg.Get(fmt.Sprintf("runtime-%d", i))
			if err != nil {
				t.Errorf("Failed to get runtime-%d: %v", i, err)
				return
			}

			if retrieved.Type() != fmt.Sprintf("runtime-%d", i) {
				t.Errorf("Wrong runtime type: expected runtime-%d, got %s", i, retrieved.Type())
			}

			// List all runtimes
			_ = reg.List()
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify all runtimes were registered
	types := reg.List()
	if len(types) != numGoroutines {
		t.Errorf("Expected %d registered runtimes, got %d", numGoroutines, len(types))
	}
}
