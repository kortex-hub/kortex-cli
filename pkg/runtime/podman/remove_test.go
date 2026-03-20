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

	"github.com/kortex-hub/kortex-cli/pkg/runtime"
	"github.com/kortex-hub/kortex-cli/pkg/runtime/podman/exec"
)

func TestRemove_ValidatesID(t *testing.T) {
	t.Parallel()

	t.Run("rejects empty ID", func(t *testing.T) {
		t.Parallel()

		p := &podmanRuntime{}

		err := p.Remove(context.Background(), "")
		if err == nil {
			t.Fatal("Expected error for empty ID, got nil")
		}

		if !errors.Is(err, runtime.ErrInvalidParams) {
			t.Errorf("Expected ErrInvalidParams, got %v", err)
		}
	})
}

func TestRemove_Success(t *testing.T) {
	t.Parallel()

	containerID := "abc123def456"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return container info showing stopped state
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "inspect" {
			// Return a stopped container
			return []byte(fmt.Sprintf("%s|stopped|kortex-cli-test", containerID)), nil
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	err := p.Remove(context.Background(), containerID)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Verify Output was called to inspect the container
	if len(fakeExec.OutputCalls) == 0 {
		t.Error("Expected Output to be called to inspect container")
	}

	// Verify Run was called to remove the container
	fakeExec.AssertRunCalledWith(t, "rm", containerID)
}

func TestRemove_IdempotentWhenContainerNotFound(t *testing.T) {
	t.Parallel()

	containerID := "nonexistent"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return a "not found" error
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "inspect" {
			return nil, fmt.Errorf("failed to inspect container: no such container")
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	// Should succeed without error (idempotent)
	err := p.Remove(context.Background(), containerID)
	if err != nil {
		t.Fatalf("Remove() should be idempotent for non-existent containers, got error: %v", err)
	}

	// Verify Output was called to check if container exists
	if len(fakeExec.OutputCalls) == 0 {
		t.Error("Expected Output to be called to check if container exists")
	}

	// Run should NOT be called since container doesn't exist
	if len(fakeExec.RunCalls) > 0 {
		t.Error("Run should not be called for non-existent container")
	}
}

func TestRemove_RejectsRunningContainer(t *testing.T) {
	t.Parallel()

	containerID := "running123"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return container info showing running state
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "inspect" {
			// Return a running container
			return []byte(fmt.Sprintf("%s|running|kortex-cli-test", containerID)), nil
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	err := p.Remove(context.Background(), containerID)
	if err == nil {
		t.Fatal("Expected error when removing running container, got nil")
	}

	expectedMsg := "is still running, stop it first"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedMsg, err)
	}

	// Verify Output was called to check container state
	if len(fakeExec.OutputCalls) == 0 {
		t.Error("Expected Output to be called to check container state")
	}

	// Run should NOT be called since container is running
	if len(fakeExec.RunCalls) > 0 {
		t.Error("Run should not be called for running container")
	}
}

func TestRemove_RemoveContainerFailure(t *testing.T) {
	t.Parallel()

	containerID := "abc123"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return container info showing stopped state
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "inspect" {
			return []byte(fmt.Sprintf("%s|stopped|kortex-cli-test", containerID)), nil
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	// Set up RunFunc to return an error when removing
	fakeExec.RunFunc = func(ctx context.Context, args ...string) error {
		return fmt.Errorf("device busy")
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	err := p.Remove(context.Background(), containerID)
	if err == nil {
		t.Fatal("Expected error when remove fails, got nil")
	}

	// Verify Run was called
	fakeExec.AssertRunCalledWith(t, "rm", containerID)
}

func TestRemove_InspectFailureWithUnexpectedError(t *testing.T) {
	t.Parallel()

	containerID := "abc123"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return an unexpected error (not a "not found" error)
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 1 && args[0] == "inspect" {
			return nil, fmt.Errorf("permission denied")
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	err := p.Remove(context.Background(), containerID)
	if err == nil {
		t.Fatal("Expected error when inspect fails with unexpected error, got nil")
	}

	expectedMsg := "permission denied"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedMsg, err)
	}

	// Verify Output was called
	if len(fakeExec.OutputCalls) == 0 {
		t.Error("Expected Output to be called")
	}

	// Run should NOT be called since inspect failed
	if len(fakeExec.RunCalls) > 0 {
		t.Error("Run should not be called when inspect fails")
	}
}

func TestIsNotFoundError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "no such container error",
			err:      fmt.Errorf("Error: no such container abc123"),
			expected: true,
		},
		{
			name:     "no such object error",
			err:      fmt.Errorf("Error: no such object: abc123"),
			expected: true,
		},
		{
			name:     "error getting container",
			err:      fmt.Errorf("error getting container abc123"),
			expected: true,
		},
		{
			name:     "failed to inspect container with not found",
			err:      fmt.Errorf("failed to inspect container: no such container"),
			expected: true,
		},
		{
			name:     "failed to inspect container with other error",
			err:      fmt.Errorf("failed to inspect container: permission denied"),
			expected: false,
		},
		{
			name:     "other error",
			err:      fmt.Errorf("permission denied"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isNotFoundError(tt.err)
			if result != tt.expected {
				t.Errorf("isNotFoundError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
