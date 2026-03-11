/**********************************************************************
 * Copyright (C) 2026 Red Hat, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/

package cmd

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kortex-hub/kortex-cli/pkg/instances"
	"github.com/kortex-hub/kortex-cli/pkg/runtime/fake"
)

func TestWorkspaceRemoveCmd(t *testing.T) {
	t.Parallel()

	cmd := NewWorkspaceRemoveCmd()
	if cmd == nil {
		t.Fatal("NewWorkspaceRemoveCmd() returned nil")
	}

	if cmd.Use != "remove ID" {
		t.Errorf("Expected Use to be 'remove ID', got '%s'", cmd.Use)
	}
}

func TestWorkspaceRemoveCmd_PreRun(t *testing.T) {
	t.Parallel()

	t.Run("requires ID argument", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "remove", "--storage", storageDir})

		err := rootCmd.Execute()
		if err == nil {
			t.Fatal("Expected error when ID argument is missing, got nil")
		}

		if !strings.Contains(err.Error(), "accepts 1 arg(s), received 0") {
			t.Errorf("Expected error to contain 'accepts 1 arg(s), received 0', got: %v", err)
		}
	})

	t.Run("rejects multiple arguments", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "remove", "id1", "id2", "--storage", storageDir})

		err := rootCmd.Execute()
		if err == nil {
			t.Fatal("Expected error when multiple arguments provided, got nil")
		}

		if !strings.Contains(err.Error(), "accepts 1 arg(s), received 2") {
			t.Errorf("Expected error to contain 'accepts 1 arg(s), received 2', got: %v", err)
		}
	})

	t.Run("creates manager from storage flag", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir := t.TempDir()

		// Create a workspace first
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instance, err := instances.NewInstance(instances.NewInstanceParams{
			SourceDir: sourcesDir,
			ConfigDir: filepath.Join(sourcesDir, ".kortex"),
		})
		if err != nil {
			t.Fatalf("Failed to create instance: %v", err)
		}

		// Register fake runtime
		if err := manager.RegisterRuntime(fake.New()); err != nil {
			t.Fatalf("Failed to register fake runtime: %v", err)
		}

		addedInstance, err := manager.Add(instance, "fake")
		if err != nil {
			t.Fatalf("Failed to add instance: %v", err)
		}

		// Now remove it
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "remove", addedInstance.GetID(), "--storage", storageDir})

		err = rootCmd.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	})
}

func TestWorkspaceRemoveCmd_E2E(t *testing.T) {
	t.Parallel()

	t.Run("removes existing workspace by ID", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir := t.TempDir()

		// Create a workspace
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instance, err := instances.NewInstance(instances.NewInstanceParams{
			SourceDir: sourcesDir,
			ConfigDir: filepath.Join(sourcesDir, ".kortex"),
		})
		if err != nil {
			t.Fatalf("Failed to create instance: %v", err)
		}

		// Register fake runtime
		if err := manager.RegisterRuntime(fake.New()); err != nil {
			t.Fatalf("Failed to register fake runtime: %v", err)
		}

		addedInstance, err := manager.Add(instance, "fake")
		if err != nil {
			t.Fatalf("Failed to add instance: %v", err)
		}

		instanceID := addedInstance.GetID()

		// Remove the workspace
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "remove", instanceID, "--storage", storageDir})

		var output bytes.Buffer
		rootCmd.SetOut(&output)

		err = rootCmd.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify output is only the ID
		result := strings.TrimSpace(output.String())
		if result != instanceID {
			t.Errorf("Expected output to be '%s', got: '%s'", instanceID, result)
		}

		// Verify workspace is removed from storage
		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 0 {
			t.Errorf("Expected 0 instances after removal, got %d", len(instancesList))
		}

		// Verify Get returns not found
		_, err = manager.Get(instanceID)
		if err == nil {
			t.Error("Expected error when getting removed instance, got nil")
		}
		if err != instances.ErrInstanceNotFound {
			t.Errorf("Expected ErrInstanceNotFound, got: %v", err)
		}
	})

	t.Run("returns error for non-existent workspace ID", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "remove", "nonexistent-id", "--storage", storageDir})

		err := rootCmd.Execute()
		if err == nil {
			t.Fatal("Expected error for non-existent ID, got nil")
		}

		if !strings.Contains(err.Error(), "workspace not found") {
			t.Errorf("Expected error to contain 'workspace not found', got: %v", err)
		}
		if !strings.Contains(err.Error(), "workspace list") {
			t.Errorf("Expected error to contain 'workspace list', got: %v", err)
		}
	})

	t.Run("removes only specified workspace when multiple exist", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir1 := t.TempDir()
		sourcesDir2 := t.TempDir()

		// Create two workspaces
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instance1, err := instances.NewInstance(instances.NewInstanceParams{
			SourceDir: sourcesDir1,
			ConfigDir: filepath.Join(sourcesDir1, ".kortex"),
		})
		if err != nil {
			t.Fatalf("Failed to create instance 1: %v", err)
		}

		instance2, err := instances.NewInstance(instances.NewInstanceParams{
			SourceDir: sourcesDir2,
			ConfigDir: filepath.Join(sourcesDir2, ".kortex"),
		})
		if err != nil {
			t.Fatalf("Failed to create instance 2: %v", err)
		}

		// Register fake runtime
		if err := manager.RegisterRuntime(fake.New()); err != nil {
			t.Fatalf("Failed to register fake runtime: %v", err)
		}

		addedInstance1, err := manager.Add(instance1, "fake")
		if err != nil {
			t.Fatalf("Failed to add instance 1: %v", err)
		}

		addedInstance2, err := manager.Add(instance2, "fake")
		if err != nil {
			t.Fatalf("Failed to add instance 2: %v", err)
		}

		// Remove the first workspace
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "remove", addedInstance1.GetID(), "--storage", storageDir})

		err = rootCmd.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify only one workspace remains
		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 1 {
			t.Fatalf("Expected 1 instance after removal, got %d", len(instancesList))
		}

		// Verify the remaining workspace is instance2
		if instancesList[0].GetID() != addedInstance2.GetID() {
			t.Errorf("Expected remaining instance ID %s, got %s", addedInstance2.GetID(), instancesList[0].GetID())
		}

		// Verify instance1 is removed
		_, err = manager.Get(addedInstance1.GetID())
		if err != instances.ErrInstanceNotFound {
			t.Errorf("Expected ErrInstanceNotFound for removed instance, got: %v", err)
		}

		// Verify instance2 still exists
		retrievedInstance, err := manager.Get(addedInstance2.GetID())
		if err != nil {
			t.Fatalf("Expected no error getting instance 2, got %v", err)
		}
		if retrievedInstance.GetID() != addedInstance2.GetID() {
			t.Errorf("Expected ID %s, got %s", addedInstance2.GetID(), retrievedInstance.GetID())
		}
	})

	t.Run("remove command alias works", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir := t.TempDir()

		// Create a workspace
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instance, err := instances.NewInstance(instances.NewInstanceParams{
			SourceDir: sourcesDir,
			ConfigDir: filepath.Join(sourcesDir, ".kortex"),
		})
		if err != nil {
			t.Fatalf("Failed to create instance: %v", err)
		}

		// Register fake runtime
		if err := manager.RegisterRuntime(fake.New()); err != nil {
			t.Fatalf("Failed to register fake runtime: %v", err)
		}

		addedInstance, err := manager.Add(instance, "fake")
		if err != nil {
			t.Fatalf("Failed to add instance: %v", err)
		}

		instanceID := addedInstance.GetID()

		// Use the alias command 'remove' instead of 'workspace remove'
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"remove", instanceID, "--storage", storageDir})

		var output bytes.Buffer
		rootCmd.SetOut(&output)

		err = rootCmd.Execute()
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify output is only the ID
		result := strings.TrimSpace(output.String())
		if result != instanceID {
			t.Errorf("Expected output to be '%s', got: '%s'", instanceID, result)
		}

		// Verify workspace is removed
		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 0 {
			t.Errorf("Expected 0 instances after removal, got %d", len(instancesList))
		}
	})
}
