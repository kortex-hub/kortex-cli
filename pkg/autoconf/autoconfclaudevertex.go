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

package autoconf

import (
	"errors"
	"fmt"
	"io"
	"strings"

	workspace "github.com/openkaiden/kdn-api/workspace-configuration/go"
	"github.com/openkaiden/kdn/pkg/config"
)

// ClaudeVertexConfigTarget identifies where detected Vertex AI configuration
// is recorded for Claude workspaces.
type ClaudeVertexConfigTarget int

const (
	// ClaudeVertexConfigTargetAgent records env vars and the ADC mount in
	// agents.json under the "claude" key (applies to all Claude workspaces).
	ClaudeVertexConfigTargetAgent ClaudeVertexConfigTarget = iota
	// ClaudeVertexConfigTargetLocal records env vars and the ADC mount in
	// the local .kaiden/workspace.json.
	ClaudeVertexConfigTargetLocal
)

// ClaudeVertexConfigTargetOption pairs a ClaudeVertexConfigTarget with a
// human-readable label for display in selection prompts.
type ClaudeVertexConfigTargetOption struct {
	Target ClaudeVertexConfigTarget
	Label  string
}

// ClaudeVertexAutoconfOptions configures a ClaudeVertexAutoconf runner.
type ClaudeVertexAutoconfOptions struct {
	Detector VertexDetector

	// AgentUpdater writes to ~/.kdn/config/agents.json for the "claude" agent.
	AgentUpdater config.AgentConfigUpdater
	// WorkspaceUpdater writes to .kaiden/workspace.json in the current directory.
	// When nil the local target is not offered.
	WorkspaceUpdater config.WorkspaceConfigUpdater

	// AgentLoader is used to check whether Vertex AI is already configured in
	// the claude agent config. May be nil (skips that check).
	AgentLoader config.AgentConfigLoader
	// WorkspaceConfig is used to check whether Vertex AI is already configured
	// in the local workspace config. May be nil (skips that check).
	WorkspaceConfig config.Config

	Yes bool

	// Confirm is called to ask the user whether to proceed.
	Confirm func(prompt string) (bool, error)

	// SelectTarget is called to ask the user where to record the configuration.
	// It may return ErrSkipped to skip without applying.
	SelectTarget func(options []ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error)
}

// ClaudeVertexAutoconf orchestrates Vertex AI configuration detection and
// application for Claude workspaces. It adds env vars and an ADC file mount
// to either the claude agent config (all Claude workspaces) or the local
// workspace config.
type ClaudeVertexAutoconf interface {
	Run(out io.Writer) error
}

type claudeVertexAutoconfRunner struct {
	detector         VertexDetector
	agentUpdater     config.AgentConfigUpdater
	workspaceUpdater config.WorkspaceConfigUpdater
	agentLoader      config.AgentConfigLoader
	workspaceConfig  config.Config
	yes              bool
	confirm          func(string) (bool, error)
	selectTarget     func([]ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error)
}

var _ ClaudeVertexAutoconf = (*claudeVertexAutoconfRunner)(nil)

// NewClaudeVertexAutoconf returns a ClaudeVertexAutoconf configured by opts.
func NewClaudeVertexAutoconf(opts ClaudeVertexAutoconfOptions) ClaudeVertexAutoconf {
	return &claudeVertexAutoconfRunner{
		detector:         opts.Detector,
		agentUpdater:     opts.AgentUpdater,
		workspaceUpdater: opts.WorkspaceUpdater,
		agentLoader:      opts.AgentLoader,
		workspaceConfig:  opts.WorkspaceConfig,
		yes:              opts.Yes,
		confirm:          opts.Confirm,
		selectTarget:     opts.SelectTarget,
	}
}

func (r *claudeVertexAutoconfRunner) Run(out io.Writer) error {
	cfg, err := r.detector.Detect()
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil
	}

	// Check if already configured in any target.
	locations := r.findExistingLocations()
	if len(locations) > 0 {
		fmt.Fprintf(out, "%s Vertex AI already configured for Claude (%s).\n", greenCheck, formatVertexLocations(locations))
		return nil
	}

	fmt.Fprintf(out, "Detected Vertex AI configuration (CLAUDE_CODE_USE_VERTEX + ADC file)\n")

	if !r.yes {
		ok, err := r.confirm("Configure Claude workspaces for Vertex AI (env vars + ADC file mount)?")
		if err != nil {
			return fmt.Errorf("confirmation failed: %w", err)
		}
		if !ok {
			fmt.Fprintf(out, "%s Skipped Vertex AI configuration.\n", greyDash)
			return nil
		}
	}

	target := ClaudeVertexConfigTargetAgent // default for --yes
	if !r.yes {
		options := r.buildTargetOptions()
		var selErr error
		target, selErr = r.selectTarget(options)
		if errors.Is(selErr, ErrSkipped) {
			fmt.Fprintf(out, "%s Skipped Vertex AI configuration.\n", greyDash)
			return nil
		}
		if selErr != nil {
			return fmt.Errorf("target selection failed: %w", selErr)
		}
	}

	return r.applyTarget(out, cfg, target)
}

