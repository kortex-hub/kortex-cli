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
	"testing"

	"github.com/openkaiden/kdn/pkg/runtime"
	"github.com/openkaiden/kdn/pkg/runtime/podman/exec"
)

func TestPodmanRuntime_Terminal(t *testing.T) {
	t.Parallel()

	t.Run("executes podman exec -it on workspace container with command", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		// instanceID is the pod name; terminal should exec into the workspace container
		podID := "kdn-test-workspace"
		err := rt.Terminal(ctx, podID, "test-agent", []string{"bash"})
		if err != nil {
			t.Fatalf("Terminal() failed: %v", err)
		}

		// Verify RunInteractive was called targeting the workspace container, not the pod name
		wsContainer := workspaceContainerName(podID)
		expectedArgs := []string{"exec", "-it", wsContainer, "bash"}
		fakeExec.AssertRunInteractiveCalledWith(t, expectedArgs...)
	})

	t.Run("executes with multiple command arguments", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		podID := "kdn-test-workspace"
		err := rt.Terminal(ctx, podID, "test-agent", []string{"claude-code", "--debug"})
		if err != nil {
			t.Fatalf("Terminal() failed: %v", err)
		}

		// Verify RunInteractive was called targeting the workspace container
		wsContainer := workspaceContainerName(podID)
		expectedArgs := []string{"exec", "-it", wsContainer, "claude-code", "--debug"}
		fakeExec.AssertRunInteractiveCalledWith(t, expectedArgs...)
	})

	t.Run("returns error when instance ID is empty", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		err := rt.Terminal(ctx, "", "test-agent", []string{"bash"})
		if err == nil {
			t.Fatal("Expected error for empty instance ID")
		}

		if !errors.Is(err, runtime.ErrInvalidParams) {
			t.Errorf("Expected ErrInvalidParams, got: %v", err)
		}
	})

	t.Run("uses agent terminal command when command is empty", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
			config:   &fakeConfig{}, // fakeConfig returns ["claude"] as terminal command
		}

		ctx := context.Background()
		podID := "kdn-test-workspace"
		err := rt.Terminal(ctx, podID, "test-agent", []string{})
		if err != nil {
			t.Fatalf("Terminal() failed: %v", err)
		}

		// Verify RunInteractive was called targeting the workspace container with agent's terminal command
		wsContainer := workspaceContainerName(podID)
		expectedArgs := []string{"exec", "-it", wsContainer, "claude"}
		fakeExec.AssertRunInteractiveCalledWith(t, expectedArgs...)
	})

	t.Run("returns error when agent is empty and command is empty", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		err := rt.Terminal(ctx, "kdn-test-workspace", "", []string{})
		if err == nil {
			t.Fatal("Expected error for empty agent and empty command")
		}

		if !errors.Is(err, runtime.ErrInvalidParams) {
			t.Errorf("Expected ErrInvalidParams, got: %v", err)
		}
	})

	t.Run("propagates executor error", func(t *testing.T) {
		t.Parallel()

		expectedErr := errors.New("exec failed")
		fakeExec := exec.NewFake()
		fakeExec.RunInteractiveFunc = func(ctx context.Context, args ...string) error {
			return expectedErr
		}

		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		err := rt.Terminal(ctx, "kdn-test-workspace", "test-agent", []string{"bash"})
		if err == nil {
			t.Fatal("Expected error to be propagated")
		}

		if !errors.Is(err, expectedErr) {
			t.Errorf("Expected error %v, got: %v", expectedErr, err)
		}
	})
}

func TestPodmanRuntime_ImplementsTerminalInterface(t *testing.T) {
	t.Parallel()

	var _ runtime.Terminal = (*podmanRuntime)(nil)
}
