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

package providerservicesetup

import (
	"errors"
	"testing"

	"github.com/openkaiden/kdn/pkg/providerservice"
)

// fakeRegistrar implements ProviderServiceRegistrar for testing.
type fakeRegistrar struct {
	registered map[string]providerservice.ProviderService
	failOn     string
}

func newFakeRegistrar() *fakeRegistrar {
	return &fakeRegistrar{
		registered: make(map[string]providerservice.ProviderService),
	}
}

func (f *fakeRegistrar) RegisterProviderService(service providerservice.ProviderService) error {
	if service.Name() == f.failOn {
		return errors.New("registration failed")
	}
	f.registered[service.Name()] = service
	return nil
}

func TestRegisterAll(t *testing.T) {
	t.Parallel()

	registrar := newFakeRegistrar()

	if err := RegisterAll(registrar); err != nil {
		t.Fatalf("RegisterAll() error = %v, want nil", err)
	}

	if len(registrar.registered) != 2 {
		t.Errorf("registered %d provider services, want 2", len(registrar.registered))
	}
	if _, exists := registrar.registered["anthropic"]; !exists {
		t.Error("anthropic provider service was not registered")
	}
	if _, exists := registrar.registered["vertexai"]; !exists {
		t.Error("vertexai provider service was not registered")
	}
}

func TestListAvailable(t *testing.T) {
	t.Parallel()

	names := ListAvailable()

	if len(names) != 2 {
		t.Fatalf("ListAvailable() returned %d names, want 2", len(names))
	}
	// Order matches providerservices.json: anthropic, vertexai.
	expected := []string{"anthropic", "vertexai"}
	for i, want := range expected {
		if names[i] != want {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want)
		}
	}
}

func TestListServices(t *testing.T) {
	t.Parallel()

	services := ListServices()

	if len(services) != 2 {
		t.Fatalf("ListServices() returned %d services, want 2", len(services))
	}
	for i, svc := range services {
		if svc == nil {
			t.Errorf("services[%d] is nil", i)
			continue
		}
		if svc.Name() == "" {
			t.Errorf("services[%d].Name() is empty", i)
		}
		if len(svc.Params()) == 0 {
			t.Errorf("services[%d] (%s) has no params", i, svc.Name())
		}
	}
	if services[0].Name() != "anthropic" {
		t.Errorf("services[0].Name() = %q, want %q", services[0].Name(), "anthropic")
	}
}

func TestLoadProviderServices(t *testing.T) {
	t.Parallel()

	factories, err := loadProviderServices(providerServicesJSON)
	if err != nil {
		t.Fatalf("loadProviderServices() error = %v", err)
	}
	if len(factories) != 2 {
		t.Fatalf("loadProviderServices() returned %d factories, want 2", len(factories))
	}

	svc := factories[0]()
	if svc == nil {
		t.Fatal("factory returned nil")
	}
	if svc.Name() != "anthropic" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "anthropic")
	}
}

func TestAnthropicProviderService(t *testing.T) {
	t.Parallel()

	services := ListServices()
	if len(services) == 0 {
		t.Fatal("no services returned")
	}

	svc := services[0]
	if svc.Name() != "anthropic" {
		t.Fatalf("expected anthropic service first, got %q", svc.Name())
	}
	if svc.Description() == "" {
		t.Error("Description() should not be empty")
	}

	params := svc.Params()
	if len(params) != 2 {
		t.Fatalf("Params() returned %d params, want 2", len(params))
	}

	token := params[0]
	if token.Name != "token" {
		t.Errorf("params[0].Name = %q, want %q", token.Name, "token")
	}
	if token.Kind != providerservice.ProviderParamKindSecret {
		t.Errorf("params[0].Kind = %q, want %q", token.Kind, providerservice.ProviderParamKindSecret)
	}
	if !token.Required {
		t.Error("params[0].Required = false, want true")
	}
	if token.SecretType != "anthropic" {
		t.Errorf("params[0].SecretType = %q, want %q", token.SecretType, "anthropic")
	}

	url := params[1]
	if url.Name != "url" {
		t.Errorf("params[1].Name = %q, want %q", url.Name, "url")
	}
	if url.Kind != providerservice.ProviderParamKindURL {
		t.Errorf("params[1].Kind = %q, want %q", url.Kind, providerservice.ProviderParamKindURL)
	}
	if url.Required {
		t.Error("params[1].Required = true, want false")
	}
}

