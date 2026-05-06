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

package credential

import (
	"errors"
	"fmt"
	"sync"
)

// registry is the internal implementation of Registry.
type registry struct {
	mu          sync.RWMutex
	credentials []Credential
	names       map[string]struct{}
}

// Compile-time check to ensure registry implements Registry.
var _ Registry = (*registry)(nil)

// NewRegistry creates a new credential registry.
func NewRegistry() Registry {
	return &registry{
		names: make(map[string]struct{}),
	}
}

// Register adds a credential to the registry.
func (r *registry) Register(c Credential) error {
	if c == nil {
		return errors.New("credential cannot be nil")
	}

	name := c.Name()
	if name == "" {
		return errors.New("credential name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.names[name]; exists {
		return fmt.Errorf("credential %q is already registered", name)
	}

	r.names[name] = struct{}{}
	r.credentials = append(r.credentials, c)
	return nil
}

// List returns all registered credentials in registration order.
func (r *registry) List() []Credential {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Credential, len(r.credentials))
	copy(result, r.credentials)
	return result
}
