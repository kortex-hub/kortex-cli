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
	"testing"
)

func TestNewSecretService(t *testing.T) {
	t.Parallel()

	svc := NewSecretService(
		"github",
		[]string{"api.github.com"},
		"/path",
		[]string{"GH_TOKEN", "GITHUB_TOKEN"},
		"Authorization",
		"Bearer ${value}",
		"GitHub API token",
	)

	if svc == nil {
		t.Fatal("NewSecretService() returned nil")
	}

	if svc.Name() != "github" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "github")
	}
	if svc.Description() != "GitHub API token" {
		t.Errorf("Description() = %q, want %q", svc.Description(), "GitHub API token")
	}
	if len(svc.HostsPatterns()) != 1 || svc.HostsPatterns()[0] != "api.github.com" {
		t.Errorf("HostsPatterns() = %v, want %v", svc.HostsPatterns(), []string{"api.github.com"})
	}
	if svc.Path() != "/path" {
		t.Errorf("Path() = %q, want %q", svc.Path(), "/path")
	}
	if len(svc.EnvVars()) != 2 || svc.EnvVars()[0] != "GH_TOKEN" || svc.EnvVars()[1] != "GITHUB_TOKEN" {
		t.Errorf("EnvVars() = %v, want %v", svc.EnvVars(), []string{"GH_TOKEN", "GITHUB_TOKEN"})
	}
	if svc.HeaderName() != "Authorization" {
		t.Errorf("HeaderName() = %q, want %q", svc.HeaderName(), "Authorization")
	}
	if svc.HeaderTemplate() != "Bearer ${value}" {
		t.Errorf("HeaderTemplate() = %q, want %q", svc.HeaderTemplate(), "Bearer ${value}")
	}
}

func TestNewSecretService_OptionalFieldsEmpty(t *testing.T) {
	t.Parallel()

	svc := NewSecretService("minimal", nil, "", nil, "X-Token", "", "")

	if svc.Name() != "minimal" {
		t.Errorf("Name() = %q, want %q", svc.Name(), "minimal")
	}
	if svc.Description() != "" {
		t.Errorf("Description() = %q, want empty string", svc.Description())
	}
	if svc.HostsPatterns() != nil {
		t.Errorf("HostsPatterns() = %v, want nil", svc.HostsPatterns())
	}
	if svc.Path() != "" {
		t.Errorf("Path() = %q, want empty string", svc.Path())
	}
	if svc.EnvVars() != nil {
		t.Errorf("EnvVars() = %v, want nil", svc.EnvVars())
	}
	if svc.HeaderName() != "X-Token" {
		t.Errorf("HeaderName() = %q, want %q", svc.HeaderName(), "X-Token")
	}
	if svc.HeaderTemplate() != "" {
		t.Errorf("HeaderTemplate() = %q, want empty string", svc.HeaderTemplate())
	}
}
