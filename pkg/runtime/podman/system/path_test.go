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

//go:build !windows

package system_test

import (
	"testing"

	"github.com/openkaiden/kdn/pkg/runtime/podman/system"
)

func TestHostPathToMachinePath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
	}{
		{"/home/user/.kdn/approval-handler/myworkspace"},
		{"/tmp/foo/bar"},
		{"relative/path"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := system.HostPathToMachinePath(tc.input); got != tc.input {
				t.Errorf("HostPathToMachinePath(%q) = %q, want %q", tc.input, got, tc.input)
			}
		})
	}
}

func TestMachinePathToHostPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
	}{
		{"/home/user/.kdn/approval-handler/myworkspace"},
		{"/tmp/foo/bar"},
		{"relative/path"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := system.MachinePathToHostPath(tc.input); got != tc.input {
				t.Errorf("MachinePathToHostPath(%q) = %q, want %q", tc.input, got, tc.input)
			}
		})
	}
}
