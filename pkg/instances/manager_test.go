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
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// fakeInstance is a test double for the Instance interface
type fakeInstance struct {
	sourceDir  string
	configDir  string
	accessible bool
}

// Compile-time check to ensure fakeInstance implements Instance interface
var _ Instance = (*fakeInstance)(nil)

func (f *fakeInstance) GetSourceDir() string {
	return f.sourceDir
}

func (f *fakeInstance) GetConfigDir() string {
	return f.configDir
}

func (f *fakeInstance) IsAccessible() bool {
	return f.accessible
}

func (f *fakeInstance) Dump() InstanceData {
	return InstanceData{
		SourceDir: f.sourceDir,
		ConfigDir: f.configDir,
	}
}

// newFakeInstance creates a new fake instance for testing
func newFakeInstance(sourceDir, configDir string, accessible bool) Instance {
	return &fakeInstance{
		sourceDir:  sourceDir,
		configDir:  configDir,
		accessible: accessible,
	}
}

// fakeInstanceFactory creates fake instances from InstanceData for testing
func fakeInstanceFactory(data InstanceData) (Instance, error) {
	if data.SourceDir == "" {
		return nil, ErrInvalidPath
	}
	if data.ConfigDir == "" {
		return nil, ErrInvalidPath
	}
	// For testing, we assume instances are accessible by default
	// Tests can verify accessibility behavior separately
	return &fakeInstance{
		sourceDir:  data.SourceDir,
		configDir:  data.ConfigDir,
		accessible: true,
	}, nil
}

func TestNewManager(t *testing.T) {
	t.Parallel()

	t.Run("creates manager with valid storage directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, err := NewManager(tmpDir)
		if err != nil {
			t.Fatalf("NewManager() unexpected error = %v", err)
		}
		if manager == nil {
			t.Fatal("NewManager() returned nil manager")
		}

		// Verify storage directory exists
		if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
			t.Error("Storage directory was not created")
		}
	})

	t.Run("creates storage directory if not exists", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		nestedDir := filepath.Join(tmpDir, "nested", "storage")

		manager, err := NewManager(nestedDir)
		if err != nil {
			t.Fatalf("NewManager() unexpected error = %v", err)
		}
		if manager == nil {
			t.Fatal("NewManager() returned nil manager")
		}

		// Verify nested directory was created
		info, err := os.Stat(nestedDir)
		if err != nil {
			t.Fatalf("Nested directory was not created: %v", err)
		}
		if !info.IsDir() {
			t.Error("Storage path is not a directory")
		}
	})

	t.Run("returns error for empty storage directory", func(t *testing.T) {
		t.Parallel()

		_, err := NewManager("")
		if err == nil {
			t.Error("NewManager() expected error for empty storage dir, got nil")
		}
	})

	t.Run("verifies storage file path is correct", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, err := NewManager(tmpDir)
		if err != nil {
			t.Fatalf("NewManager() unexpected error = %v", err)
		}

		// We can't directly access storageFile since it's on the unexported struct,
		// but we can verify behavior by adding an instance and checking file creation
		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager.Add(inst)

		expectedFile := filepath.Join(tmpDir, DefaultStorageFileName)
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Storage file was not created at expected path: %v", expectedFile)
		}
	})
}

func TestManager_Add(t *testing.T) {
	t.Parallel()

	t.Run("adds valid instance successfully", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		err := manager.Add(inst)
		if err != nil {
			t.Fatalf("Add() unexpected error = %v", err)
		}

		// Verify instance was added
		instances, _ := manager.List()
		if len(instances) != 1 {
			t.Errorf("List() returned %d instances, want 1", len(instances))
		}
	})

	t.Run("returns error for nil instance", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		err := manager.Add(nil)
		if err == nil {
			t.Error("Add() expected error for nil instance, got nil")
		}
	})

	t.Run("returns error for duplicate instance", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst1 := newFakeInstance("/tmp/source", "/tmp/config1", true)
		manager.Add(inst1)

		// Try to add instance with same source directory
		inst2 := newFakeInstance("/tmp/source", "/tmp/config2", true)
		err := manager.Add(inst2)
		if err != ErrInstanceExists {
			t.Errorf("Add() error = %v, want %v", err, ErrInstanceExists)
		}
	})

	t.Run("verifies persistence to JSON file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager.Add(inst)

		// Check that JSON file exists and is readable
		storageFile := filepath.Join(tmpDir, DefaultStorageFileName)
		data, err := os.ReadFile(storageFile)
		if err != nil {
			t.Fatalf("Failed to read storage file: %v", err)
		}
		if len(data) == 0 {
			t.Error("Storage file is empty")
		}
	})

	t.Run("can add multiple instances", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst1 := newFakeInstance("/tmp/source1", "/tmp/config1", true)
		inst2 := newFakeInstance("/tmp/source2", "/tmp/config2", true)
		inst3 := newFakeInstance("/tmp/source3", "/tmp/config3", true)

		manager.Add(inst1)
		manager.Add(inst2)
		manager.Add(inst3)

		instances, _ := manager.List()
		if len(instances) != 3 {
			t.Errorf("List() returned %d instances, want 3", len(instances))
		}
	})
}

