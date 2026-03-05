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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

const (
	// DefaultStorageFileName is the default filename for storing instances
	DefaultStorageFileName = "instances.json"
)

// InstanceFactory is a function that creates an Instance from InstanceData
type InstanceFactory func(InstanceData) (Instance, error)

// Manager handles instance storage and operations
type Manager interface {
	// Add registers a new instance
	Add(inst Instance) error
	// List returns all registered instances
	List() ([]Instance, error)
	// Get retrieves a specific instance by source directory (unique key)
	Get(sourceDir string) (Instance, error)
	// Delete unregisters an instance by source directory (unique key)
	Delete(sourceDir string) error
	// Reconcile removes instances with inaccessible directories
	// Returns the list of removed instance source directories
	Reconcile() ([]string, error)
}

// manager is the internal implementation of Manager
type manager struct {
	storageFile string
	mu          sync.RWMutex
	factory     InstanceFactory
}

// Compile-time check to ensure manager implements Manager interface
var _ Manager = (*manager)(nil)

// NewManager creates a new instance manager with the given storage directory.
func NewManager(storageDir string) (Manager, error) {
	return newManagerWithFactory(storageDir, NewInstanceFromData)
}

// newManagerWithFactory creates a new instance manager with a custom instance factory.
// This is unexported and primarily useful for testing with fake instances.
func newManagerWithFactory(storageDir string, factory InstanceFactory) (Manager, error) {
	if storageDir == "" {
		return nil, errors.New("storage directory cannot be empty")
	}
	if factory == nil {
		return nil, errors.New("factory cannot be nil")
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, err
	}

	storageFile := filepath.Join(storageDir, DefaultStorageFileName)
	return &manager{
		storageFile: storageFile,
		factory:     factory,
	}, nil
}

// Add registers a new instance.
// The instance must be created using NewInstance to ensure proper validation.
func (m *manager) Add(inst Instance) error {
	if inst == nil {
		return errors.New("instance cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	instances, err := m.loadInstances()
	if err != nil {
		return err
	}

	// Check for duplicate source directory (unique key)
	for _, existing := range instances {
		if existing.GetSourceDir() == inst.GetSourceDir() {
			return ErrInstanceExists
		}
	}

	instances = append(instances, inst)
	return m.saveInstances(instances)
}

// List returns all registered instances
func (m *manager) List() ([]Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.loadInstances()
}

// Get retrieves a specific instance by source directory (unique key)
func (m *manager) Get(sourceDir string) (Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Convert to absolute path for comparison
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, err
	}

	instances, err := m.loadInstances()
	if err != nil {
		return nil, err
	}

	// Look up by source directory (unique key)
	for _, instance := range instances {
		if instance.GetSourceDir() == absSourceDir {
			return instance, nil
		}
	}

	return nil, ErrInstanceNotFound
}

// Delete unregisters an instance by source directory (unique key)
func (m *manager) Delete(sourceDir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert to absolute path for comparison
	absSourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return err
	}

	instances, err := m.loadInstances()
	if err != nil {
		return err
	}

	// Find and remove by source directory (unique key)
	found := false
	filtered := make([]Instance, 0, len(instances))
	for _, instance := range instances {
		if instance.GetSourceDir() != absSourceDir {
			filtered = append(filtered, instance)
		} else {
			found = true
		}
	}

	if !found {
		return ErrInstanceNotFound
	}

	return m.saveInstances(filtered)
}

// Reconcile removes instances with inaccessible directories
// Returns the list of removed instance source directories
func (m *manager) Reconcile() ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	instances, err := m.loadInstances()
	if err != nil {
		return nil, err
	}

	removed := []string{}
	accessible := make([]Instance, 0, len(instances))

	for _, instance := range instances {
		if instance.IsAccessible() {
			accessible = append(accessible, instance)
		} else {
			removed = append(removed, instance.GetSourceDir())
		}
	}

	if len(removed) > 0 {
		if err := m.saveInstances(accessible); err != nil {
			return nil, err
		}
	}

	return removed, nil
}

// loadInstances reads instances from the storage file
func (m *manager) loadInstances() ([]Instance, error) {
	// If file doesn't exist, return empty list
	if _, err := os.Stat(m.storageFile); os.IsNotExist(err) {
		return []Instance{}, nil
	}

	data, err := os.ReadFile(m.storageFile)
	if err != nil {
		return nil, err
	}

	// Empty file case
	if len(data) == 0 {
		return []Instance{}, nil
	}

	// Unmarshal into InstanceData slice
	var instancesData []InstanceData
	if err := json.Unmarshal(data, &instancesData); err != nil {
		return nil, err
	}

	// Convert to Instance slice using the factory
	instances := make([]Instance, len(instancesData))
	for i, data := range instancesData {
		inst, err := m.factory(data)
		if err != nil {
			return nil, err
		}
		instances[i] = inst
	}

	return instances, nil
}

// saveInstances writes instances to the storage file
func (m *manager) saveInstances(instances []Instance) error {
	// Convert to InstanceData slice for marshaling
	instancesData := make([]InstanceData, len(instances))
	for i, inst := range instances {
		instancesData[i] = inst.Dump()
	}

	data, err := json.MarshalIndent(instancesData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.storageFile, data, 0644)
}