func (r *claudeVertexAutoconfRunner) buildTargetOptions() []ClaudeVertexConfigTargetOption {
	opts := []ClaudeVertexConfigTargetOption{
		{Target: ClaudeVertexConfigTargetAgent, Label: "Claude agent config (all Claude workspaces)"},
	}
	if r.workspaceUpdater != nil {
		opts = append(opts, ClaudeVertexConfigTargetOption{
			Target: ClaudeVertexConfigTargetLocal,
			Label:  "Local (.kaiden/workspace.json)",
		})
	}
	return opts
}

func (r *claudeVertexAutoconfRunner) applyTarget(out io.Writer, cfg *VertexConfig, target ClaudeVertexConfigTarget) error {
	switch target {
	case ClaudeVertexConfigTargetAgent:
		for _, name := range vertexEnvVars {
			value, ok := cfg.EnvVars[name]
			if !ok {
				continue
			}
			if err := r.agentUpdater.AddEnvVar("claude", name, value); err != nil {
				return fmt.Errorf("failed to add env var %q to claude agent config: %w", name, err)
			}
		}
		if err := r.agentUpdater.AddMount("claude", cfg.ADCHostPath, ADCContainerPath, true); err != nil {
			return fmt.Errorf("failed to add ADC mount to claude agent config: %w", err)
		}
		fmt.Fprintf(out, "%s Configured Vertex AI in claude agent config.\n", greenCheck)

	case ClaudeVertexConfigTargetLocal:
		if r.workspaceUpdater == nil {
			return fmt.Errorf("local config target selected but workspace updater is not configured")
		}
		for _, name := range vertexEnvVars {
			value, ok := cfg.EnvVars[name]
			if !ok {
				continue
			}
			if err := r.workspaceUpdater.AddEnvVar(name, value); err != nil {
				return fmt.Errorf("failed to add env var %q to workspace config: %w", name, err)
			}
		}
		if err := r.workspaceUpdater.AddMount(cfg.ADCHostPath, ADCContainerPath, true); err != nil {
			return fmt.Errorf("failed to add ADC mount to workspace config: %w", err)
		}
		fmt.Fprintf(out, "%s Configured Vertex AI in local workspace config.\n", greenCheck)

	default:
		return fmt.Errorf("unknown vertex config target %d", target)
	}
	return nil
}

// findExistingLocations returns the config targets where CLAUDE_CODE_USE_VERTEX
// is already present as an environment variable.
func (r *claudeVertexAutoconfRunner) findExistingLocations() []ClaudeVertexConfigTarget {
	var locations []ClaudeVertexConfigTarget

	if r.agentLoader != nil {
		if cfg, err := r.agentLoader.Load("claude"); err == nil && hasVertexEnvVar(cfg) {
			locations = append(locations, ClaudeVertexConfigTargetAgent)
		}
	}

	if r.workspaceConfig != nil {
		if cfg, err := r.workspaceConfig.Load(); err == nil && hasVertexEnvVar(cfg) {
			locations = append(locations, ClaudeVertexConfigTargetLocal)
		}
	}

	return locations
}

// hasVertexEnvVar returns true if the config contains CLAUDE_CODE_USE_VERTEX
// in its environment list.
func hasVertexEnvVar(cfg *workspace.WorkspaceConfiguration) bool {
	if cfg == nil || cfg.Environment == nil {
		return false
	}
	for _, e := range *cfg.Environment {
		if e.Name == "CLAUDE_CODE_USE_VERTEX" {
			return true
		}
	}
	return false
}

func formatVertexLocations(locs []ClaudeVertexConfigTarget) string {
	names := make([]string, 0, len(locs))
	for _, l := range locs {
		switch l {
		case ClaudeVertexConfigTargetAgent:
			names = append(names, "claude agent")
		case ClaudeVertexConfigTargetLocal:
			names = append(names, "local")
		}
	}
	return strings.Join(names, ", ")
}
