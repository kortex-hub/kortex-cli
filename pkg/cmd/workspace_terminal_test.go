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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kortex-hub/kortex-cli/pkg/cmd/testutil"
	"github.com/kortex-hub/kortex-cli/pkg/instances"
	"github.com/spf13/cobra"
)

func TestWorkspaceTerminalCmd(t *testing.T) {
	t.Parallel()

	cmd := NewWorkspaceTerminalCmd()
	if cmd == nil {
		t.Fatal("NewWorkspaceTerminalCmd() returned nil")
	}

	if cmd.Use != "terminal NAME|ID [COMMAND...]" {
		t.Errorf("Expected Use to be 'terminal NAME|ID [COMMAND...]', got '%s'", cmd.Use)
	}
}

func TestWorkspaceTerminalCmd_PreRun(t *testing.T) {
	t.Parallel()

	t.Run("extracts id from args and creates manager", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()

		c := &workspaceTerminalCmd{}
		cmd := &cobra.Command{}
		cmd.Flags().String("storage", storageDir, "test storage flag")

		args := []string{"test-workspace-id"}

		err := c.preRun(cmd, args)
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}

		if c.manager == nil {
			t.Error("Expected manager to be created")
		}

		if c.nameOrID != "test-workspace-id" {
			t.Errorf("Expected id to be 'test-workspace-id', got %s", c.nameOrID)
		}

		// Verify command is empty when no command args provided
		// The runtime will choose the agent's terminal command
		if len(c.command) != 0 {
			t.Errorf("Expected empty command [], got %v", c.command)
		}
	})

	t.Run("handles id with command args", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()

		c := &workspaceTerminalCmd{}
		cmd := &cobra.Command{}
		cmd.Flags().String("storage", storageDir, "test storage flag")

		// args contains ID and command
		args := []string{"test-id", "bash", "-l"}

		err := c.preRun(cmd, args)
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}

		if c.nameOrID != "test-id" {
			t.Errorf("Expected id to be 'test-id', got %s", c.nameOrID)
		}

		// Verify command was extracted in preRun
		if len(c.command) != 2 {
			t.Errorf("Expected command length 2, got %d", len(c.command))
		}
		if len(c.command) >= 2 && (c.command[0] != "bash" || c.command[1] != "-l") {
			t.Errorf("Expected command ['bash', '-l'], got %v", c.command)
		}
	})
}

func TestWorkspaceTerminalCmd_Examples(t *testing.T) {
	t.Parallel()

	// Get the command
	cmd := NewWorkspaceTerminalCmd()

	// Verify Example field is not empty
	if cmd.Example == "" {
		t.Fatal("Example field should not be empty")
	}

	// Parse the examples
	commands, err := testutil.ParseExampleCommands(cmd.Example)
	if err != nil {
		t.Fatalf("Failed to parse examples: %v", err)
	}

	// Verify we have the expected number of examples
	expectedCount := 4
	if len(commands) != expectedCount {
		t.Errorf("Expected %d example commands, got %d", expectedCount, len(commands))
	}

	// Validate all examples against the root command
	rootCmd := NewRootCmd()
	err = testutil.ValidateCommandExamples(rootCmd, cmd.Example)
	if err != nil {
		t.Errorf("Example validation failed: %v", err)
	}
}

func TestWorkspaceTerminalCmd_E2E(t *testing.T) {
	t.Parallel()

	t.Run("fails for nonexistent workspace", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()

		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "terminal", "nonexistent-id", "--storage", storageDir})

		var outBuf bytes.Buffer
		rootCmd.SetOut(&outBuf)
		rootCmd.SetErr(&outBuf)

		err := rootCmd.Execute()
		if err == nil {
			t.Fatal("Expected error for nonexistent workspace")
		}

		output := outBuf.String()
		if !strings.Contains(output, "workspace not found") && !strings.Contains(err.Error(), "workspace not found") {
			t.Errorf("Expected 'workspace not found' error, got: %v (output: %s)", err, output)
		}
	})

	t.Run("fails for stopped workspace", func(t *testing.T) {
		t.Parallel()

		storageDir := t.TempDir()
		sourceDir := t.TempDir()
		configDir := filepath.Join(sourceDir, ".kaiden")
		os.MkdirAll(configDir, 0755)

		// Initialize a workspace
		rootCmd := NewRootCmd()
		rootCmd.SetArgs([]string{"init", sourceDir, "--storage", storageDir, "--runtime", "fake", "--agent", "test-agent"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Failed to init workspace: %v", err)
		}

		// Get the workspace ID
		manager, err := instances.NewManager(storageDir)
		if err != nil {
			t.Fatalf("Failed to create manager: %v", err)
		}
		instancesList, err := manager.List()
		if err != nil {
			t.Fatalf("Failed to list instances: %v", err)
		}
		if len(instancesList) == 0 {
			t.Fatal("No instances found after init")
		}
		workspaceID := instancesList[0].GetID()

		// Try to connect to terminal (workspace is not started)
		rootCmd = NewRootCmd()
		rootCmd.SetArgs([]string{"workspace", "terminal", workspaceID, "--storage", storageDir})

		var outBuf bytes.Buffer
		rootCmd.SetOut(&outBuf)
		rootCmd.SetErr(&outBuf)

		err = rootCmd.Execute()
		if err == nil {
			t.Fatal("Expected error for stopped workspace")
		}

		// Should fail because workspace is not running
		if !strings.Contains(err.Error(), "not running") {
			t.Errorf("Expected 'not running' error, got: %v", err)
		}
	})
}
