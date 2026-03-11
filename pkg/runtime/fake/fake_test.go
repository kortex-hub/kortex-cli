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

package fake

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/kortex-hub/kortex-cli/pkg/runtime"
)

func TestFakeRuntime_Type(t *testing.T) {
	t.Parallel()

	rt := New()
	if rt.Type() != "fake" {
		t.Errorf("Expected type 'fake', got '%s'", rt.Type())
	}
}

func TestFakeRuntime_CreateStartStopRemove(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	params := runtime.CreateParams{
		Name:       "test-instance",
		SourcePath: "/path/to/source",
		ConfigPath: "/path/to/config",
	}

	// Create instance
	info, err := rt.Create(ctx, params)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if info.ID == "" {
		t.Error("Expected non-empty instance ID")
	}
	if info.State != "created" {
		t.Errorf("Expected state 'created', got '%s'", info.State)
	}
	if !strings.HasPrefix(info.ID, "fake-") {
		t.Errorf("Expected ID to start with 'fake-', got '%s'", info.ID)
	}

	instanceID := info.ID

	// Start instance
	info, err = rt.Start(ctx, instanceID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if info.State != "running" {
		t.Errorf("Expected state 'running', got '%s'", info.State)
	}

	// Stop instance
	err = rt.Stop(ctx, instanceID)
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	// Verify stopped state
	info, err = rt.Info(ctx, instanceID)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info.State != "stopped" {
		t.Errorf("Expected state 'stopped', got '%s'", info.State)
	}

	// Remove instance
	err = rt.Remove(ctx, instanceID)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify instance is gone
	_, err = rt.Info(ctx, instanceID)
	if !errors.Is(err, runtime.ErrInstanceNotFound) {
		t.Errorf("Expected ErrInstanceNotFound after remove, got %v", err)
	}
}

func TestFakeRuntime_InfoRetrievesCorrectState(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	params := runtime.CreateParams{
		Name:       "info-test",
		SourcePath: "/source",
		ConfigPath: "/config",
	}

	info, err := rt.Create(ctx, params)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	instanceID := info.ID

	// Info should return created state
	info, err = rt.Info(ctx, instanceID)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info.State != "created" {
		t.Errorf("Expected state 'created', got '%s'", info.State)
	}

	// Start and verify running state
	_, err = rt.Start(ctx, instanceID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	info, err = rt.Info(ctx, instanceID)
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}
	if info.State != "running" {
		t.Errorf("Expected state 'running', got '%s'", info.State)
	}

	// Verify info contains expected metadata
	if info.Info["source"] != "/source" {
		t.Errorf("Expected source '/source', got '%s'", info.Info["source"])
	}
	if info.Info["config"] != "/config" {
		t.Errorf("Expected config '/config', got '%s'", info.Info["config"])
	}
	if info.Info["created_at"] == "" {
		t.Error("Expected created_at timestamp")
	}
	if info.Info["started_at"] == "" {
		t.Error("Expected started_at timestamp")
	}
}

func TestFakeRuntime_DuplicateCreate(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	params := runtime.CreateParams{
		Name:       "duplicate-test",
		SourcePath: "/source",
		ConfigPath: "/config",
	}

	// Create first instance
	_, err := rt.Create(ctx, params)
	if err != nil {
		t.Fatalf("First create failed: %v", err)
	}

	// Try to create duplicate
	_, err = rt.Create(ctx, params)
	if err == nil {
		t.Fatal("Expected error when creating duplicate instance, got nil")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got '%s'", err.Error())
	}
}

func TestFakeRuntime_UnknownInstanceID(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	// Try to start non-existent instance
	_, err := rt.Start(ctx, "unknown-id")
	if !errors.Is(err, runtime.ErrInstanceNotFound) {
		t.Errorf("Expected ErrInstanceNotFound, got %v", err)
	}

	// Try to stop non-existent instance
	err = rt.Stop(ctx, "unknown-id")
	if !errors.Is(err, runtime.ErrInstanceNotFound) {
		t.Errorf("Expected ErrInstanceNotFound, got %v", err)
	}

	// Try to remove non-existent instance
	err = rt.Remove(ctx, "unknown-id")
	if !errors.Is(err, runtime.ErrInstanceNotFound) {
		t.Errorf("Expected ErrInstanceNotFound, got %v", err)
	}

	// Try to get info for non-existent instance
	_, err = rt.Info(ctx, "unknown-id")
	if !errors.Is(err, runtime.ErrInstanceNotFound) {
		t.Errorf("Expected ErrInstanceNotFound, got %v", err)
	}
}

func TestFakeRuntime_InvalidParams(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	tests := []struct {
		name   string
		params runtime.CreateParams
	}{
		{
			name: "missing name",
			params: runtime.CreateParams{
				SourcePath: "/source",
				ConfigPath: "/config",
			},
		},
		{
			name: "missing source path",
			params: runtime.CreateParams{
				Name:       "test",
				ConfigPath: "/config",
			},
		},
		{
			name: "missing config path",
			params: runtime.CreateParams{
				Name:       "test",
				SourcePath: "/source",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := rt.Create(ctx, tt.params)
			if !errors.Is(err, runtime.ErrInvalidParams) {
				t.Errorf("Expected ErrInvalidParams, got %v", err)
			}
		})
	}
}

func TestFakeRuntime_StateTransitionErrors(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	params := runtime.CreateParams{
		Name:       "state-test",
		SourcePath: "/source",
		ConfigPath: "/config",
	}

	info, err := rt.Create(ctx, params)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	instanceID := info.ID

	// Can't stop created instance
	err = rt.Stop(ctx, instanceID)
	if err == nil {
		t.Error("Expected error when stopping created instance")
	}

	// Can't remove running instance
	_, err = rt.Start(ctx, instanceID)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	err = rt.Remove(ctx, instanceID)
	if err == nil {
		t.Error("Expected error when removing running instance")
	}

	// Can't start already running instance
	_, err = rt.Start(ctx, instanceID)
	if err == nil {
		t.Error("Expected error when starting already running instance")
	}
}

func TestFakeRuntime_SequentialIDs(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	// Create multiple instances and verify sequential IDs
	var ids []string
	for i := 1; i <= 3; i++ {
		params := runtime.CreateParams{
			Name:       fmt.Sprintf("instance-%d", i),
			SourcePath: "/source",
			ConfigPath: "/config",
		}

		info, err := rt.Create(ctx, params)
		if err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}

		ids = append(ids, info.ID)
	}

	// Verify IDs are sequential
	expectedIDs := []string{"fake-001", "fake-002", "fake-003"}
	for i, id := range ids {
		if id != expectedIDs[i] {
			t.Errorf("Expected ID %s, got %s", expectedIDs[i], id)
		}
	}
}

func TestFakeRuntime_ParallelOperations(t *testing.T) {
	t.Parallel()

	rt := New()
	ctx := context.Background()

	const numInstances = 10
	var wg sync.WaitGroup
	wg.Add(numInstances)

	// Create multiple instances in parallel
	for i := 0; i < numInstances; i++ {
		i := i
		go func() {
			defer wg.Done()

			params := runtime.CreateParams{
				Name:       fmt.Sprintf("parallel-%d", i),
				SourcePath: "/source",
				ConfigPath: "/config",
			}

			info, err := rt.Create(ctx, params)
			if err != nil {
				t.Errorf("Create failed for instance %d: %v", i, err)
				return
			}

			// Start the instance
			_, err = rt.Start(ctx, info.ID)
			if err != nil {
				t.Errorf("Start failed for instance %d: %v", i, err)
			}
		}()
	}

	wg.Wait()
}
