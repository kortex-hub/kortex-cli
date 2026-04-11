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

	"github.com/openkaiden/kdn/pkg/runtime"
	"github.com/openkaiden/kdn/pkg/runtime/podman/exec"
	"github.com/openkaiden/kdn/pkg/steplogger"
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

	podID := "kdn-test-workspace"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return pod info showing stopped state
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "pod" && args[1] == "inspect" {
			return []byte(fmt.Sprintf("%s|Stopped", podID)), nil
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	err := p.Remove(context.Background(), podID)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Verify Output was called to inspect the pod
	if len(fakeExec.OutputCalls) == 0 {
		t.Error("Expected Output to be called to inspect pod")
	}

	// Verify Run was called to remove the pod
	fakeExec.AssertRunCalledWith(t, "pod", "rm", podID)
}

func TestRemove_IdempotentWhenPodNotFound(t *testing.T) {
	t.Parallel()

	podID := "kdn-nonexistent"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return a "not found" error
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "pod" && args[1] == "inspect" {
			return nil, fmt.Errorf("failed to inspect pod: no such pod")
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	// Should succeed without error (idempotent)
	err := p.Remove(context.Background(), podID)
	if err != nil {
		t.Fatalf("Remove() should be idempotent for non-existent pods, got error: %v", err)
	}

	// Verify Output was called to check if pod exists
	if len(fakeExec.OutputCalls) == 0 {
		t.Error("Expected Output to be called to check if pod exists")
	}

	// Run should NOT be called since pod doesn't exist
	if len(fakeExec.RunCalls) > 0 {
		t.Error("Run should not be called for non-existent pod")
	}
}

func TestRemove_RejectsRunningPod(t *testing.T) {
	t.Parallel()

	podID := "kdn-running"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return pod info showing running state
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "pod" && args[1] == "inspect" {
			return []byte(fmt.Sprintf("%s|Running", podID)), nil
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	err := p.Remove(context.Background(), podID)
	if err == nil {
		t.Fatal("Expected error when removing running pod, got nil")
	}

	expectedMsg := "is still running, stop it first"
	if !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error message to contain %q, got: %v", expectedMsg, err)
	}

	// Verify Output was called to check pod state
	if len(fakeExec.OutputCalls) == 0 {
		t.Error("Expected Output to be called to check pod state")
	}

	// Run should NOT be called since pod is running
	if len(fakeExec.RunCalls) > 0 {
		t.Error("Run should not be called for running pod")
	}
}

func TestRemove_RemovePodFailure(t *testing.T) {
	t.Parallel()

	podID := "kdn-test"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return pod info showing stopped state
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(args) >= 2 && args[0] == "pod" && args[1] == "inspect" {
			return []byte(fmt.Sprintf("%s|Stopped", podID)), nil
		}
		return nil, fmt.Errorf("unexpected command: %v", args)
	}

	// Set up RunFunc to return an error when removing
	fakeExec.RunFunc = func(ctx context.Context, args ...string) error {
		return fmt.Errorf("device busy")
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	err := p.Remove(context.Background(), podID)
	if err == nil {
		t.Fatal("Expected error when remove fails, got nil")
	}

	// Verify Run was called with pod rm
	fakeExec.AssertRunCalledWith(t, "pod", "rm", podID)
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
			name:     "no such pod error",
			err:      fmt.Errorf("Error: no such pod abc123"),
			expected: true,
		},
		{
			name:     "pod not found error",
			err:      fmt.Errorf("pod not found: abc123"),
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
			name:     "failed to inspect pod with not found",
			err:      fmt.Errorf("failed to inspect pod: no such pod"),
			expected: true,
		},
		{
			name:     "failed to inspect pod with other error",
			err:      fmt.Errorf("failed to inspect pod: permission denied"),
			expected: true,
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

func TestRemove_StepLogger_Success(t *testing.T) {
	t.Parallel()

	podID := "kdn-test-workspace"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return stopped pod info
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		output := fmt.Sprintf("%s|Exited\n", podID)
		return []byte(output), nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	err := p.Remove(ctx, podID)
	if err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}

	// Verify Complete was called once (deferred call)
	if fakeLogger.completeCalls != 1 {
		t.Errorf("Expected Complete() to be called 1 time, got %d", fakeLogger.completeCalls)
	}

	// Verify no Fail calls
	if len(fakeLogger.failCalls) != 0 {
		t.Errorf("Expected no Fail() calls, got %d", len(fakeLogger.failCalls))
	}

	// Verify Start was called 2 times with correct messages
	expectedSteps := []stepCall{
		{
			inProgress: "Checking pod state",
			completed:  "Pod state checked",
		},
		{
			inProgress: "Removing pod: kdn-test-workspace",
			completed:  "Pod removed",
		},
	}

	if len(fakeLogger.startCalls) != len(expectedSteps) {
		t.Fatalf("Expected %d Start() calls, got %d", len(expectedSteps), len(fakeLogger.startCalls))
	}

	for i, expected := range expectedSteps {
		actual := fakeLogger.startCalls[i]
		if actual.inProgress != expected.inProgress {
			t.Errorf("Step %d: expected inProgress %q, got %q", i, expected.inProgress, actual.inProgress)
		}
		if actual.completed != expected.completed {
			t.Errorf("Step %d: expected completed %q, got %q", i, expected.completed, actual.completed)
		}
	}
}

func TestRemove_StepLogger_PodNotFound(t *testing.T) {
	t.Parallel()

	podID := "kdn-test"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return a "not found" error
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("no such pod: %s", podID)
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	err := p.Remove(ctx, podID)
	if err != nil {
		t.Fatalf("Remove() should be idempotent for not found, got error: %v", err)
	}

	// Verify Complete was called once (deferred call)
	if fakeLogger.completeCalls != 1 {
		t.Errorf("Expected Complete() to be called 1 time, got %d", fakeLogger.completeCalls)
	}

	// Verify Start was called once (checking pod state)
	if len(fakeLogger.startCalls) != 1 {
		t.Fatalf("Expected 1 Start() call, got %d", len(fakeLogger.startCalls))
	}

	if fakeLogger.startCalls[0].inProgress != "Checking pod state" {
		t.Errorf("Expected first step to be 'Checking pod state', got %q", fakeLogger.startCalls[0].inProgress)
	}

	// Verify no Fail calls (idempotent operation)
	if len(fakeLogger.failCalls) != 0 {
		t.Errorf("Expected no Fail() calls for not found (idempotent), got %d", len(fakeLogger.failCalls))
	}
}

func TestRemove_StepLogger_FailOnGetPodInfo(t *testing.T) {
	t.Parallel()

	podID := "kdn-test"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return malformed output (not an error that matches isNotFoundError)
	// This will cause getPodInfo to fail with a parsing error
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return []byte("invalid|output|extra"), nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	err := p.Remove(ctx, podID)
	if err == nil {
		t.Fatal("Expected Remove() to fail, got nil")
	}

	// Verify Complete was called once (deferred call)
	if fakeLogger.completeCalls != 1 {
		t.Errorf("Expected Complete() to be called 1 time, got %d", fakeLogger.completeCalls)
	}

	// Verify Start was called once (checking pod state)
	if len(fakeLogger.startCalls) != 1 {
		t.Fatalf("Expected 1 Start() call, got %d", len(fakeLogger.startCalls))
	}

	if fakeLogger.startCalls[0].inProgress != "Checking pod state" {
		t.Errorf("Expected first step to be 'Checking pod state', got %q", fakeLogger.startCalls[0].inProgress)
	}

	// Verify Fail was called once
	if len(fakeLogger.failCalls) != 1 {
		t.Fatalf("Expected 1 Fail() call, got %d", len(fakeLogger.failCalls))
	}

	if fakeLogger.failCalls[0] == nil {
		t.Error("Expected Fail() to be called with non-nil error")
	}
}

func TestRemove_StepLogger_FailOnRunningPod(t *testing.T) {
	t.Parallel()

	podID := "kdn-test"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return a running pod
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		output := fmt.Sprintf("%s|Running\n", podID)
		return []byte(output), nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	err := p.Remove(ctx, podID)
	if err == nil {
		t.Fatal("Expected Remove() to fail for running pod, got nil")
	}

	// Verify Complete was called once (deferred call)
	if fakeLogger.completeCalls != 1 {
		t.Errorf("Expected Complete() to be called 1 time, got %d", fakeLogger.completeCalls)
	}

	// Verify Start was called once (checking pod state)
	if len(fakeLogger.startCalls) != 1 {
		t.Fatalf("Expected 1 Start() call, got %d", len(fakeLogger.startCalls))
	}

	if fakeLogger.startCalls[0].inProgress != "Checking pod state" {
		t.Errorf("Expected first step to be 'Checking pod state', got %q", fakeLogger.startCalls[0].inProgress)
	}

	// Verify Fail was called once
	if len(fakeLogger.failCalls) != 1 {
		t.Fatalf("Expected 1 Fail() call, got %d", len(fakeLogger.failCalls))
	}

	if fakeLogger.failCalls[0] == nil {
		t.Error("Expected Fail() to be called with non-nil error")
	}
}

func TestRemove_StepLogger_FailOnRemovePod(t *testing.T) {
	t.Parallel()

	podID := "kdn-test"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return stopped pod info
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		output := fmt.Sprintf("%s|Exited\n", podID)
		return []byte(output), nil
	}

	// Set up RunFunc to fail on remove
	fakeExec.RunFunc = func(ctx context.Context, args ...string) error {
		if len(args) >= 2 && args[0] == "pod" && args[1] == "rm" {
			return fmt.Errorf("failed to remove pod")
		}
		return nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	err := p.Remove(ctx, podID)
	if err == nil {
		t.Fatal("Expected Remove() to fail, got nil")
	}

	// Verify Complete was called once (deferred call)
	if fakeLogger.completeCalls != 1 {
		t.Errorf("Expected Complete() to be called 1 time, got %d", fakeLogger.completeCalls)
	}

	// Verify Start was called twice (checking state, removing pod)
	if len(fakeLogger.startCalls) != 2 {
		t.Fatalf("Expected 2 Start() calls, got %d", len(fakeLogger.startCalls))
	}

	expectedSteps := []string{
		"Checking pod state",
		"Removing pod: kdn-test",
	}

	for i, expected := range expectedSteps {
		if fakeLogger.startCalls[i].inProgress != expected {
			t.Errorf("Step %d: expected %q, got %q", i, expected, fakeLogger.startCalls[i].inProgress)
		}
	}

	// Verify Fail was called once
	if len(fakeLogger.failCalls) != 1 {
		t.Fatalf("Expected 1 Fail() call, got %d", len(fakeLogger.failCalls))
	}

	if fakeLogger.failCalls[0] == nil {
		t.Error("Expected Fail() to be called with non-nil error")
	}
}
