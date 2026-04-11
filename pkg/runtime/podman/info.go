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
	"fmt"
	"strings"

	api "github.com/openkaiden/kdn-api/cli/go"
	"github.com/openkaiden/kdn/pkg/logger"
	"github.com/openkaiden/kdn/pkg/runtime"
)

// mapPodmanPodState maps podman pod states to valid WorkspaceState values.
// Pod states use title case: https://docs.podman.io/en/latest/markdown/podman-pod-inspect.1.html
func mapPodmanPodState(podmanState string) api.WorkspaceState {
	switch podmanState {
	case "Running":
		return api.WorkspaceStateRunning
	case "Created", "Stopped", "Exited":
		return api.WorkspaceStateStopped
	case "Dead", "Degraded":
		return api.WorkspaceStateError
	default:
		return api.WorkspaceStateUnknown
	}
}

// Info retrieves information about a Podman runtime instance.
func (p *podmanRuntime) Info(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
	// Validate the ID parameter
	if id == "" {
		return runtime.RuntimeInfo{}, fmt.Errorf("%w: pod ID is required", runtime.ErrInvalidParams)
	}

	// Get pod information
	info, err := p.getPodInfo(ctx, id)
	if err != nil {
		return runtime.RuntimeInfo{}, err
	}

	return info, nil
}

// getPodInfo retrieves detailed information about a pod.
func (p *podmanRuntime) getPodInfo(ctx context.Context, id string) (runtime.RuntimeInfo, error) {
	// Use podman pod inspect to get pod details
	// Format: Name|State
	l := logger.FromContext(ctx)
	output, err := p.executor.Output(ctx, l.Stderr(), "pod", "inspect", "--format", "{{.Name}}|{{.State}}", id)
	if err != nil {
		return runtime.RuntimeInfo{}, fmt.Errorf("failed to inspect pod: %w", err)
	}

	// Parse the output
	fields := strings.Split(strings.TrimSpace(string(output)), "|")
	if len(fields) != 2 {
		return runtime.RuntimeInfo{}, fmt.Errorf("unexpected inspect output format: %s", string(output))
	}

	podN := fields[0]
	podmanState := fields[1]

	// Map podman pod state to valid WorkspaceState
	state := mapPodmanPodState(podmanState)

	// Build the info map
	info := map[string]string{
		"pod_name":            podN,
		"workspace_container": workspaceContainerName(podN),
	}

	return runtime.RuntimeInfo{
		ID:    podN,
		State: state,
		Info:  info,
	}, nil
}