func TestManager_List(t *testing.T) {
	t.Parallel()

	t.Run("returns empty list when no instances exist", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		instances, err := manager.List()
		if err != nil {
			t.Fatalf("List() unexpected error = %v", err)
		}
		if len(instances) != 0 {
			t.Errorf("List() returned %d instances, want 0", len(instances))
		}
	})

	t.Run("returns all added instances", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst1 := newFakeInstance("/tmp/source1", "/tmp/config1", true)
		inst2 := newFakeInstance("/tmp/source2", "/tmp/config2", true)

		manager.Add(inst1)
		manager.Add(inst2)

		instances, err := manager.List()
		if err != nil {
			t.Fatalf("List() unexpected error = %v", err)
		}
		if len(instances) != 2 {
			t.Errorf("List() returned %d instances, want 2", len(instances))
		}
	})

	t.Run("handles empty storage file", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		// Create empty storage file
		storageFile := filepath.Join(tmpDir, DefaultStorageFileName)
		os.WriteFile(storageFile, []byte{}, 0644)

		instances, err := manager.List()
		if err != nil {
			t.Fatalf("List() unexpected error = %v", err)
		}
		if len(instances) != 0 {
			t.Errorf("List() returned %d instances, want 0 for empty file", len(instances))
		}
	})
}

func TestManager_Get(t *testing.T) {
	t.Parallel()

	t.Run("retrieves existing instance by source directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager.Add(inst)

		retrieved, err := manager.Get("/tmp/source")
		if err != nil {
			t.Fatalf("Get() unexpected error = %v", err)
		}
		if retrieved.GetSourceDir() != "/tmp/source" {
			t.Errorf("Get() returned instance with SourceDir = %v, want /tmp/source", retrieved.GetSourceDir())
		}
		if retrieved.GetConfigDir() != "/tmp/config" {
			t.Errorf("Get() returned instance with ConfigDir = %v, want /tmp/config", retrieved.GetConfigDir())
		}
	})

	t.Run("returns error for nonexistent instance", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		_, err := manager.Get("/tmp/nonexistent")
		if err != ErrInstanceNotFound {
			t.Errorf("Get() error = %v, want %v", err, ErrInstanceNotFound)
		}
	})

	t.Run("converts relative sourceDir to absolute for lookup", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		// Get current working directory
		wd, err := os.Getwd()
		if err != nil {
			t.Fatalf("Failed to get working directory: %v", err)
		}

		// Add instance with absolute path
		absolutePath := filepath.Join(wd, "relative-source")
		inst := newFakeInstance(absolutePath, "/tmp/config", true)
		manager.Add(inst)

		// Try to get with relative path
		retrieved, err := manager.Get("relative-source")
		if err != nil {
			t.Fatalf("Get() with relative path unexpected error = %v", err)
		}
		if retrieved.GetSourceDir() != absolutePath {
			t.Errorf("Get() returned instance with SourceDir = %v, want %v", retrieved.GetSourceDir(), absolutePath)
		}
	})
}

func TestManager_Delete(t *testing.T) {
	t.Parallel()

	t.Run("deletes existing instance successfully", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager.Add(inst)

		err := manager.Delete("/tmp/source")
		if err != nil {
			t.Fatalf("Delete() unexpected error = %v", err)
		}

		// Verify instance was deleted
		_, err = manager.Get("/tmp/source")
		if err != ErrInstanceNotFound {
			t.Errorf("Get() after Delete() error = %v, want %v", err, ErrInstanceNotFound)
		}
	})

	t.Run("returns error for nonexistent instance", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		err := manager.Delete("/tmp/nonexistent")
		if err != ErrInstanceNotFound {
			t.Errorf("Delete() error = %v, want %v", err, ErrInstanceNotFound)
		}
	})

	t.Run("deletes only specified instance", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst1 := newFakeInstance("/tmp/source1", "/tmp/config1", true)
		inst2 := newFakeInstance("/tmp/source2", "/tmp/config2", true)
		manager.Add(inst1)
		manager.Add(inst2)

		manager.Delete("/tmp/source1")

		// Verify source2 still exists
		_, err := manager.Get("/tmp/source2")
		if err != nil {
			t.Errorf("Get(source2) after Delete(source1) unexpected error = %v", err)
		}

		// Verify source1 is gone
		_, err = manager.Get("/tmp/source1")
		if err != ErrInstanceNotFound {
			t.Errorf("Get(source1) after Delete(source1) error = %v, want %v", err, ErrInstanceNotFound)
		}
	})

	t.Run("verifies deletion is persisted", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager1, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager1.Add(inst)
		manager1.Delete("/tmp/source")

		// Create new manager with same storage
		manager2, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)
		_, err := manager2.Get("/tmp/source")
		if err != ErrInstanceNotFound {
			t.Errorf("Get() from new manager error = %v, want %v", err, ErrInstanceNotFound)
		}
	})
}

