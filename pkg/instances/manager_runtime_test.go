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

package instances

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/kortex-hub/kortex-cli/pkg/runtime"
)

// failableRuntime is a fake runtime that can be configured to fail on specific operations
type failableRuntime struct {
	mu             sync.RWMutex
	instances      map[string]*runtimeInstance
	nextID         int
	failOnCreate   bool
	failOnStart    bool
	failOnStop     bool
	failOnRemove   bool
	failOnInfo     bool
	createErr      error
	startErr       error
	stopErr        error
	removeErr      error
	infoErr        error
	startCallCount int
	stopCallCount  int
}

type runtimeInstance struct {
	id     string
	name   string
	state  string
	source string
	config string
}

var _ runtime.Runtime = (*failableRuntime)(nil)

func newFailableRuntime() *failableRuntime {
	return &failableRuntime{
		instances: make(map[string]*runtimeInstance),
		nextID:    1,
	}
}

func (f *failableRuntime) Type() string {
	return "failable"
}

func (f *failableRuntime) Create(ctx context.Context, params runtime.CreateParams) (runtime.RuntimeInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failOnCreate {
		if f.createErr != nil {
			return runtime.RuntimeInfo{}, f.createErr
		}
		return runtime.RuntimeInfo{}, errors.New("create failed")
	}

	id := fmt.Sprintf("failable-%03d", f.nextID)
	f.nextID++

	inst := &runtimeInstance{
		id:     id,
		name:   params.Name,
		state:  "created",
		source: params.SourcePath,
		config: params.ConfigPath,
	}

	f.instances[id] = inst

	return runtime.RuntimeInfo{
		ID:    id,
		State: inst.state,
		Info:  map[string]string{"source": inst.source, "config": inst.config},
	}, nil
}

func (f *failableRuntime) Start(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.startCallCount++

	if f.failOnStart {
		if f.startErr != nil {
			return runtime.RuntimeInfo{}, f.startErr
		}
		return runtime.RuntimeInfo{}, errors.New("start failed")
	}

	inst, exists := f.instances[id]
	if !exists {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: %s", runtime.ErrInstanceNotFound, id)
	}

	inst.state = "running"

	return runtime.RuntimeInfo{
		ID:    inst.id,
		State: inst.state,
		Info:  map[string]string{"source": inst.source, "config": inst.config},
	}, nil
}

func (f *failableRuntime) Stop(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.stopCallCount++

	if f.failOnStop {
		if f.stopErr != nil {
			return f.stopErr
		}
		return errors.New("stop failed")
	}

	inst, exists := f.instances[id]
	if !exists {
		return fmt.Errorf("%w: %s", runtime.ErrInstanceNotFound, id)
	}

	inst.state = "stopped"
	return nil
}

func (f *failableRuntime) Remove(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.failOnRemove {
		if f.removeErr != nil {
			return f.removeErr
		}
		return errors.New("remove failed")
	}

	_, exists := f.instances[id]
	if !exists {
		return fmt.Errorf("%w: %s", runtime.ErrInstanceNotFound, id)
	}

	delete(f.instances, id)
	return nil
}

func (f *failableRuntime) Info(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.failOnInfo {
		if f.infoErr != nil {
			return runtime.RuntimeInfo{}, f.infoErr
		}
		return runtime.RuntimeInfo{}, errors.New("info failed")
	}

	inst, exists := f.instances[id]
	if !exists {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: %s", runtime.ErrInstanceNotFound, id)
	}

	return runtime.RuntimeInfo{
		ID:    inst.id,
		State: inst.state,
		Info:  map[string]string{"source": inst.source, "config": inst.config},
	}, nil
}

func (f *failableRuntime) getState(id string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	inst, exists := f.instances[id]
	if !exists {
		return ""
	}
	return inst.state
}

func (f *failableRuntime) instanceExists(id string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	_, exists := f.instances[id]
	return exists
}

