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
	"github.com/spf13/cobra"
)

func TestInitCmd_PreRun(t *testing.T) {
	t.Parallel()

	t.Run("default arguments", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()

		c := &initCmd{}
		cmd := &cobra.Command{}
		cmd.Flags().String("workspace-configuration", "", "test flag")
		cmd.Flags().String("storage", tempDir, "test storage flag")

		args := []string{}

		err := c.preRun(cmd, args)
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}

		if c.manager == nil {
			t.Error("Expected manager to be created")
		}

		if c.sourcesDir != "." {
			t.Errorf("Expected sourcesDir to be '.', got %s", c.sourcesDir)
		}

		expectedAbsSourcesDir, _ := filepath.Abs(".")
		if c.absSourcesDir != expectedAbsSourcesDir {
			t.Errorf("Expected absSourcesDir to be %s, got %s", expectedAbsSourcesDir, c.absSourcesDir)
		}

		expectedConfigDir := filepath.Join(".", ".kortex")
		if c.workspaceConfigDir != expectedConfigDir {
			t.Errorf("Expected workspaceConfigDir to be %s, got %s", expectedConfigDir, c.workspaceConfigDir)
		}

		expectedAbsConfigDir, _ := filepath.Abs(expectedConfigDir)
		if c.absConfigDir != expectedAbsConfigDir {
			t.Errorf("Expected absConfigDir to be %s, got %s", expectedAbsConfigDir, c.absConfigDir)
		}
	})

	t.Run("with sources directory", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		sourcesDir := t.TempDir()

		c := &initCmd{}
		cmd := &cobra.Command{}
		cmd.Flags().String("workspace-configuration", "", "test flag")
		cmd.Flags().String("storage", tempDir, "test storage flag")

		args := []string{sourcesDir}

		err := c.preRun(cmd, args)
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}

		if c.manager == nil {
			t.Error("Expected manager to be created")
		}

		if c.sourcesDir != sourcesDir {
			t.Errorf("Expected sourcesDir to be %s, got %s", sourcesDir, c.sourcesDir)
		}

		expectedAbsSourcesDir, _ := filepath.Abs(sourcesDir)
		if c.absSourcesDir != expectedAbsSourcesDir {
			t.Errorf("Expected absSourcesDir to be %s, got %s", expectedAbsSourcesDir, c.absSourcesDir)
		}

		expectedConfigDir := filepath.Join(sourcesDir, ".kortex")
		if c.workspaceConfigDir != expectedConfigDir {
			t.Errorf("Expected workspaceConfigDir to be %s, got %s", expectedConfigDir, c.workspaceConfigDir)
		}

		expectedAbsConfigDir, _ := filepath.Abs(expectedConfigDir)
		if c.absConfigDir != expectedAbsConfigDir {
			t.Errorf("Expected absConfigDir to be %s, got %s", expectedAbsConfigDir, c.absConfigDir)
		}
	})

	t.Run("with workspace configuration flag", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		configDir := t.TempDir()

		c := &initCmd{
			workspaceConfigDir: configDir,
		}
		cmd := &cobra.Command{}
		cmd.Flags().String("workspace-configuration", "", "test flag")
		cmd.Flags().Set("workspace-configuration", configDir)
		cmd.Flags().String("storage", tempDir, "test storage flag")

		args := []string{}

		err := c.preRun(cmd, args)
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}

		if c.manager == nil {
			t.Error("Expected manager to be created")
		}

		if c.sourcesDir != "." {
			t.Errorf("Expected sourcesDir to be '.', got %s", c.sourcesDir)
		}

		if c.workspaceConfigDir != configDir {
			t.Errorf("Expected workspaceConfigDir to be %s, got %s", configDir, c.workspaceConfigDir)
		}

		expectedAbsConfigDir, _ := filepath.Abs(configDir)
		if c.absConfigDir != expectedAbsConfigDir {
			t.Errorf("Expected absConfigDir to be %s, got %s", expectedAbsConfigDir, c.absConfigDir)
		}
	})

	t.Run("with both arguments", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		sourcesDir := t.TempDir()
		configDir := t.TempDir()

		c := &initCmd{
			workspaceConfigDir: configDir,
		}
		cmd := &cobra.Command{}
		cmd.Flags().String("workspace-configuration", "", "test flag")
		cmd.Flags().Set("workspace-configuration", configDir)
		cmd.Flags().String("storage", tempDir, "test storage flag")

		args := []string{sourcesDir}

		err := c.preRun(cmd, args)
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}

		if c.manager == nil {
			t.Error("Expected manager to be created")
		}

		if c.sourcesDir != sourcesDir {
			t.Errorf("Expected sourcesDir to be %s, got %s", sourcesDir, c.sourcesDir)
		}

		expectedAbsSourcesDir, _ := filepath.Abs(sourcesDir)
		if c.absSourcesDir != expectedAbsSourcesDir {
			t.Errorf("Expected absSourcesDir to be %s, got %s", expectedAbsSourcesDir, c.absSourcesDir)
		}

		if c.workspaceConfigDir != configDir {
			t.Errorf("Expected workspaceConfigDir to be %s, got %s", configDir, c.workspaceConfigDir)
		}

		expectedAbsConfigDir, _ := filepath.Abs(configDir)
		if c.absConfigDir != expectedAbsConfigDir {
			t.Errorf("Expected absConfigDir to be %s, got %s", expectedAbsConfigDir, c.absConfigDir)
		}
	})

	t.Run("relative sources directory", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		relativePath := filepath.Join(".", "relative", "path")

		c := &initCmd{}
		cmd := &cobra.Command{}
		cmd.Flags().String("workspace-configuration", "", "test flag")
		cmd.Flags().String("storage", tempDir, "test storage flag")

		args := []string{relativePath}

		err := c.preRun(cmd, args)
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}

		if c.manager == nil {
			t.Error("Expected manager to be created")
		}

		if c.sourcesDir != relativePath {
			t.Errorf("Expected sourcesDir to be %s, got %s", relativePath, c.sourcesDir)
		}

		expectedAbsSourcesDir, _ := filepath.Abs(relativePath)
		if c.absSourcesDir != expectedAbsSourcesDir {
			t.Errorf("Expected absSourcesDir to be %s, got %s", expectedAbsSourcesDir, c.absSourcesDir)
		}

		expectedConfigDir := filepath.Join(relativePath, ".kortex")
		if c.workspaceConfigDir != expectedConfigDir {
			t.Errorf("Expected workspaceConfigDir to be %s, got %s", expectedConfigDir, c.workspaceConfigDir)
		}
	})
}

