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

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	workspace "github.com/openkaiden/kdn-api/workspace-configuration/go"
)

const (
	// AgentsConfigFile is the name of the agents configuration file
	AgentsConfigFile = "agents.json"
)

var (
	// ErrInvalidAgentConfig is returned when the agent configuration is invalid
	ErrInvalidAgentConfig = errors.New("invalid agent configuration")
)

// AgentConfigUpdater adds entries to the per-agent configuration file (agents.json).
type AgentConfigUpdater interface {
	// AddEnvVar adds or updates an environment variable in the agent's config.
	// If an entry with the same name already exists its value is replaced.
	AddEnvVar(agentName, name, value string) error

	// AddMount adds a mount entry to the agent's config.
	// The call is idempotent: if a mount with the same host and target already exists
	// it is not duplicated.
	AddMount(agentName, host, target string, ro bool) error
}

// agentConfigUpdater is the unexported implementation.
type agentConfigUpdater struct {
	storageDir string
}

var _ AgentConfigUpdater = (*agentConfigUpdater)(nil)

// NewAgentConfigUpdater returns an AgentConfigUpdater backed by
// <storageDir>/config/agents.json.
func NewAgentConfigUpdater(storageDir string) (AgentConfigUpdater, error) {
	if storageDir == "" {
		return nil, ErrInvalidPath
	}
	absPath, err := filepath.Abs(storageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve storage directory path: %w", err)
	}
	return &agentConfigUpdater{storageDir: absPath}, nil
}

func (a *agentConfigUpdater) AddEnvVar(agentName, name, value string) error {
	configPath := filepath.Join(a.storageDir, "config", AgentsConfigFile)

	agentsConfig, err := a.readAgentsFile(configPath)
	if err != nil {
		return err
	}

	cfg := agentsConfig[agentName]

	if cfg.Environment == nil {
		v := value
		envVars := []workspace.EnvironmentVariable{{Name: name, Value: &v}}
		cfg.Environment = &envVars
	} else {
		for i, e := range *cfg.Environment {
			if e.Name == name {
				v := value
				(*cfg.Environment)[i].Value = &v
				(*cfg.Environment)[i].Secret = nil
				agentsConfig[agentName] = cfg
				return a.writeAgentsFile(configPath, agentsConfig)
			}
		}
		v := value
		*cfg.Environment = append(*cfg.Environment, workspace.EnvironmentVariable{Name: name, Value: &v})
	}

	agentsConfig[agentName] = cfg
	return a.writeAgentsFile(configPath, agentsConfig)
}

func (a *agentConfigUpdater) AddMount(agentName, host, target string, ro bool) error {
	configPath := filepath.Join(a.storageDir, "config", AgentsConfigFile)

	agentsConfig, err := a.readAgentsFile(configPath)
	if err != nil {
		return err
	}

	cfg := agentsConfig[agentName]

	if cfg.Mounts == nil {
		roVal := ro
		mounts := []workspace.Mount{{Host: host, Target: target, Ro: &roVal}}
		cfg.Mounts = &mounts
	} else {
		for _, m := range *cfg.Mounts {
			if m.Host == host && m.Target == target {
				return nil
			}
		}
		roVal := ro
		*cfg.Mounts = append(*cfg.Mounts, workspace.Mount{Host: host, Target: target, Ro: &roVal})
	}

	agentsConfig[agentName] = cfg
	return a.writeAgentsFile(configPath, agentsConfig)
}

func (a *agentConfigUpdater) readAgentsFile(configPath string) (map[string]workspace.WorkspaceConfiguration, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]workspace.WorkspaceConfiguration), nil
		}
		return nil, fmt.Errorf("failed to read agents config: %w", err)
	}

	var agentsConfig map[string]workspace.WorkspaceConfiguration
	if err := json.Unmarshal(data, &agentsConfig); err != nil {
		return nil, fmt.Errorf("failed to parse agents config: %w", err)
	}
	return agentsConfig, nil
}

func (a *agentConfigUpdater) writeAgentsFile(configPath string, agentsConfig map[string]workspace.WorkspaceConfiguration) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(agentsConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agents config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write agents config: %w", err)
	}
	return nil
}

// AgentConfigLoader loads agent-specific workspace configurations
type AgentConfigLoader interface {
	// Load reads and returns the workspace configuration for the specified agent.
	// Returns an empty configuration (not an error) if the agents.json file doesn't exist.
	// Returns an error if the file exists but is invalid JSON or malformed.
	Load(agentName string) (*workspace.WorkspaceConfiguration, error)
}

// agentConfigLoader is the internal implementation
type agentConfigLoader struct {
	storageDir string
}

// Compile-time check to ensure agentConfigLoader implements AgentConfigLoader interface
var _ AgentConfigLoader = (*agentConfigLoader)(nil)

// NewAgentConfigLoader creates a new agent configuration loader
func NewAgentConfigLoader(storageDir string) (AgentConfigLoader, error) {
	if storageDir == "" {
		return nil, ErrInvalidPath
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(storageDir)
	if err != nil {
		return nil, err
	}

	return &agentConfigLoader{
		storageDir: absPath,
	}, nil
}

// Load reads and returns the workspace configuration for the specified agent
func (a *agentConfigLoader) Load(agentName string) (*workspace.WorkspaceConfiguration, error) {
	if agentName == "" {
		return nil, fmt.Errorf("%w: agent name cannot be empty", ErrInvalidAgentConfig)
	}

	configPath := filepath.Join(a.storageDir, "config", AgentsConfigFile)

	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return empty config (not an error)
			return &workspace.WorkspaceConfiguration{}, nil
		}
		return nil, err
	}

	// Parse the JSON
	var agentsConfig map[string]workspace.WorkspaceConfiguration
	if err := json.Unmarshal(data, &agentsConfig); err != nil {
		return nil, fmt.Errorf("%w: failed to parse agents.json: %v", ErrInvalidAgentConfig, err)
	}

	// Get the configuration for the specified agent
	cfg, exists := agentsConfig[agentName]
	if !exists {
		// Agent not found - return empty config (not an error)
		return &workspace.WorkspaceConfiguration{}, nil
	}

	// Validate the configuration
	validator := &config{path: a.storageDir}
	if err := validator.validate(&cfg); err != nil {
		return nil, fmt.Errorf("%w: agent %q configuration validation failed: %v", ErrInvalidAgentConfig, agentName, err)
	}

	return &cfg, nil
}