func TestManager_Start_FailureRollback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("rolls back runtime state when saveInstances fails", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()

		// Create a registry with the failable runtime
		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		// Add an instance
		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		runtimeID := added.GetRuntimeData().InstanceID

		// Verify initial state is "created"
		if state := fakeRT.getState(runtimeID); state != "created" {
			t.Errorf("Initial state = %v, want 'created'", state)
		}

		// Make the storage file read-only to force saveInstances to fail
		storageFile := filepath.Join(tmpDir, DefaultStorageFileName)
		if err := makeReadOnly(storageFile); err != nil {
			t.Fatalf("Failed to make storage file read-only: %v", err)
		}
		defer makeWritable(storageFile)

		// Attempt to start - should fail to save
		err = manager.Start(ctx, added.GetID())
		if err == nil {
			t.Fatal("Start() expected error when saveInstances fails, got nil")
		}

		// Verify runtime state was rolled back to "stopped"
		// (Start succeeded, then saveInstances failed, so rollback called Stop)
		if state := fakeRT.getState(runtimeID); state != "stopped" {
			t.Errorf("After rollback, state = %v, want 'stopped'", state)
		}
	})

	t.Run("returns error when runtime Start fails", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()
		fakeRT.failOnStart = true
		fakeRT.startErr = errors.New("start operation not available")

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		// Attempt to start - should fail immediately
		err = manager.Start(ctx, added.GetID())
		if err == nil {
			t.Fatal("Start() expected error when runtime Start fails, got nil")
		}
		if !errors.Is(err, fakeRT.startErr) {
			t.Errorf("Start() error should contain runtime error, got: %v", err)
		}

		// Verify instance state in storage is still "created"
		retrieved, _ := manager.Get(added.GetID())
		if state := retrieved.GetRuntimeData().State; state != "created" {
			t.Errorf("Instance state = %v, want 'created'", state)
		}
	})

	t.Run("fails when rollback Stop also fails", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		// Make storage file read-only to force saveInstances to fail
		storageFile := filepath.Join(tmpDir, DefaultStorageFileName)
		if err := makeReadOnly(storageFile); err != nil {
			t.Fatalf("Failed to make storage file read-only: %v", err)
		}
		defer makeWritable(storageFile)

		// Configure runtime to fail on Stop (rollback will fail)
		fakeRT.failOnStop = true
		fakeRT.stopErr = errors.New("stop not supported")

		// Attempt to start
		err = manager.Start(ctx, added.GetID())
		if err == nil {
			t.Fatal("Start() expected error, got nil")
		}

		// Error should mention both save failure and rollback failure
		errMsg := err.Error()
		if !strings.Contains(errMsg, "save") || !strings.Contains(errMsg, "stop") {
			t.Errorf("Expected error to mention both save and stop failures, got: %v", err)
		}
	})
}

func TestManager_Stop_FailureRollback(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("rolls back runtime state when saveInstances fails", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		// Start the instance first
		if err := manager.Start(ctx, added.GetID()); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}

		runtimeID := added.GetRuntimeData().InstanceID

		// Verify state is "running"
		if state := fakeRT.getState(runtimeID); state != "running" {
			t.Errorf("State after Start = %v, want 'running'", state)
		}

		// Make the storage file read-only to force saveInstances to fail
		storageFile := filepath.Join(tmpDir, DefaultStorageFileName)
		if err := makeReadOnly(storageFile); err != nil {
			t.Fatalf("Failed to make storage file read-only: %v", err)
		}
		defer makeWritable(storageFile)

		// Attempt to stop - should fail to save
		err = manager.Stop(ctx, added.GetID())
		if err == nil {
			t.Fatal("Stop() expected error when saveInstances fails, got nil")
		}

		// Verify runtime state was rolled back to "running"
		// (Stop succeeded, then saveInstances failed, so rollback called Start)
		if state := fakeRT.getState(runtimeID); state != "running" {
			t.Errorf("After rollback, state = %v, want 'running'", state)
		}
	})

	t.Run("returns error when runtime Stop fails", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		// Start the instance
		if err := manager.Start(ctx, added.GetID()); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}

		// Configure runtime to fail on Stop
		fakeRT.failOnStop = true
		fakeRT.stopErr = errors.New("cannot stop running instance")

		// Attempt to stop - should fail immediately
		err = manager.Stop(ctx, added.GetID())
		if err == nil {
			t.Fatal("Stop() expected error when runtime Stop fails, got nil")
		}
		if !errors.Is(err, fakeRT.stopErr) {
			t.Errorf("Stop() error should contain runtime error, got: %v", err)
		}

		// Verify instance state in storage is still "running"
		retrieved, _ := manager.Get(added.GetID())
		if state := retrieved.GetRuntimeData().State; state != "running" {
			t.Errorf("Instance state = %v, want 'running'", state)
		}
	})

	t.Run("fails when rollback Start also fails", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		// Start the instance
		if err := manager.Start(ctx, added.GetID()); err != nil {
			t.Fatalf("Start() failed: %v", err)
		}

		// Make storage file read-only to force saveInstances to fail
		storageFile := filepath.Join(tmpDir, DefaultStorageFileName)
		if err := makeReadOnly(storageFile); err != nil {
			t.Fatalf("Failed to make storage file read-only: %v", err)
		}
		defer makeWritable(storageFile)

		// Configure runtime to fail on Start (rollback will fail)
		fakeRT.failOnStart = true
		fakeRT.startErr = errors.New("start not available")

		// Reset start call count to track rollback attempt
		fakeRT.startCallCount = 0

		// Attempt to stop
		err = manager.Stop(ctx, added.GetID())
		if err == nil {
			t.Fatal("Stop() expected error, got nil")
		}

		// Error should mention both save failure and rollback failure
		errMsg := err.Error()
		if !strings.Contains(errMsg, "save") || !strings.Contains(errMsg, "start") {
			t.Errorf("Expected error to mention both save and start failures, got: %v", err)
		}

		// Verify rollback was attempted
		if fakeRT.startCallCount == 0 {
			t.Error("Expected rollback to attempt Start, but it was not called")
		}
	})
}

