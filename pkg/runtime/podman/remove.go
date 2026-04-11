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
	"github.com/openkaiden/kdn/pkg/steplogger"
)

// Remove removes a Podman pod and its associated resources.
func (p *podmanRuntime) Remove(ctx context.Context, id string) error {
	stepLogger := steplogger.FromContext(ctx)
	defer stepLogger.Complete()

	// Validate the ID parameter
	if id == "" {
		return fmt.Errorf("%w: pod ID is required", runtime.ErrInvalidParams)
	}

	// Check if the pod exists and get its state
	stepLogger.Start("Checking pod state", "Pod state checked")
	info, err := p.getPodInfo(ctx, id)
	if err != nil {
		// If the pod doesn't exist, treat it as already removed (idempotent)
		if isNotFoundError(err) {
			return nil
		}
		stepLogger.Fail(err)
		return err
	}

	// Check if the pod is running
	if info.State == api.WorkspaceStateRunning {
		err := fmt.Errorf("pod %s is still running, stop it first", id)
		stepLogger.Fail(err)
		return err
	}

	// Remove the pod
	stepLogger.Start(fmt.Sprintf("Removing pod: %s", id), "Pod removed")
	if err := p.removeContainer(ctx, id); err != nil {
		stepLogger.Fail(err)
		return err
	}

	return nil
}

// removeContainer removes a podman pod by ID.
func (p *podmanRuntime) removeContainer(ctx context.Context, id string) error {
	l := logger.FromContext(ctx)
	if err := p.executor.Run(ctx, l.Stdout(), l.Stderr(), "pod", "rm", id); err != nil {
		return fmt.Errorf("failed to remove pod: %w", err)
	}
	return nil
}

// isNotFoundError checks if an error indicates that a pod or container was not found.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// Check for podman-specific "not found" error messages
	return strings.Contains(errMsg, "no such container") ||
		strings.Contains(errMsg, "no such pod") ||
		strings.Contains(errMsg, "pod not found") ||
		strings.Contains(errMsg, "no such object") ||
		strings.Contains(errMsg, "error getting container") ||
		strings.Contains(errMsg, "failed to inspect pod")
}
