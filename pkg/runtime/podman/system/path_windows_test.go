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

//go:build windows

package system_test

import (
	"testing"

	"github.com/openkaiden/kdn/pkg/runtime/podman/system"
)

func TestHostPathToMachinePath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  string
	}{
		{`C:\Users\foo\.kdn\approval-handler\myworkspace`, "/mnt/c/Users/foo/.kdn/approval-handler/myworkspace"},
		{`D:\some\path`, "/mnt/d/some/path"},
		{`c:\lowercase\drive`, "/mnt/c/lowercase/drive"},
		{"/already/posix", "/already/posix"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := system.HostPathToMachinePath(tc.input); got != tc.want {
				t.Errorf("HostPathToMachinePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestMachinePathToHostPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  string
	}{
		{"/mnt/c/Users/foo/.kdn/approval-handler/myworkspace", `C:\Users\foo\.kdn\approval-handler\myworkspace`},
		{"/mnt/d/some/path", `D:\some\path`},
		{"/not/mnt/path", `\not\mnt\path`},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			if got := system.MachinePathToHostPath(tc.input); got != tc.want {
				t.Errorf("MachinePathToHostPath(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
