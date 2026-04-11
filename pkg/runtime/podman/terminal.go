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
	"fmt"

	"github.com/openkaiden/kdn/pkg/runtime"
)

// Ensure podmanRuntime implements runtime.Terminal at compile time.
var _ runtime.Terminal = (*podmanRuntime)(nil)

// Terminal starts an interactive terminal session inside the workspace container of a running pod.
func (p *podmanRuntime) Terminal(ctx context.Context, instanceID string, agent string, command []string) error {
	if instanceID == "" {
		return fmt.Errorf("%w: instance ID is required", runtime.ErrInvalidParams)
	}

	// If no command provided, retrieve the terminal command from agent config
	if len(command) == 0 {
		if agent == "" {
			return fmt.Errorf("%w: agent name is required when command is not provided", runtime.ErrInvalidParams)
		}

		// Load agent config to get terminal command
		agentConfig, err := p.config.LoadAgent(agent)
		if err != nil {
			return fmt.Errorf("failed to load agent config: %w", err)
		}

		if len(agentConfig.TerminalCommand) == 0 {
			return fmt.Errorf("%w: agent %q has no terminal command configured", runtime.ErrInvalidParams, agent)
		}

		// Use the terminal command from agent config
		command = agentConfig.TerminalCommand
	}

	// The instanceID is the pod name; exec targets the workspace container.
	wsContainer := workspaceContainerName(instanceID)

	// Build podman exec -it <workspace-container> <command...>
	args := []string{"exec", "-it", wsContainer}
	args = append(args, command...)

	return p.executor.RunInteractive(ctx, args...)
}
