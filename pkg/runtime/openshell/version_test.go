// Copyright 2026 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openshell

import (
	"strings"
	"testing"
)

func TestDefaultVersion_Format(t *testing.T) {
	t.Parallel()

	if !strings.HasPrefix(DefaultVersion, "v") {
		t.Errorf("DefaultVersion should start with 'v', got %q", DefaultVersion)
	}
}

func TestResolveVersion_Default(t *testing.T) {
	t.Parallel()

	rt := &openshellRuntime{}
	if got := rt.resolveVersion(); got != DefaultVersion {
		t.Errorf("resolveVersion() = %q, want %q", got, DefaultVersion)
	}
}

func TestResolveVersion_Override(t *testing.T) {
	t.Parallel()

	rt := &openshellRuntime{version: "v0.1.0"}
	if got := rt.resolveVersion(); got != "v0.1.0" {
		t.Errorf("resolveVersion() = %q, want %q", got, "v0.1.0")
	}
}