func TestManager_Delete_RuntimeCleanupRequired(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("does not delete from storage when runtime Remove fails", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()
		fakeRT.failOnRemove = true
		fakeRT.removeErr = errors.New("runtime cannot be removed")

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		instanceID := added.GetID()
		runtimeID := added.GetRuntimeData().InstanceID

		// Attempt to delete - should fail due to runtime removal failure
		err = manager.Delete(ctx, instanceID)
		if err == nil {
			t.Fatal("Delete() expected error when runtime Remove fails, got nil")
		}
		if !errors.Is(err, fakeRT.removeErr) {
			t.Errorf("Delete() error should contain runtime error, got: %v", err)
		}

		// Verify instance still exists in storage
		retrieved, err := manager.Get(instanceID)
		if err != nil {
			t.Errorf("Get() failed, instance should still exist: %v", err)
		}
		if retrieved.GetID() != instanceID {
			t.Errorf("Retrieved instance ID = %v, want %v", retrieved.GetID(), instanceID)
		}

		// Verify runtime instance still exists
		if !fakeRT.instanceExists(runtimeID) {
			t.Error("Runtime instance should still exist after failed delete")
		}
	})

	t.Run("deletes from storage only after successful runtime cleanup", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		instanceID := added.GetID()
		runtimeID := added.GetRuntimeData().InstanceID

		// Verify runtime instance exists
		if !fakeRT.instanceExists(runtimeID) {
			t.Fatal("Runtime instance should exist before delete")
		}

		// Delete should succeed
		err = manager.Delete(ctx, instanceID)
		if err != nil {
			t.Fatalf("Delete() failed: %v", err)
		}

		// Verify instance was removed from storage
		_, err = manager.Get(instanceID)
		if err != ErrInstanceNotFound {
			t.Errorf("Get() error = %v, want ErrInstanceNotFound", err)
		}

		// Verify runtime instance was removed
		if fakeRT.instanceExists(runtimeID) {
			t.Error("Runtime instance should be removed after successful delete")
		}
	})

	t.Run("returns error when runtime not available for cleanup", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		fakeRT := newFailableRuntime()

		reg := runtime.NewRegistry()
		if err := reg.Register(fakeRT); err != nil {
			t.Fatalf("Failed to register runtime: %v", err)
		}

		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), reg)

		instanceTmpDir := t.TempDir()
		inst := newFakeInstance(newFakeInstanceParams{
			SourceDir:  filepath.Join(instanceTmpDir, "source"),
			ConfigDir:  filepath.Join(instanceTmpDir, "config"),
			Accessible: true,
		})
		added, err := manager.Add(ctx, inst, "failable")
		if err != nil {
			t.Fatalf("Add() failed: %v", err)
		}

		instanceID := added.GetID()

		// Create a new manager with empty registry (runtime not available)
		manager2, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), runtime.NewRegistry())

		// Attempt to delete - should fail because runtime is not available
		err = manager2.Delete(ctx, instanceID)
		if err == nil {
			t.Fatal("Delete() expected error when runtime not available, got nil")
		}

		// Verify instance still exists in storage
		retrieved, err := manager2.Get(instanceID)
		if err != nil {
			t.Errorf("Get() failed, instance should still exist: %v", err)
		}
		if retrieved.GetID() != instanceID {
			t.Errorf("Retrieved instance ID = %v, want %v", retrieved.GetID(), instanceID)
		}
	})
}

// Helper functions

func chmod(path string, mode uint32) error {
	return os.Chmod(path, os.FileMode(mode))
}

func makeReadOnly(path string) error {
	return chmod(path, 0555)
}

func makeWritable(path string) error {
	return chmod(path, 0755)
}
