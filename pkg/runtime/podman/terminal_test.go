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

	"github.com/kortex-hub/kortex-cli/pkg/runtime"
	"github.com/kortex-hub/kortex-cli/pkg/runtime/podman/exec"
)

func TestPodmanRuntime_Terminal(t *testing.T) {
	t.Parallel()

	t.Run("executes podman exec -it with command", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		err := rt.Terminal(ctx, "container123", []string{"bash"})
		if err != nil {
			t.Fatalf("Terminal() failed: %v", err)
		}

		// Verify RunInteractive was called with correct arguments
		expectedArgs := []string{"exec", "-it", "container123", "bash"}
		fakeExec.AssertRunInteractiveCalledWith(t, expectedArgs...)
	})

	t.Run("executes with multiple command arguments", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		err := rt.Terminal(ctx, "container123", []string{"claude-code", "--debug"})
		if err != nil {
			t.Fatalf("Terminal() failed: %v", err)
		}

		// Verify RunInteractive was called with correct arguments
		expectedArgs := []string{"exec", "-it", "container123", "claude-code", "--debug"}
		fakeExec.AssertRunInteractiveCalledWith(t, expectedArgs...)
	})

	t.Run("returns error when instance ID is empty", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		err := rt.Terminal(ctx, "", []string{"bash"})
		if err == nil {
			t.Fatal("Expected error for empty instance ID")
		}

		if !errors.Is(err, runtime.ErrInvalidParams) {
			t.Errorf("Expected ErrInvalidParams, got: %v", err)
		}
	})

	t.Run("returns error when command is empty", func(t *testing.T) {
		t.Parallel()

		fakeExec := exec.NewFake()
		rt := &podmanRuntime{
			executor: fakeExec,
		}

		ctx := context.Background()
		err := rt.Terminal(ctx, "container123", []string{})
		if err == nil {
			t.Fatal("Expected error for empty command")
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
		err := rt.Terminal(ctx, "container123", []string{"bash"})
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
