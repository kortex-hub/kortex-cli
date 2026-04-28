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

package onecli

import (
	"testing"

	"github.com/openkaiden/kdn/pkg/secret"
	"github.com/openkaiden/kdn/pkg/secretservice"
)

func registryWithGitHub(t *testing.T) secretservice.Registry {
	t.Helper()
	reg := secretservice.NewRegistry()
	svc := secretservice.NewSecretService(
		"github",
		[]string{"api.github.com"},
		"",
		[]string{"GH_TOKEN", "GITHUB_TOKEN"},
		"Authorization",
		"Bearer ${value}",
		"",
	)
	if err := reg.Register(svc); err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestMapper_KnownType_GitHub(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(registryWithGitHub(t))
	item := secret.ListItem{
		Name: "my-gh-token",
		Type: "github",
	}

	got, err := mapper.Map(item, "ghp_abc123")
	if err != nil {
		t.Fatalf("Map() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Map() returned %d inputs, want 1", len(got))
	}

	if got[0].Name != "my-gh-token" {
		t.Errorf("Name = %q, want %q", got[0].Name, "my-gh-token")
	}
	if got[0].Type != "generic" {
		t.Errorf("Type = %q, want %q", got[0].Type, "generic")
	}
	if got[0].Value != "ghp_abc123" {
		t.Errorf("Value = %q, want %q", got[0].Value, "ghp_abc123")
	}
	if got[0].HostPattern != "api.github.com" {
		t.Errorf("HostPattern = %q, want %q", got[0].HostPattern, "api.github.com")
	}
	if got[0].InjectionConfig == nil {
		t.Fatal("InjectionConfig is nil")
	}
	if got[0].InjectionConfig.HeaderName != "Authorization" {
		t.Errorf("HeaderName = %q, want %q", got[0].InjectionConfig.HeaderName, "Authorization")
	}
	if got[0].InjectionConfig.ValueFormat != "Bearer {value}" {
		t.Errorf("ValueFormat = %q, want %q", got[0].InjectionConfig.ValueFormat, "Bearer {value}")
	}
}

func TestMapper_KnownType_MultiplePatterns(t *testing.T) {
	t.Parallel()

	reg := secretservice.NewRegistry()
	svc := secretservice.NewSecretService(
		"multi",
		[]string{"api.example.com", "api2.example.com"},
		"",
		nil,
		"Authorization",
		"Bearer ${value}",
		"",
	)
	if err := reg.Register(svc); err != nil {
		t.Fatal(err)
	}

	mapper := NewSecretMapper(reg)
	item := secret.ListItem{Name: "my-token", Type: "multi"}

	got, err := mapper.Map(item, "val")
	if err != nil {
		t.Fatalf("Map() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Map() returned %d inputs, want 2", len(got))
	}
	if got[0].Name != "my-token-api-example-com" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "my-token-api-example-com")
	}
	if got[0].HostPattern != "api.example.com" {
		t.Errorf("got[0].HostPattern = %q, want %q", got[0].HostPattern, "api.example.com")
	}
	if got[1].Name != "my-token-api2-example-com" {
		t.Errorf("got[1].Name = %q, want %q", got[1].Name, "my-token-api2-example-com")
	}
	if got[1].HostPattern != "api2.example.com" {
		t.Errorf("got[1].HostPattern = %q, want %q", got[1].HostPattern, "api2.example.com")
	}
}

func TestMapper_KnownType_EmptyPatterns(t *testing.T) {
	t.Parallel()

	reg := secretservice.NewRegistry()
	svc := secretservice.NewSecretService("empty", nil, "", nil, "X-Token", "${value}", "")
	if err := reg.Register(svc); err != nil {
		t.Fatal(err)
	}

	mapper := NewSecretMapper(reg)
	item := secret.ListItem{Name: "my-token", Type: "empty"}

	_, err := mapper.Map(item, "val")
	if err == nil {
		t.Fatal("expected error for service with no host patterns")
	}
}

