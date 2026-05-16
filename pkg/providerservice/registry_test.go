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
	"testing"
)

// fakeProviderService is a test implementation of the ProviderService interface.
type fakeProviderService struct {
	name        string
	description string
	params      []ProviderParam
}

func (f *fakeProviderService) Name() string            { return f.name }
func (f *fakeProviderService) Description() string     { return f.description }
func (f *fakeProviderService) Params() []ProviderParam { return f.params }

func TestNewRegistry(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	if reg == nil {
		t.Fatal("NewRegistry() returned nil")
	}
}

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("successfully registers provider service", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		svc := &fakeProviderService{name: "anthropic"}

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
		svc := &fakeProviderService{name: ""}

		err := reg.Register(svc)
		if err == nil {
			t.Error("Register() with empty name should return error")
		}
	})

	t.Run("returns error for duplicate registration", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		svc1 := &fakeProviderService{name: "anthropic"}
		svc2 := &fakeProviderService{name: "anthropic"}

		if err := reg.Register(svc1); err != nil {
			t.Fatalf("First Register() error = %v, want nil", err)
		}

		err := reg.Register(svc2)
		if err == nil {
			t.Error("Register() duplicate should return error")
		}
	})
}

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("retrieves registered provider service", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()
		svc := &fakeProviderService{name: "anthropic"}

		if err := reg.Register(svc); err != nil {
			t.Fatalf("Register() error = %v", err)
		}

		retrieved, err := reg.Get("anthropic")
		if err != nil {
			t.Errorf("Get() error = %v, want nil", err)
		}

		if retrieved == nil {
			t.Fatal("Get() returned nil provider service")
		}

		if retrieved.Name() != "anthropic" {
			t.Errorf("Get() returned service with name %q, want %q", retrieved.Name(), "anthropic")
		}
	})

	t.Run("returns ErrProviderServiceNotFound for unregistered service", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()

		_, err := reg.Get("nonexistent")
		if err == nil {
			t.Error("Get() for nonexistent service should return error")
		}

		if !errors.Is(err, ErrProviderServiceNotFound) {
			t.Errorf("Get() error = %v, want ErrProviderServiceNotFound", err)
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

	t.Run("returns all registered provider service names", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()

		services := []string{"anthropic", "vertexai"}
		for _, name := range services {
			if err := reg.Register(&fakeProviderService{name: name}); err != nil {
				t.Fatalf("Register(%q) error = %v", name, err)
			}
		}

		names := reg.List()
		if len(names) != len(services) {
			t.Errorf("List() returned %d names, want %d", len(names), len(services))
		}

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

	t.Run("returns names in sorted order", func(t *testing.T) {
		t.Parallel()

		reg := NewRegistry()

		for _, name := range []string{"zebra", "alpha", "mango"} {
			if err := reg.Register(&fakeProviderService{name: name}); err != nil {
				t.Fatalf("Register(%q) error = %v", name, err)
			}
		}

		names := reg.List()
		expected := []string{"alpha", "mango", "zebra"}
		for i, want := range expected {
			if names[i] != want {
				t.Errorf("names[%d] = %q, want %q", i, names[i], want)
			}
		}
	})
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()

	if err := reg.Register(&fakeProviderService{name: "anthropic"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = reg.Get("anthropic")
			_ = reg.List()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
