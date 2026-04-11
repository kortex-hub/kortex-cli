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

package podman

import (
	"context"
	"errors"
	"fmt"
	"testing"

	api "github.com/openkaiden/kdn-api/cli/go"
	"github.com/openkaiden/kdn/pkg/runtime"
	"github.com/openkaiden/kdn/pkg/runtime/podman/exec"
)

func TestInfo_ValidatesID(t *testing.T) {
	t.Parallel()

	t.Run("rejects empty ID", func(t *testing.T) {
		t.Parallel()

		p := &podmanRuntime{}

		_, err := p.Info(context.Background(), "")
		if err == nil {
			t.Fatal("Expected error for empty ID, got nil")
		}

		if !errors.Is(err, runtime.ErrInvalidParams) {
			t.Errorf("Expected ErrInvalidParams, got %v", err)
		}
	})
}

func TestInfo_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		podID         string
		output        string
		expectedState api.WorkspaceState
	}{
		{
			name:          "running pod",
			podID:         "kdn-test",
			output:        "kdn-test|Running\n",
			expectedState: api.WorkspaceStateRunning,
		},
		{
			name:          "stopped pod",
			podID:         "kdn-stopped",
			output:        "kdn-stopped|Stopped\n",
			expectedState: api.WorkspaceStateStopped,
		},
		{
			name:          "created pod",
			podID:         "kdn-new",
			output:        "kdn-new|Created\n",
			expectedState: api.WorkspaceStateStopped,
		},
		{
			name:          "exited pod",
			podID:         "kdn-exited",
			output:        "kdn-exited|Exited\n",
			expectedState: api.WorkspaceStateStopped,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeExec := exec.NewFake()
			fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
				return []byte(tt.output), nil
			}

			p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

			info, err := p.Info(context.Background(), tt.podID)
			if err != nil {
				t.Fatalf("Info() failed: %v", err)
			}

			// Verify Output was called with pod inspect args
			fakeExec.AssertOutputCalledWith(t, "pod", "inspect", "--format", "{{.Name}}|{{.State}}", tt.podID)

			// Verify returned info
			if info.ID != tt.podID {
				t.Errorf("Expected ID %s, got %s", tt.podID, info.ID)
			}
			if info.State != tt.expectedState {
				t.Errorf("Expected state %s, got %s", tt.expectedState, info.State)
			}
			if info.Info["pod_name"] != tt.podID {
				t.Errorf("Expected pod_name %s, got %s", tt.podID, info.Info["pod_name"])
			}
			expectedWsContainer := workspaceContainerName(tt.podID)
			if info.Info["workspace_container"] != expectedWsContainer {
				t.Errorf("Expected workspace_container %s, got %s", expectedWsContainer, info.Info["workspace_container"])
			}
		})
	}
}

func TestInfo_InspectFailure(t *testing.T) {
	t.Parallel()

	podID := "kdn-test"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return an error
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("pod not found")
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	_, err := p.Info(context.Background(), podID)
	if err == nil {
		t.Fatal("Expected error when inspect fails, got nil")
	}

	// Verify Output was called with pod inspect args
	fakeExec.AssertOutputCalledWith(t, "pod", "inspect", "--format", "{{.Name}}|{{.State}}", podID)
}

func TestInfo_MalformedOutput(t *testing.T) {
	t.Parallel()

	podID := "kdn-test"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return malformed output (missing pipe separator)
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return []byte("invalid-output-without-pipes\n"), nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	_, err := p.Info(context.Background(), podID)
	if err == nil {
		t.Fatal("Expected error for malformed output, got nil")
	}

	// Verify Output was called with pod inspect args
	fakeExec.AssertOutputCalledWith(t, "pod", "inspect", "--format", "{{.Name}}|{{.State}}", podID)
}

func TestGetPodInfo_ParsesOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		podID         string
		output        string
		expectedState api.WorkspaceState
	}{
		{
			name:          "running pod",
			podID:         "kdn-test",
			output:        "kdn-test|Running\n",
			expectedState: api.WorkspaceStateRunning,
		},
		{
			name:          "stopped pod",
			podID:         "kdn-stopped",
			output:        "kdn-stopped|Stopped\n",
			expectedState: api.WorkspaceStateStopped,
		},
		{
			name:          "created pod",
			podID:         "kdn-new",
			output:        "kdn-new|Created\n",
			expectedState: api.WorkspaceStateStopped,
		},
		{
			name:          "exited pod",
			podID:         "kdn-exited",
			output:        "kdn-exited|Exited\n",
			expectedState: api.WorkspaceStateStopped,
		},
		{
			name:          "dead pod",
			podID:         "kdn-dead",
			output:        "kdn-dead|Dead\n",
			expectedState: api.WorkspaceStateError,
		},
		{
			name:          "degraded pod",
			podID:         "kdn-degraded",
			output:        "kdn-degraded|Degraded\n",
			expectedState: api.WorkspaceStateError,
		},
		{
			name:          "unknown pod state",
			podID:         "kdn-weird",
			output:        "kdn-weird|SomeFutureState\n",
			expectedState: api.WorkspaceStateUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fakeExec := exec.NewFake()
			fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
				return []byte(tt.output), nil
			}

			p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

			info, err := p.getPodInfo(context.Background(), tt.podID)
			if err != nil {
				t.Fatalf("getPodInfo() failed: %v", err)
			}

			if info.State != tt.expectedState {
				t.Errorf("Expected state %s, got %s", tt.expectedState, info.State)
			}
			if info.Info["pod_name"] != tt.podID {
				t.Errorf("Expected pod_name %s, got %s", tt.podID, info.Info["pod_name"])
			}
		})
	}
}

func TestGetPodInfo_MalformedOutput(t *testing.T) {
	t.Parallel()

	fakeExec := exec.NewFake()
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return []byte("invalid-output-without-pipes\n"), nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	_, err := p.getPodInfo(context.Background(), "kdn-test")
	if err == nil {
		t.Fatal("Expected error for malformed output, got nil")
	}
}
