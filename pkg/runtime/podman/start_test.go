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
	"github.com/kortex-hub/kortex-cli/pkg/steplogger"
)

func TestStart_ValidatesID(t *testing.T) {
	t.Parallel()

	t.Run("rejects empty ID", func(t *testing.T) {
		t.Parallel()

		p := &podmanRuntime{}

		_, err := p.Start(context.Background(), "")
		if err == nil {
			t.Fatal("Expected error for empty ID, got nil")
		}

		if !errors.Is(err, runtime.ErrInvalidParams) {
			t.Errorf("Expected ErrInvalidParams, got %v", err)
		}
	})
}

func TestStart_Success(t *testing.T) {
	t.Parallel()

	containerID := "abc123def456"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return container info
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		// Simulate podman inspect output
		output := fmt.Sprintf("%s|running|kdn-test\n", containerID)
		return []byte(output), nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	info, err := p.Start(context.Background(), containerID)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Verify Run was called to start the container
	fakeExec.AssertRunCalledWith(t, "start", containerID)

	// Verify Output was called to inspect the container
	fakeExec.AssertOutputCalledWith(t, "inspect", "--format", "{{.Id}}|{{.State.Status}}|{{.ImageName}}", containerID)

	// Verify returned info
	if info.ID != containerID {
		t.Errorf("Expected ID %s, got %s", containerID, info.ID)
	}
	if info.State != "running" {
		t.Errorf("Expected state 'running', got %s", info.State)
	}
	if info.Info["container_id"] != containerID {
		t.Errorf("Expected container_id %s, got %s", containerID, info.Info["container_id"])
	}
	if info.Info["image_name"] != "kdn-test" {
		t.Errorf("Expected image_name 'kdn-test', got %s", info.Info["image_name"])
	}
}

func TestStart_StartContainerFailure(t *testing.T) {
	t.Parallel()

	containerID := "abc123"
	fakeExec := exec.NewFake()

	// Set up RunFunc to return an error
	fakeExec.RunFunc = func(ctx context.Context, args ...string) error {
		return fmt.Errorf("container not found")
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	_, err := p.Start(context.Background(), containerID)
	if err == nil {
		t.Fatal("Expected error when start fails, got nil")
	}

	// Verify Run was called
	fakeExec.AssertRunCalledWith(t, "start", containerID)
}

func TestStart_InspectFailure(t *testing.T) {
	t.Parallel()

	containerID := "abc123"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return an error
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("inspect failed")
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	_, err := p.Start(context.Background(), containerID)
	if err == nil {
		t.Fatal("Expected error when inspect fails, got nil")
	}

	// Verify both Run and Output were called
	fakeExec.AssertRunCalledWith(t, "start", containerID)
	fakeExec.AssertOutputCalledWith(t, "inspect", "--format", "{{.Id}}|{{.State.Status}}|{{.ImageName}}", containerID)
}

func TestStart_StepLogger_Success(t *testing.T) {
	t.Parallel()

	containerID := "abc123def456"
	fakeExec := exec.NewFake()

	// Set up OutputFunc to return container info
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		output := fmt.Sprintf("%s|running|kdn-test\n", containerID)
		return []byte(output), nil
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	_, err := p.Start(ctx, containerID)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
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
			inProgress: "Starting container: abc123def456",
			completed:  "Container started",
		},
		{
			inProgress: "Verifying container status",
			completed:  "Container status verified",
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

func TestStart_StepLogger_FailOnStartContainer(t *testing.T) {
	t.Parallel()

	containerID := "abc123"
	fakeExec := exec.NewFake()

	// Set up RunFunc to return an error
	fakeExec.RunFunc = func(ctx context.Context, args ...string) error {
		return fmt.Errorf("container not found")
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	_, err := p.Start(ctx, containerID)
	if err == nil {
		t.Fatal("Expected Start() to fail, got nil")
	}

	// Verify Complete was called once (deferred call)
	if fakeLogger.completeCalls != 1 {
		t.Errorf("Expected Complete() to be called 1 time, got %d", fakeLogger.completeCalls)
	}

	// Verify Start was called once (start container step)
	if len(fakeLogger.startCalls) != 1 {
		t.Fatalf("Expected 1 Start() call, got %d", len(fakeLogger.startCalls))
	}

	if fakeLogger.startCalls[0].inProgress != "Starting container: abc123" {
		t.Errorf("Expected first step to be 'Starting container: abc123', got %q", fakeLogger.startCalls[0].inProgress)
	}

	// Verify Fail was called once
	if len(fakeLogger.failCalls) != 1 {
		t.Fatalf("Expected 1 Fail() call, got %d", len(fakeLogger.failCalls))
	}

	if fakeLogger.failCalls[0] == nil {
		t.Error("Expected Fail() to be called with non-nil error")
	}
}

func TestStart_StepLogger_FailOnGetContainerInfo(t *testing.T) {
	t.Parallel()

	containerID := "abc123"
	fakeExec := exec.NewFake()

	// Set up RunFunc to succeed, but OutputFunc to fail
	fakeExec.RunFunc = func(ctx context.Context, args ...string) error {
		return nil
	}
	fakeExec.OutputFunc = func(ctx context.Context, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("failed to inspect container")
	}

	p := newWithDeps(&fakeSystem{}, fakeExec).(*podmanRuntime)

	fakeLogger := &fakeStepLogger{}
	ctx := steplogger.WithLogger(context.Background(), fakeLogger)

	_, err := p.Start(ctx, containerID)
	if err == nil {
		t.Fatal("Expected Start() to fail, got nil")
	}

	// Verify Complete was called once (deferred call)
	if fakeLogger.completeCalls != 1 {
		t.Errorf("Expected Complete() to be called 1 time, got %d", fakeLogger.completeCalls)
	}

	// Verify Start was called twice (start container, verify status)
	if len(fakeLogger.startCalls) != 2 {
		t.Fatalf("Expected 2 Start() calls, got %d", len(fakeLogger.startCalls))
	}

	expectedSteps := []string{
		"Starting container: abc123",
		"Verifying container status",
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