func TestInitCmd_E2E(t *testing.T) {
	t.Parallel()

	t.Run("registers workspace with default arguments", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"--storage", storageDir, "init"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		// Verify instance was created
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 1 {
			t.Fatalf("Expected 1 instance, got %d", len(instancesList))
		}

		inst := instancesList[0]

		// Verify instance has a non-empty ID
		if inst.GetID() == "" {
			t.Error("Expected instance to have a non-empty ID")
		}

		// Verify output contains only the ID (default non-verbose output)
		output := strings.TrimSpace(buf.String())
		if output != inst.GetID() {
			t.Errorf("Expected output to be just the ID %s, got: %s", inst.GetID(), output)
		}

		// Verify sources directory is current directory (absolute)
		expectedAbsSourcesDir, _ := filepath.Abs(".")
		if inst.GetSourceDir() != expectedAbsSourcesDir {
			t.Errorf("Expected source dir %s, got %s", expectedAbsSourcesDir, inst.GetSourceDir())
		}

		// Verify config directory defaults to .kortex in current directory
		expectedConfigDir := filepath.Join(expectedAbsSourcesDir, ".kortex")
		if inst.GetConfigDir() != expectedConfigDir {
			t.Errorf("Expected config dir %s, got %s", expectedConfigDir, inst.GetConfigDir())
		}

		// Verify paths are absolute
		if !filepath.IsAbs(inst.GetSourceDir()) {
			t.Errorf("Expected source dir to be absolute, got %s", inst.GetSourceDir())
		}
		if !filepath.IsAbs(inst.GetConfigDir()) {
			t.Errorf("Expected config dir to be absolute, got %s", inst.GetConfigDir())
		}
	})

	t.Run("registers workspace with custom sources directory", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir := t.TempDir()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"--storage", storageDir, "init", sourcesDir})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		// Verify instance was created with correct paths
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 1 {
			t.Fatalf("Expected 1 instance, got %d", len(instancesList))
		}

		inst := instancesList[0]

		// Verify instance has a non-empty ID
		if inst.GetID() == "" {
			t.Error("Expected instance to have a non-empty ID")
		}

		// Verify output contains only the ID (default non-verbose output)
		output := strings.TrimSpace(buf.String())
		if output != inst.GetID() {
			t.Errorf("Expected output to be just the ID %s, got: %s", inst.GetID(), output)
		}

		expectedAbsSourcesDir, _ := filepath.Abs(sourcesDir)
		if inst.GetSourceDir() != expectedAbsSourcesDir {
			t.Errorf("Expected source dir %s, got %s", expectedAbsSourcesDir, inst.GetSourceDir())
		}

		expectedConfigDir := filepath.Join(expectedAbsSourcesDir, ".kortex")
		if inst.GetConfigDir() != expectedConfigDir {
			t.Errorf("Expected config dir %s, got %s", expectedConfigDir, inst.GetConfigDir())
		}

		// Verify paths are absolute
		if !filepath.IsAbs(inst.GetSourceDir()) {
			t.Errorf("Expected source dir to be absolute, got %s", inst.GetSourceDir())
		}
		if !filepath.IsAbs(inst.GetConfigDir()) {
			t.Errorf("Expected config dir to be absolute, got %s", inst.GetConfigDir())
		}
	})

	t.Run("registers workspace with custom configuration directory", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		configDir := t.TempDir()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"--storage", storageDir, "init", "--workspace-configuration", configDir})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		// Verify instance was created with correct paths
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 1 {
			t.Fatalf("Expected 1 instance, got %d", len(instancesList))
		}

		inst := instancesList[0]

		// Verify instance has a non-empty ID
		if inst.GetID() == "" {
			t.Error("Expected instance to have a non-empty ID")
		}

		// Verify output contains only the ID (default non-verbose output)
		output := strings.TrimSpace(buf.String())
		if output != inst.GetID() {
			t.Errorf("Expected output to be just the ID %s, got: %s", inst.GetID(), output)
		}

		// Verify sources directory defaults to current directory
		expectedAbsSourcesDir, _ := filepath.Abs(".")
		if inst.GetSourceDir() != expectedAbsSourcesDir {
			t.Errorf("Expected source dir %s, got %s", expectedAbsSourcesDir, inst.GetSourceDir())
		}

		expectedAbsConfigDir, _ := filepath.Abs(configDir)
		if inst.GetConfigDir() != expectedAbsConfigDir {
			t.Errorf("Expected config dir %s, got %s", expectedAbsConfigDir, inst.GetConfigDir())
		}

		// Verify paths are absolute
		if !filepath.IsAbs(inst.GetSourceDir()) {
			t.Errorf("Expected source dir to be absolute, got %s", inst.GetSourceDir())
		}
		if !filepath.IsAbs(inst.GetConfigDir()) {
			t.Errorf("Expected config dir to be absolute, got %s", inst.GetConfigDir())
		}
	})

	t.Run("registers workspace with both custom directories", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir := t.TempDir()
		configDir := t.TempDir()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"--storage", storageDir, "init", sourcesDir, "--workspace-configuration", configDir})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		// Verify instance was created with correct paths
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 1 {
			t.Fatalf("Expected 1 instance, got %d", len(instancesList))
		}

		inst := instancesList[0]

		// Verify instance has a non-empty ID
		if inst.GetID() == "" {
			t.Error("Expected instance to have a non-empty ID")
		}

		// Verify output contains only the ID (default non-verbose output)
		output := strings.TrimSpace(buf.String())
		if output != inst.GetID() {
			t.Errorf("Expected output to be just the ID %s, got: %s", inst.GetID(), output)
		}

		expectedAbsSourcesDir, _ := filepath.Abs(sourcesDir)
		if inst.GetSourceDir() != expectedAbsSourcesDir {
			t.Errorf("Expected source dir %s, got %s", expectedAbsSourcesDir, inst.GetSourceDir())
		}

		expectedAbsConfigDir, _ := filepath.Abs(configDir)
		if inst.GetConfigDir() != expectedAbsConfigDir {
			t.Errorf("Expected config dir %s, got %s", expectedAbsConfigDir, inst.GetConfigDir())
		}

		// Verify paths are absolute
		if !filepath.IsAbs(inst.GetSourceDir()) {
			t.Errorf("Expected source dir to be absolute, got %s", inst.GetSourceDir())
		}
		if !filepath.IsAbs(inst.GetConfigDir()) {
			t.Errorf("Expected config dir to be absolute, got %s", inst.GetConfigDir())
		}
	})

	t.Run("registers multiple workspaces", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir1 := t.TempDir()
		sourcesDir2 := t.TempDir()

		// Register first workspace
		rootCmd1 := NewRootCmd()
		buf1 := new(bytes.Buffer)
		rootCmd1.SetOut(buf1)
		rootCmd1.SetArgs([]string{"--storage", storageDir, "init", sourcesDir1})

		err := rootCmd1.Execute()
		if err != nil {
			t.Fatalf("Execute() failed for first workspace: %v", err)
		}

		// Register second workspace
		rootCmd2 := NewRootCmd()
		buf2 := new(bytes.Buffer)
		rootCmd2.SetOut(buf2)
		rootCmd2.SetArgs([]string{"--storage", storageDir, "init", sourcesDir2})

		err = rootCmd2.Execute()
		if err != nil {
			t.Fatalf("Execute() failed for second workspace: %v", err)
		}

		// Verify both instances exist
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 2 {
			t.Fatalf("Expected 2 instances, got %d", len(instancesList))
		}

		// Verify both instances have unique IDs
		if instancesList[0].GetID() == "" || instancesList[1].GetID() == "" {
			t.Error("Expected both instances to have non-empty IDs")
		}
		if instancesList[0].GetID() == instancesList[1].GetID() {
			t.Error("Expected instances to have unique IDs")
		}

		// Verify both instances have correct source directories
		expectedAbsSourcesDir1, _ := filepath.Abs(sourcesDir1)
		expectedAbsSourcesDir2, _ := filepath.Abs(sourcesDir2)

		foundDir1 := false
		foundDir2 := false
		for _, inst := range instancesList {
			if inst.GetSourceDir() == expectedAbsSourcesDir1 {
				foundDir1 = true
				// Verify config dir for first workspace
				expectedConfigDir1 := filepath.Join(expectedAbsSourcesDir1, ".kortex")
				if inst.GetConfigDir() != expectedConfigDir1 {
					t.Errorf("Expected config dir %s for first workspace, got %s", expectedConfigDir1, inst.GetConfigDir())
				}
			}
			if inst.GetSourceDir() == expectedAbsSourcesDir2 {
				foundDir2 = true
				// Verify config dir for second workspace
				expectedConfigDir2 := filepath.Join(expectedAbsSourcesDir2, ".kortex")
				if inst.GetConfigDir() != expectedConfigDir2 {
					t.Errorf("Expected config dir %s for second workspace, got %s", expectedConfigDir2, inst.GetConfigDir())
				}
			}

			// Verify paths are absolute
			if !filepath.IsAbs(inst.GetSourceDir()) {
				t.Errorf("Expected source dir to be absolute, got %s", inst.GetSourceDir())
			}
			if !filepath.IsAbs(inst.GetConfigDir()) {
				t.Errorf("Expected config dir to be absolute, got %s", inst.GetConfigDir())
			}
		}

		if !foundDir1 {
			t.Errorf("Expected to find instance with source dir %s", expectedAbsSourcesDir1)
		}
		if !foundDir2 {
			t.Errorf("Expected to find instance with source dir %s", expectedAbsSourcesDir2)
		}
	})

	t.Run("registers workspace with verbose flag", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourcesDir := t.TempDir()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"--storage", storageDir, "init", sourcesDir, "--verbose"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		output := buf.String()

		// Verify verbose output contains expected strings
		if !strings.Contains(output, "Registered workspace:") {
			t.Errorf("Expected verbose output to contain 'Registered workspace:', got: %s", output)
		}
		if !strings.Contains(output, "ID:") {
			t.Errorf("Expected verbose output to contain 'ID:', got: %s", output)
		}
		if !strings.Contains(output, "Sources directory:") {
			t.Errorf("Expected verbose output to contain 'Sources directory:', got: %s", output)
		}
		if !strings.Contains(output, "Configuration directory:") {
			t.Errorf("Expected verbose output to contain 'Configuration directory:', got: %s", output)
		}

		// Verify instance was created with correct paths
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}

		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}

		if len(instancesList) != 1 {
			t.Fatalf("Expected 1 instance, got %d", len(instancesList))
		}

		inst := instancesList[0]

		// Verify verbose output contains the actual values
		expectedAbsSourcesDir, _ := filepath.Abs(sourcesDir)
		if !strings.Contains(output, expectedAbsSourcesDir) {
			t.Errorf("Expected verbose output to contain sources directory %s, got: %s", expectedAbsSourcesDir, output)
		}

		expectedConfigDir := filepath.Join(expectedAbsSourcesDir, ".kortex")
		if !strings.Contains(output, expectedConfigDir) {
			t.Errorf("Expected verbose output to contain config directory %s, got: %s", expectedConfigDir, output)
		}

		if !strings.Contains(output, inst.GetID()) {
			t.Errorf("Expected verbose output to contain instance ID %s, got: %s", inst.GetID(), output)
		}
	})
}
