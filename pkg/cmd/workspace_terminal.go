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
	"errors"
	"fmt"
	"path/filepath"

	"github.com/kortex-hub/kortex-cli/pkg/instances"
	"github.com/kortex-hub/kortex-cli/pkg/runtimesetup"
	"github.com/spf13/cobra"
)

// workspaceTerminalCmd contains the configuration for the workspace terminal command
type workspaceTerminalCmd struct {
	manager instances.Manager
	id      string
	command []string
}

// preRun validates the parameters and flags
func (w *workspaceTerminalCmd) preRun(cmd *cobra.Command, args []string) error {
	w.id = args[0]

	// Extract command from args[1:] if provided
	if len(args) > 1 {
		w.command = args[1:]
	} else {
		// Default command - will be configurable from runtime in PR #94
		// For now, use the default claude command
		w.command = []string{"claude"}
	}

	// Get storage directory from global flag
	storageDir, err := cmd.Flags().GetString("storage")
	if err != nil {
		return fmt.Errorf("failed to read --storage flag: %w", err)
	}

	// Normalize storage path to absolute path
	absStorageDir, err := filepath.Abs(storageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for storage directory: %w", err)
	}

	// Create manager
	manager, err := instances.NewManager(absStorageDir)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	// Register all available runtimes
	if err := runtimesetup.RegisterAll(manager); err != nil {
		return fmt.Errorf("failed to register runtimes: %w", err)
	}

	w.manager = manager

	return nil
}

// run executes the workspace terminal command logic
func (w *workspaceTerminalCmd) run(cmd *cobra.Command, args []string) error {
	// Start terminal session
	err := w.manager.Terminal(cmd.Context(), w.id, w.command)
	if err != nil {
		if errors.Is(err, instances.ErrInstanceNotFound) {
			return fmt.Errorf("workspace not found: %s\nUse 'workspace list' to see available workspaces", w.id)
		}
		return err
	}

	return nil
}

func NewWorkspaceTerminalCmd() *cobra.Command {
	c := &workspaceTerminalCmd{}

	cmd := &cobra.Command{
		Use:   "terminal ID [COMMAND...]",
		Short: "Connect to a running workspace with an interactive terminal",
		Long: `Connect to a running workspace with an interactive terminal session.

The terminal command starts an interactive session inside a running workspace instance.
By default, it launches the agent command configured in the runtime. You can override
this by providing a custom command.

The workspace must be in a running state. Use 'workspace start' to start a workspace
before connecting.`,
		Example: `# Connect using the default agent command
kortex-cli workspace terminal abc123

# Run a bash shell
kortex-cli workspace terminal abc123 bash

# Run a command with flags (use -- to prevent kortex-cli from parsing them)
kortex-cli workspace terminal abc123 -- bash -c 'echo hello'`,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeRunningWorkspaceID,
		PreRunE:           c.preRun,
		RunE:              c.run,
	}

	return cmd
}
