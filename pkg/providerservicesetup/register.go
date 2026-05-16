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

// Package providerservicesetup provides centralized registration of all available provider service implementations.
package providerservicesetup

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/openkaiden/kdn/pkg/providerservice"
)

// ProviderServiceRegistrar is an interface for types that can register provider services.
type ProviderServiceRegistrar interface {
	RegisterProviderService(service providerservice.ProviderService) error
}

// providerServiceFactory is a function that creates a new provider service instance.
type providerServiceFactory func() providerservice.ProviderService

//go:embed providerservices.json
var providerServicesJSON []byte

// providerParamDefinition represents a parameter entry in the embedded JSON file.
type providerParamDefinition struct {
	Name        string                            `json:"name"`
	Description string                            `json:"description"`
	Required    bool                              `json:"required"`
	Kind        providerservice.ProviderParamKind `json:"kind"`
	SecretType  string                            `json:"secret_type"`
}

// providerServiceDefinition represents a provider service entry in the embedded JSON file.
type providerServiceDefinition struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Params      []providerParamDefinition `json:"params"`
}

// availableProviderServices is the list of all provider services loaded from the embedded JSON file.
var availableProviderServices = mustLoadProviderServices()

// mustLoadProviderServices loads provider service definitions from the embedded JSON and returns
// factory functions for each. It panics on error since embedded data corruption is a build defect.
func mustLoadProviderServices() []providerServiceFactory {
	factories, err := loadProviderServices(providerServicesJSON)
	if err != nil {
		panic(fmt.Sprintf("failed to load embedded provider services: %v", err))
	}
	return factories
}

// validProviderParamKinds is the set of accepted ProviderParamKind values.
var validProviderParamKinds = map[providerservice.ProviderParamKind]bool{
	providerservice.ProviderParamKindSecret:     true,
	providerservice.ProviderParamKindCredential: true,
	providerservice.ProviderParamKindURL:        true,
	providerservice.ProviderParamKindText:       true,
}

// loadProviderServices parses the given JSON bytes and returns a factory function for each definition.
func loadProviderServices(data []byte) ([]providerServiceFactory, error) {
	var definitions []providerServiceDefinition
	if err := json.Unmarshal(data, &definitions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provider services JSON: %w", err)
	}

	for _, def := range definitions {
		if def.Name == "" {
			return nil, fmt.Errorf("provider service definition has empty name")
		}
		for _, p := range def.Params {
			if p.Name == "" {
				return nil, fmt.Errorf("provider %q: param has empty name", def.Name)
			}
			if !validProviderParamKinds[p.Kind] {
				return nil, fmt.Errorf("provider %q: param %q has invalid kind %q", def.Name, p.Name, p.Kind)
			}
			if p.Kind == providerservice.ProviderParamKindSecret && p.SecretType == "" {
				return nil, fmt.Errorf("provider %q: secret param %q must have a non-empty secret_type", def.Name, p.Name)
			}
			if p.Kind != providerservice.ProviderParamKindSecret && p.SecretType != "" {
				return nil, fmt.Errorf("provider %q: non-secret param %q must not have a secret_type", def.Name, p.Name)
			}
		}
	}

	factories := make([]providerServiceFactory, 0, len(definitions))
	for _, def := range definitions {
		d := def // capture loop variable
		factories = append(factories, func() providerservice.ProviderService {
			params := make([]providerservice.ProviderParam, 0, len(d.Params))
			for _, p := range d.Params {
				params = append(params, providerservice.ProviderParam{
					Name:        p.Name,
					Description: p.Description,
					Required:    p.Required,
					Kind:        p.Kind,
					SecretType:  p.SecretType,
				})
			}
			return providerservice.NewProviderService(d.Name, d.Description, params)
		})
	}

	return factories, nil
}

// RegisterAll registers all available provider service implementations to the given registrar.
// Returns an error if any provider service fails to register.
func RegisterAll(registrar ProviderServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("provider service registrar cannot be nil")
	}
	return registerAllWithFactories(registrar, availableProviderServices)
}

// ListAvailable returns the names of all available provider services.
func ListAvailable() []string {
	return listAvailableWithFactories(availableProviderServices)
}

// ListServices returns fully-constructed instances of all available provider services.
func ListServices() []providerservice.ProviderService {
	return listServicesWithFactories(availableProviderServices)
}

// listServicesWithFactories returns fully-constructed provider services from the given factories.
func listServicesWithFactories(factories []providerServiceFactory) []providerservice.ProviderService {
	services := make([]providerservice.ProviderService, 0, len(factories))
	for _, factory := range factories {
		services = append(services, factory())
	}
	return services
}

// listAvailableWithFactories returns the names of provider services from the given factories.
func listAvailableWithFactories(factories []providerServiceFactory) []string {
	names := make([]string, 0, len(factories))
	for _, factory := range factories {
		svc := factory()
		names = append(names, svc.Name())
	}
	return names
}

// registerAllWithFactories registers provider services from the given factories to the registrar.
func registerAllWithFactories(registrar ProviderServiceRegistrar, factories []providerServiceFactory) error {
	if registrar == nil {
		return fmt.Errorf("provider service registrar cannot be nil")
	}
	for _, factory := range factories {
		svc := factory()
		if svc == nil {
			return fmt.Errorf("provider service factory returned nil")
		}
		if err := registrar.RegisterProviderService(svc); err != nil {
			return fmt.Errorf("failed to register provider service %q: %w", svc.Name(), err)
		}
	}
	return nil
}
