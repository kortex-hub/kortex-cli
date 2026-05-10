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

package runtime

import (
	"net"
	"testing"

	workspace "github.com/openkaiden/kdn-api/workspace-configuration/go"
)

func TestCollectPorts_FromWorkspaceConfig(t *testing.T) {
	t.Parallel()

	ports := []int{8080, 3000}
	params := CreateParams{
		Agent: "claude",
		WorkspaceConfig: &workspace.WorkspaceConfiguration{
			Ports: &ports,
		},
	}

	result := CollectPorts(params)
	if len(result) != 2 {
		t.Fatalf("Expected 2 ports, got %d", len(result))
	}
	if result[0] != 8080 || result[1] != 3000 {
		t.Errorf("Expected [8080, 3000], got %v", result)
	}
}

func TestCollectPorts_DefaultPorts(t *testing.T) {
	t.Parallel()

	params := CreateParams{
		DefaultPorts: []int{18789},
	}

	result := CollectPorts(params)
	if len(result) != 1 {
		t.Fatalf("Expected 1 port, got %d", len(result))
	}
	if result[0] != 18789 {
		t.Errorf("Expected port 18789, got %d", result[0])
	}
}

func TestCollectPorts_Deduplicates(t *testing.T) {
	t.Parallel()

	ports := []int{18789, 3000}
	params := CreateParams{
		DefaultPorts: []int{18789},
		WorkspaceConfig: &workspace.WorkspaceConfiguration{
			Ports: &ports,
		},
	}

	result := CollectPorts(params)
	if len(result) != 2 {
		t.Fatalf("Expected 2 ports (deduplicated), got %d", len(result))
	}
	if result[0] != 18789 || result[1] != 3000 {
		t.Errorf("Expected [18789, 3000], got %v", result)
	}
}

func TestCollectPorts_NilConfig(t *testing.T) {
	t.Parallel()

	params := CreateParams{
		Agent: "claude",
	}

	result := CollectPorts(params)
	if len(result) != 0 {
		t.Errorf("Expected no ports for claude without config, got %v", result)
	}
}

func TestIsPortFree_Free(t *testing.T) {
	t.Parallel()

	// Get a free port, close it, then verify IsPortFree returns true
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to get a free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	if !IsPortFree(port) {
		t.Errorf("Expected port %d to be free after closing listener", port)
	}
}

func TestIsPortFree_Occupied(t *testing.T) {
	t.Parallel()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port
	if IsPortFree(port) {
		t.Errorf("Expected port %d to be occupied while listener is active", port)
	}
}