func TestVertexAIProviderService(t *testing.T) {
	t.Parallel()

	services := ListServices()
	if len(services) < 2 {
		t.Fatal("expected at least 2 services")
	}

	svc := services[1]
	if svc.Name() != "vertexai" {
		t.Fatalf("expected vertexai service second, got %q", svc.Name())
	}
	if svc.Description() == "" {
		t.Error("Description() should not be empty")
	}

	params := svc.Params()
	if len(params) != 3 {
		t.Fatalf("Params() returned %d params, want 3", len(params))
	}

	project := params[0]
	if project.Name != "project" || project.Kind != providerservice.ProviderParamKindText || !project.Required {
		t.Errorf("params[0] = %+v, unexpected", project)
	}

	region := params[1]
	if region.Name != "region" || region.Kind != providerservice.ProviderParamKindText || !region.Required {
		t.Errorf("params[1] = %+v, unexpected", region)
	}

	creds := params[2]
	if creds.Name != "credentials" || creds.Kind != providerservice.ProviderParamKindCredential || creds.Required {
		t.Errorf("params[2] = %+v, unexpected", creds)
	}
}

func TestRegisterAllWithFactories_StopsOnError(t *testing.T) {
	t.Parallel()

	registrar := newFakeRegistrar()
	registrar.failOn = "anthropic"

	factories := []providerServiceFactory{
		func() providerservice.ProviderService {
			return providerservice.NewProviderService("anthropic", "test", nil)
		},
	}

	err := registerAllWithFactories(registrar, factories)
	if err == nil {
		t.Error("registerAllWithFactories() should return error when registration fails")
	}
}

func TestRegisterAll_NilRegistrar(t *testing.T) {
	t.Parallel()

	err := RegisterAll(nil)
	if err == nil {
		t.Error("RegisterAll(nil) should return error")
	}
}

func TestRegisterAllWithFactories_NilRegistrar(t *testing.T) {
	t.Parallel()

	err := registerAllWithFactories(nil, nil)
	if err == nil {
		t.Error("registerAllWithFactories(nil, ...) should return error")
	}
}

func TestLoadProviderServices_ValidatesEmptyName(t *testing.T) {
	t.Parallel()

	_, err := loadProviderServices([]byte(`[{"name":"","description":"test","params":[]}]`))
	if err == nil {
		t.Error("loadProviderServices() should return error for empty provider name")
	}
}

func TestLoadProviderServices_ValidatesParamKind(t *testing.T) {
	t.Parallel()

	_, err := loadProviderServices([]byte(`[{"name":"test","params":[{"name":"p","kind":"invalid"}]}]`))
	if err == nil {
		t.Error("loadProviderServices() should return error for invalid param kind")
	}
}

func TestLoadProviderServices_ValidatesSecretParamHasSecretType(t *testing.T) {
	t.Parallel()

	_, err := loadProviderServices([]byte(`[{"name":"test","params":[{"name":"token","kind":"secret","secret_type":""}]}]`))
	if err == nil {
		t.Error("loadProviderServices() should return error when secret param has no secret_type")
	}
}

func TestLoadProviderServices_ValidatesNonSecretParamHasNoSecretType(t *testing.T) {
	t.Parallel()

	_, err := loadProviderServices([]byte(`[{"name":"test","params":[{"name":"url","kind":"url","secret_type":"something"}]}]`))
	if err == nil {
		t.Error("loadProviderServices() should return error when non-secret param has secret_type")
	}
}

func TestListAvailableWithFactories(t *testing.T) {
	t.Parallel()

	factories := []providerServiceFactory{
		func() providerservice.ProviderService { return providerservice.NewProviderService("alpha", "", nil) },
		func() providerservice.ProviderService { return providerservice.NewProviderService("beta", "", nil) },
	}

	names := listAvailableWithFactories(factories)

	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "alpha" {
		t.Errorf("names[0] = %q, want %q", names[0], "alpha")
	}
	if names[1] != "beta" {
		t.Errorf("names[1] = %q, want %q", names[1], "beta")
	}
}
