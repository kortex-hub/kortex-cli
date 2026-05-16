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

// Package providerservice provides interfaces and types for managing LLM provider service definitions.
// A provider service describes the parameters required to configure an LLM provider connection.
package providerservice

// ProviderParamKind describes the nature of a provider parameter and how it is stored/used.
type ProviderParamKind string

const (
	// ProviderParamKindSecret — sensitive value (API token); stored in system keychain.
	ProviderParamKindSecret ProviderParamKind = "secret"
	// ProviderParamKindCredential — path to a credential file (e.g. gcloud ADC JSON);
	// only the path is stored (non-sensitive); the runtime intercepts and injects the file.
	ProviderParamKindCredential ProviderParamKind = "credential"
	// ProviderParamKindURL — a URL endpoint; stored as plain text but may be rewritten
	// by runtimes (e.g. localhost → host.containers.internal inside a container).
	ProviderParamKindURL ProviderParamKind = "url"
	// ProviderParamKindText — free-form plain text (e.g. GCP project ID, region);
	// stored as-is in providers.json.
	ProviderParamKindText ProviderParamKind = "text"
)

// ProviderParam describes a single parameter accepted by a provider service.
type ProviderParam struct {
	Name        string
	Description string
	Required    bool
	Kind        ProviderParamKind
	// SecretType is the secret type used when storing this param in the secret store.
	// Only meaningful when Kind == ProviderParamKindSecret.
	// Valid values are the names of registered secret services (e.g. "anthropic", "github").
	SecretType string
}

// ProviderService defines the contract for a provider service implementation.
// Each provider service describes how a particular type of LLM provider is configured.
type ProviderService interface {
	// Name returns the identifier of the provider service.
	Name() string

	// Description returns a human-readable description of the provider service.
	Description() string

	// Params returns the list of parameters accepted by this provider service.
	Params() []ProviderParam
}

// providerService is the concrete implementation of ProviderService.
type providerService struct {
	name        string
	description string
	params      []ProviderParam
}

// Compile-time check to ensure providerService implements ProviderService interface.
var _ ProviderService = (*providerService)(nil)

// NewProviderService creates a new ProviderService implementation with the given parameters.
func NewProviderService(name, description string, params []ProviderParam) ProviderService {
	return &providerService{
		name:        name,
		description: description,
		params:      append([]ProviderParam(nil), params...),
	}
}

func (p *providerService) Name() string        { return p.name }
func (p *providerService) Description() string { return p.description }
func (p *providerService) Params() []ProviderParam {
	return append([]ProviderParam(nil), p.params...)
}
