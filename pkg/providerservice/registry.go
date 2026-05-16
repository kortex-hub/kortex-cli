/**********************************************************************
 * Copyright (C) 2026 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package providerservice

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

var (
	// ErrProviderServiceNotFound is returned when a provider service is not found in the registry.
	ErrProviderServiceNotFound = errors.New("provider service not found")
)

// Registry manages provider service implementations.
type Registry interface {
	// Register registers a provider service implementation.
	// Returns an error if a provider service with the same name is already registered.
	Register(service ProviderService) error
	// Get retrieves a provider service implementation by name.
	// Returns ErrProviderServiceNotFound if the provider service is not registered.
	Get(name string) (ProviderService, error)
	// List returns all registered provider service names.
	List() []string
}

// registry is the internal implementation of Registry.
type registry struct {
	mu       sync.RWMutex
	services map[string]ProviderService
}

// Compile-time check to ensure registry implements Registry interface.
var _ Registry = (*registry)(nil)

// NewRegistry creates a new provider service registry.
func NewRegistry() Registry {
	return &registry{
		services: make(map[string]ProviderService),
	}
}

// Register registers a provider service implementation.
func (r *registry) Register(service ProviderService) error {
	if service == nil {
		return errors.New("provider service cannot be nil")
	}

	name := service.Name()
	if name == "" {
		return errors.New("provider service name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.services[name]; exists {
		return fmt.Errorf("provider service %q is already registered", name)
	}

	r.services[name] = service
	return nil
}

// Get retrieves a provider service implementation by name.
func (r *registry) Get(name string) (ProviderService, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[name]
	if !exists {
		return nil, ErrProviderServiceNotFound
	}

	return service, nil
}

// List returns all registered provider service names.
func (r *registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.services))
	for name := range r.services {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