func TestManager_Reconcile(t *testing.T) {
	t.Parallel()

	t.Run("removes instances with inaccessible source directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		// Custom factory that creates inaccessible instances for testing
		inaccessibleFactory := func(data InstanceData) (Instance, error) {
			if data.SourceDir == "" || data.ConfigDir == "" {
				return nil, ErrInvalidPath
			}
			return &fakeInstance{
				sourceDir:  data.SourceDir,
				configDir:  data.ConfigDir,
				accessible: false, // Always inaccessible for this test
			}, nil
		}
		manager, _ := newManagerWithFactory(tmpDir, inaccessibleFactory)

		// Add instance that is inaccessible
		inst := newFakeInstance("/tmp/nonexistent-source", "/tmp/config", false)
		manager.Add(inst)

		removed, err := manager.Reconcile()
		if err != nil {
			t.Fatalf("Reconcile() unexpected error = %v", err)
		}

		if len(removed) != 1 {
			t.Errorf("Reconcile() removed %d instances, want 1", len(removed))
		}
	})

	t.Run("removes instances with inaccessible config directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		// Custom factory that creates inaccessible instances for testing
		inaccessibleFactory := func(data InstanceData) (Instance, error) {
			if data.SourceDir == "" || data.ConfigDir == "" {
				return nil, ErrInvalidPath
			}
			return &fakeInstance{
				sourceDir:  data.SourceDir,
				configDir:  data.ConfigDir,
				accessible: false, // Always inaccessible for this test
			}, nil
		}
		manager, _ := newManagerWithFactory(tmpDir, inaccessibleFactory)

		// Add instance that is inaccessible
		inst := newFakeInstance("/tmp/source", "/tmp/nonexistent-config", false)
		manager.Add(inst)

		removed, err := manager.Reconcile()
		if err != nil {
			t.Fatalf("Reconcile() unexpected error = %v", err)
		}

		if len(removed) != 1 {
			t.Errorf("Reconcile() removed %d instances, want 1", len(removed))
		}
	})

	t.Run("returns list of removed source directories", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		// Custom factory that creates inaccessible instances for testing
		inaccessibleFactory := func(data InstanceData) (Instance, error) {
			if data.SourceDir == "" || data.ConfigDir == "" {
				return nil, ErrInvalidPath
			}
			return &fakeInstance{
				sourceDir:  data.SourceDir,
				configDir:  data.ConfigDir,
				accessible: false, // Always inaccessible for this test
			}, nil
		}
		manager, _ := newManagerWithFactory(tmpDir, inaccessibleFactory)

		inaccessibleSource := "/tmp/nonexistent-source"
		inst := newFakeInstance(inaccessibleSource, "/tmp/config", false)
		manager.Add(inst)

		removed, err := manager.Reconcile()
		if err != nil {
			t.Fatalf("Reconcile() unexpected error = %v", err)
		}

		if len(removed) != 1 || removed[0] != inaccessibleSource {
			t.Errorf("Reconcile() removed = %v, want [%v]", removed, inaccessibleSource)
		}
	})

	t.Run("keeps accessible instances", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		accessibleSource := "/tmp/accessible-source"
		inaccessibleSource := "/tmp/nonexistent-source"

		// Custom factory that checks source directory to determine accessibility
		mixedFactory := func(data InstanceData) (Instance, error) {
			if data.SourceDir == "" || data.ConfigDir == "" {
				return nil, ErrInvalidPath
			}
			accessible := data.SourceDir == accessibleSource
			return &fakeInstance{
				sourceDir:  data.SourceDir,
				configDir:  data.ConfigDir,
				accessible: accessible,
			}, nil
		}
		manager, _ := newManagerWithFactory(tmpDir, mixedFactory)

		accessibleConfig := "/tmp/accessible-config"

		// Add accessible instance
		accessible := newFakeInstance(accessibleSource, accessibleConfig, true)
		manager.Add(accessible)

		// Add inaccessible instance
		inaccessible := newFakeInstance(inaccessibleSource, "/tmp/nonexistent-config", false)
		manager.Add(inaccessible)

		removed, err := manager.Reconcile()
		if err != nil {
			t.Fatalf("Reconcile() unexpected error = %v", err)
		}

		if len(removed) != 1 {
			t.Errorf("Reconcile() removed %d instances, want 1", len(removed))
		}

		// Verify accessible instance still exists
		instances, _ := manager.List()
		if len(instances) != 1 {
			t.Errorf("List() after Reconcile() returned %d instances, want 1", len(instances))
		}
		if instances[0].GetSourceDir() != accessibleSource {
			t.Errorf("Remaining instance SourceDir = %v, want %v", instances[0].GetSourceDir(), accessibleSource)
		}
	})

	t.Run("returns empty list when all instances are accessible", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager.Add(inst)

		removed, err := manager.Reconcile()
		if err != nil {
			t.Fatalf("Reconcile() unexpected error = %v", err)
		}

		if len(removed) != 0 {
			t.Errorf("Reconcile() removed %d instances, want 0", len(removed))
		}
	})

	t.Run("handles empty instance list", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		removed, err := manager.Reconcile()
		if err != nil {
			t.Fatalf("Reconcile() unexpected error = %v", err)
		}

		if len(removed) != 0 {
			t.Errorf("Reconcile() removed %d instances, want 0", len(removed))
		}
	})
}