func TestMapper_UnknownType(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(secretservice.NewRegistry())
	item := secret.ListItem{
		Name: "my-token",
		Type: "unknown-service",
	}

	_, err := mapper.Map(item, "token")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestMapper_OtherType_AllFields(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(secretservice.NewRegistry())
	item := secret.ListItem{
		Name:           "custom-api",
		Type:           "other",
		Hosts:          []string{"api.example.com"},
		Path:           "/v2",
		Header:         "X-Api-Key",
		HeaderTemplate: "Token ${value}",
	}

	got, err := mapper.Map(item, "my-key-123")
	if err != nil {
		t.Fatalf("Map() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Map() returned %d inputs, want 1", len(got))
	}

	if got[0].Name != "custom-api" {
		t.Errorf("Name = %q, want %q", got[0].Name, "custom-api")
	}
	if got[0].Type != "generic" {
		t.Errorf("Type = %q, want %q", got[0].Type, "generic")
	}
	if got[0].Value != "my-key-123" {
		t.Errorf("Value = %q, want %q", got[0].Value, "my-key-123")
	}
	if got[0].HostPattern != "api.example.com" {
		t.Errorf("HostPattern = %q, want %q", got[0].HostPattern, "api.example.com")
	}
	if got[0].PathPattern != "/v2" {
		t.Errorf("PathPattern = %q, want %q", got[0].PathPattern, "/v2")
	}
	if got[0].InjectionConfig == nil {
		t.Fatal("InjectionConfig is nil")
	}
	if got[0].InjectionConfig.HeaderName != "X-Api-Key" {
		t.Errorf("HeaderName = %q, want %q", got[0].InjectionConfig.HeaderName, "X-Api-Key")
	}
	if got[0].InjectionConfig.ValueFormat != "Token {value}" {
		t.Errorf("ValueFormat = %q, want %q", got[0].InjectionConfig.ValueFormat, "Token {value}")
	}
}

func TestMapper_OtherType_MultipleHosts(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(secretservice.NewRegistry())
	item := secret.ListItem{
		Name:  "my-token",
		Type:  "other",
		Hosts: []string{"api.example.com", "api2.example.com"},
	}

	got, err := mapper.Map(item, "my-key-123")
	if err != nil {
		t.Fatalf("Map() unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("Map() returned %d inputs, want 2", len(got))
	}

	if got[0].Name != "my-token-api-example-com" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "my-token-api-example-com")
	}
	if got[0].HostPattern != "api.example.com" {
		t.Errorf("got[0].HostPattern = %q, want %q", got[0].HostPattern, "api.example.com")
	}
	if got[1].Name != "my-token-api2-example-com" {
		t.Errorf("got[1].Name = %q, want %q", got[1].Name, "my-token-api2-example-com")
	}
	if got[1].HostPattern != "api2.example.com" {
		t.Errorf("got[1].HostPattern = %q, want %q", got[1].HostPattern, "api2.example.com")
	}
}

func TestMapper_OtherType_EmptyNameSegment(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(secretservice.NewRegistry())
	// "***" sanitizes to "" (all non-alphanumeric, trimmed to empty)
	item := secret.ListItem{
		Name:  "my-token",
		Type:  "other",
		Hosts: []string{"api.example.com", "***"},
	}

	_, err := mapper.Map(item, "val")
	if err == nil {
		t.Fatal("expected error for host that sanitizes to empty, got nil")
	}
}

func TestMapper_OtherType_DuplicateNameSegment(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(secretservice.NewRegistry())
	// Both hosts sanitize to "example-com", producing the same secret name
	item := secret.ListItem{
		Name:  "my-token",
		Type:  "other",
		Hosts: []string{"example.com", "example-com"},
	}

	_, err := mapper.Map(item, "val")
	if err == nil {
		t.Fatal("expected error for duplicate sanitized name, got nil")
	}
}

func TestMapper_OtherType_MinimalFields(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(secretservice.NewRegistry())
	item := secret.ListItem{
		Name: "other",
		Type: "other",
	}

	got, err := mapper.Map(item, "secret-val")
	if err != nil {
		t.Fatalf("Map() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Map() returned %d inputs, want 1", len(got))
	}

	if got[0].Name != "other" {
		t.Errorf("Name = %q, want %q", got[0].Name, "other")
	}
	if got[0].HostPattern != "*" {
		t.Errorf("HostPattern = %q, want %q", got[0].HostPattern, "*")
	}
	if got[0].PathPattern != "" {
		t.Errorf("PathPattern = %q, want empty", got[0].PathPattern)
	}
	if got[0].InjectionConfig != nil {
		t.Errorf("InjectionConfig should be nil for other type without header, got %+v", got[0].InjectionConfig)
	}
}

func TestMapper_OtherType_EmptyHosts(t *testing.T) {
	t.Parallel()

	mapper := NewSecretMapper(secretservice.NewRegistry())
	item := secret.ListItem{
		Name:  "my-token",
		Type:  "other",
		Hosts: []string{},
	}

	got, err := mapper.Map(item, "val")
	if err != nil {
		t.Fatalf("Map() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("Map() returned %d inputs, want 1", len(got))
	}
	if got[0].HostPattern != "*" {
		t.Errorf("HostPattern = %q, want %q for empty hosts", got[0].HostPattern, "*")
	}
}

func TestConvertTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"Bearer ${value}", "Bearer {value}"},
		{"${value}", "{value}"},
		{"no-placeholder", "no-placeholder"},
		{"", ""},
		{"${value} and ${value}", "{value} and {value}"},
	}

	for _, tt := range tests {
		if got := convertTemplate(tt.input); got != tt.want {
			t.Errorf("convertTemplate(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"api.example.com", "api-example-com"},
		{"*.example.com", "example-com"},
		{"api2.example.com", "api2-example-com"},
		{"api.example.com/v2", "api-example-com-v2"},
		{"already-safe", "already-safe"},
		{"UPPER.case.com", "UPPER-case-com"},
	}

	for _, tt := range tests {
		if got := sanitizeName(tt.input); got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
