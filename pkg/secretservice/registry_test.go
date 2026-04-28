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

package secretservice

import (
	"errors"
	"testing"
)

// fakeSecretService is a test implementation of the SecretService interface
type fakeSecretService struct {
	name           string
	description    string
	hostsPatterns  []string
	path           string
	envVars        []string
	headerName     string
	headerTemplate string
}

func (f *fakeSecretService) Name() string            { return f.name }
func (f *fakeSecretService) Description() string     { return f.description }
func (f *fakeSecretService) HostsPatterns() []string { return f.hostsPatterns }
func (f *fakeSecretService) Path() string            { return f.path }
func (f *fakeSecretService) EnvVars() []string       { return f.envVars }
func (f *fakeSecretService) HeaderName() string      { return f.headerName }
func (f *fakeSecretService) HeaderTemplate() string  { return f.headerTemplate }

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("successfully registers secret service", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		svc := &fakeSecretService{name: "github"}

		err := reg.Register(svc)
		if err != nil {
			t.Errorf("Register() error = %v, want nil", err)
		}
	})

	t.Run("returns error for nil service", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()

		err := reg.Register(nil)
		if err == nil {
			t.Error("Register() with nil service should return error")
		}
	})

	t.Run("returns error for empty name", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		svc := &fakeSecretService{name: ""}

		err := reg.Register(svc)
		if err == nil {
			t.Error("Register() with empty name should return error")
		}
	})

	t.Run("returns error for duplicate registration", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		svc1 := &fakeSecretService{name: "github"}
		svc2 := &fakeSecretService{name: "github"}

		err := reg.Register(svc1)
		if err != nil {
			t.Fatalf("First Register() error = %v, want nil", err)
		}

		err = reg.Register(svc2)
		if err == nil {
			t.Error("Register() duplicate should return error")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("retrieves registered secret service", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		svc := &fakeSecretService{name: "github", hostsPatterns: []string{"github.com"}}

		err := reg.Register(svc)
		if err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		retrieved, err := reg.Get("github")
		if err != nil {
			t.Errorf("Get() error = %v, want nil", err)
		}

		if retrieved == nil {
			t.Fatal("Get() returned nil secret service")
		}

		if retrieved.Name() != "github" {
			t.Errorf("Get() returned service with name %q, want %q", retrieved.Name(), "github")
		}

		if len(retrieved.HostsPatterns()) == 0 || retrieved.HostsPatterns()[0] != "github.com" {
			t.Errorf("Get() returned service with host patterns %v, want %v", retrieved.HostsPatterns(), []string{"github.com"})
		}
	})

	t.Run("returns ErrSecretServiceNotFound for unregistered service", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()

		_, err := reg.Get("nonexistent")
		if err == nil {
			t.Error("Get() for nonexistent service should return error")
		}

		if !errors.Is(err, ErrSecretServiceNotFound) {
			t.Errorf("Get() error = %v, want ErrSecretServiceNotFound", err)
		}
	})
}

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	t.Run("returns empty list for new registry", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		names := reg.List()

		if len(names) != 0 {
			t.Errorf("List() returned %d names, want 0", len(names))
		}
	})

	t.Run("returns all registered secret service names", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()

		services := []string{"github", "slack", "vault"}
		for _, name := range services {
			err := reg.Register(&fakeSecretService{name: name})
			if err != nil {
				t.Fatalf("Register(%q) error = %v", name, err)
			}
		}

		names := reg.List()
		if len(names) != len(services) {
			t.Errorf("List() returned %d names, want %d", len(names), len(services))
		}

		// Check all expected names are present
		nameMap := make(map[string]bool)
		for _, name := range names {
			nameMap[name] = true
		}

		for _, expected := range services {
			if !nameMap[expected] {
				t.Errorf("List() missing expected service %q", expected)
			}
		}
	})
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	// Register a service
	err := reg.Register(&fakeSecretService{name: "github"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Concurrent reads should be safe
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = reg.Get("github")
			_ = reg.List()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