func TestManager_Persistence(t *testing.T) {
	t.Parallel()

	t.Run("data persists across different manager instances", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()

		// Create first manager and add instance
		manager1, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)
		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager1.Add(inst)

		// Create second manager with same storage
		manager2, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)
		instances, err := manager2.List()
		if err != nil {
			t.Fatalf("List() from second manager unexpected error = %v", err)
		}

		if len(instances) != 1 {
			t.Errorf("List() from second manager returned %d instances, want 1", len(instances))
		}
		if instances[0].GetSourceDir() != "/tmp/source" {
			t.Errorf("Instance SourceDir = %v, want /tmp/source", instances[0].GetSourceDir())
		}
	})

	t.Run("verifies correct JSON serialization", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		inst := newFakeInstance("/tmp/source", "/tmp/config", true)
		manager.Add(inst)

		// Read and verify JSON content
		storageFile := filepath.Join(tmpDir, DefaultStorageFileName)
		data, err := os.ReadFile(storageFile)
		if err != nil {
			t.Fatalf("Failed to read storage file: %v", err)
		}

		content := string(data)
		// Verify JSON contains expected fields
		if !contains(content, "source_dir") {
			t.Error("JSON does not contain 'source_dir' field")
		}
		if !contains(content, "config_dir") {
			t.Error("JSON does not contain 'config_dir' field")
		}
		if !contains(content, "/tmp/source") {
			t.Error("JSON does not contain source directory value")
		}
		if !contains(content, "/tmp/config") {
			t.Error("JSON does not contain config directory value")
		}
	})
}

func TestManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Run("thread safety with concurrent Add operations", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				sourceDir := filepath.Join("/tmp", "source", string(rune('a'+id)))
				configDir := filepath.Join("/tmp", "config", string(rune('a'+id)))
				inst := newFakeInstance(sourceDir, configDir, true)
				manager.Add(inst)
			}(i)
		}

		wg.Wait()

		instances, _ := manager.List()
		if len(instances) != numGoroutines {
			t.Errorf("After concurrent adds, List() returned %d instances, want %d", len(instances), numGoroutines)
		}
	})

	t.Run("thread safety with concurrent reads", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		manager, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory)

		// Add some instances first
		for i := 0; i < 5; i++ {
			sourceDir := filepath.Join("/tmp", "source", string(rune('a'+i)))
			configDir := filepath.Join("/tmp", "config", string(rune('a'+i)))
			inst := newFakeInstance(sourceDir, configDir, true)
			manager.Add(inst)
		}

		var wg sync.WaitGroup
		numGoroutines := 20

		// Concurrent List operations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				manager.List()
			}()
		}

		wg.Wait()

		// Verify data is still consistent
		instances, _ := manager.List()
		if len(instances) != 5 {
			t.Errorf("After concurrent reads, List() returned %d instances, want 5", len(instances))
		}
	})
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
