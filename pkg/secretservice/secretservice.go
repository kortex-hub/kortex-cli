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

// Package secretservice provides interfaces and types for managing secret service definitions.
// A secret service describes how a particular type of secret is applied to workspace requests,
// including which hosts it matches, what HTTP header to set, and how to format the value.
package secretservice

// SecretService defines the contract for a secret service implementation.
// Each secret service describes how secrets of a particular type are applied.
type SecretService interface {
	// Name returns the identifier of the secret service.
	Name() string

	// Description returns a human-readable description of the secret service.
	// Returns an empty string if not set.
	Description() string

	// HostsPatterns returns the list of regular expression patterns for matching hosts.
	// Returns nil if not set.
	HostsPatterns() []string

	// Path returns the optional path for the secret service.
	// Returns an empty string if not set.
	Path() string

	// EnvVars returns the optional list of environment variable names.
	// Returns nil if not set.
	EnvVars() []string

	// HeaderName returns the name of the HTTP header.
	HeaderName() string

	// HeaderTemplate returns the optional template for the header value.
	// The template uses ${value} for value insertion.
	// Returns an empty string if not set.
	HeaderTemplate() string
}

// service is the concrete implementation of SecretService.
type service struct {
	name           string
	description    string
	hostsPatterns  []string
	path           string
	envVars        []string
	headerName     string
	headerTemplate string
}

// Compile-time check to ensure service implements SecretService interface
var _ SecretService = (*service)(nil)

// NewSecretService creates a new SecretService implementation with the given parameters.
func NewSecretService(name string, hostsPatterns []string, path string, envVars []string, headerName, headerTemplate, description string) SecretService {
	return &service{
		name:           name,
		description:    description,
		hostsPatterns:  hostsPatterns,
		path:           path,
		envVars:        envVars,
		headerName:     headerName,
		headerTemplate: headerTemplate,
	}
}

// Name returns the identifier of the secret service.
func (s *service) Name() string {
	return s.name
}

// Description returns a human-readable description of the secret service.
func (s *service) Description() string {
	return s.description
}

// HostsPatterns returns the list of regular expression patterns for matching hosts.
func (s *service) HostsPatterns() []string {
	return s.hostsPatterns
}

// Path returns the optional path for the secret service.
func (s *service) Path() string {
	return s.path
}

// EnvVars returns the optional list of environment variable names.
func (s *service) EnvVars() []string {
	return s.envVars
}

// HeaderName returns the name of the HTTP header.
func (s *service) HeaderName() string {
	return s.headerName
}

// HeaderTemplate returns the optional template for the header value.
func (s *service) HeaderTemplate() string {
	return s.headerTemplate
}
