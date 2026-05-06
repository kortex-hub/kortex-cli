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
	"bytes"
	"strings"
	"testing"

	workspace "github.com/openkaiden/kdn-api/workspace-configuration/go"
	"github.com/openkaiden/kdn/pkg/config"
)

// fakeVertexDetector returns a fixed *VertexConfig.
type fakeVertexDetector struct {
	cfg *VertexConfig
}

func (f *fakeVertexDetector) Detect() (*VertexConfig, error) {
	return f.cfg, nil
}

// detectedVertexCfg is a convenience helper that returns a fully-populated VertexConfig.
func detectedVertexCfg() *VertexConfig {
	return &VertexConfig{
		EnvVars: map[string]string{
			"CLAUDE_CODE_USE_VERTEX":      "1",
			"ANTHROPIC_VERTEX_PROJECT_ID": "my-project",
			"CLOUD_ML_REGION":             "us-east5",
		},
		ADCHostPath: ADCContainerPath,
	}
}

// fakeAgentUpdater records AddEnvVar and AddMount calls.
type fakeAgentUpdater struct {
	envVars []struct{ agentName, name, value string }
	mounts  []struct {
		agentName, host, target string
		ro                      bool
	}
}

func (f *fakeAgentUpdater) AddEnvVar(agentName, name, value string) error {
	f.envVars = append(f.envVars, struct{ agentName, name, value string }{agentName, name, value})
	return nil
}

func (f *fakeAgentUpdater) AddMount(agentName, host, target string, ro bool) error {
	f.mounts = append(f.mounts, struct {
		agentName, host, target string
		ro                      bool
	}{agentName, host, target, ro})
	return nil
}

// fakeAgentLoader returns a fixed *workspace.WorkspaceConfiguration for "claude".
type fakeAgentLoader struct {
	claudeCfg *workspace.WorkspaceConfiguration
}

func (f *fakeAgentLoader) Load(agentName string) (*workspace.WorkspaceConfiguration, error) {
	if agentName == "claude" && f.claudeCfg != nil {
		return f.claudeCfg, nil
	}
	return &workspace.WorkspaceConfiguration{}, nil
}

// fakeVertexWorkspaceConfig returns a fixed *workspace.WorkspaceConfiguration.
type fakeVertexWorkspaceConfig struct {
	cfg *workspace.WorkspaceConfiguration
}

func (f *fakeVertexWorkspaceConfig) Load() (*workspace.WorkspaceConfiguration, error) {
	if f.cfg != nil {
		return f.cfg, nil
	}
	return nil, config.ErrConfigNotFound
}

// alwaysAgent is a selectTarget stub that always picks the agent target.
func alwaysAgent(_ []ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error) {
	return ClaudeVertexConfigTargetAgent, nil
}

func TestClaudeVertexAutoconf_NotDetected(t *testing.T) {
	t.Parallel()

	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: nil},
		AgentUpdater: agentUpdater,
		Confirm:      func(string) (bool, error) { return true, nil },
		SelectTarget: alwaysAgent,
	})

	var buf bytes.Buffer
	if err := runner.Run(&buf); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(agentUpdater.envVars) != 0 || len(agentUpdater.mounts) != 0 {
		t.Error("expected no updates when Vertex AI is not detected")
	}
	if buf.String() != "" {
		t.Errorf("expected no output when not detected, got: %q", buf.String())
	}
}

func TestClaudeVertexAutoconf_AgentTarget_Confirmed(t *testing.T) {
	t.Parallel()

	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: agentUpdater,
		Confirm:      func(string) (bool, error) { return true, nil },
		SelectTarget: alwaysAgent,
	})

	var buf bytes.Buffer
	if err := runner.Run(&buf); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	// Three env vars in fixed order.
	if len(agentUpdater.envVars) != 3 {
		t.Fatalf("expected 3 env var calls, got %d", len(agentUpdater.envVars))
	}
	for _, ev := range agentUpdater.envVars {
		if ev.agentName != "claude" {
			t.Errorf("expected agentName=claude, got %q", ev.agentName)
		}
	}

	// One mount call for the ADC file.
	if len(agentUpdater.mounts) != 1 {
		t.Fatalf("expected 1 mount call, got %d", len(agentUpdater.mounts))
	}
	m := agentUpdater.mounts[0]
	if m.agentName != "claude" {
		t.Errorf("expected agentName=claude, got %q", m.agentName)
	}
	if m.host != ADCContainerPath || m.target != ADCContainerPath {
		t.Errorf("unexpected mount paths: host=%q target=%q", m.host, m.target)
	}
	if !m.ro {
		t.Error("expected mount to be read-only")
	}

	if !strings.Contains(buf.String(), "claude agent config") {
		t.Errorf("expected 'claude agent config' in output, got: %q", buf.String())
	}
}

func TestClaudeVertexAutoconf_LocalTarget(t *testing.T) {
	t.Parallel()

	workspaceUpdater := &fakeWorkspaceUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:         &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater:     &fakeAgentUpdater{},
		WorkspaceUpdater: workspaceUpdater,
		Confirm:          func(string) (bool, error) { return true, nil },
		SelectTarget: func(_ []ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error) {
			return ClaudeVertexConfigTargetLocal, nil
		},
	})

	var buf bytes.Buffer
	if err := runner.Run(&buf); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if len(workspaceUpdater.envVars) != 3 {
		t.Fatalf("expected 3 env var calls, got %d", len(workspaceUpdater.envVars))
	}
	if len(workspaceUpdater.mounts) != 1 {
		t.Fatalf("expected 1 mount call, got %d", len(workspaceUpdater.mounts))
	}
	if !strings.Contains(buf.String(), "local workspace config") {
		t.Errorf("expected 'local workspace config' in output, got: %q", buf.String())
	}
}

func TestClaudeVertexAutoconf_Declined(t *testing.T) {
	t.Parallel()

	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: agentUpdater,
		Confirm:      func(string) (bool, error) { return false, nil },
		SelectTarget: alwaysAgent,
	})

	var buf bytes.Buffer
	if err := runner.Run(&buf); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(agentUpdater.envVars) != 0 || len(agentUpdater.mounts) != 0 {
		t.Error("expected no updates when user declines")
	}
	if !strings.Contains(buf.String(), "Skipped") {
		t.Errorf("expected 'Skipped' in output, got: %q", buf.String())
	}
}

func TestClaudeVertexAutoconf_YesFlag_DefaultsToAgent(t *testing.T) {
	t.Parallel()

	confirmCalled := false
	selectCalled := false
	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: agentUpdater,
		Yes:          true,
		Confirm:      func(string) (bool, error) { confirmCalled = true; return true, nil },
		SelectTarget: func(_ []ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error) {
			selectCalled = true
			return ClaudeVertexConfigTargetAgent, nil
		},
	})

	if err := runner.Run(&bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if confirmCalled {
		t.Error("expected confirm not to be called when yes=true")
	}
	if selectCalled {
		t.Error("expected selectTarget not to be called when yes=true")
	}
	if len(agentUpdater.envVars) != 3 {
		t.Errorf("expected 3 env var calls, got %d", len(agentUpdater.envVars))
	}
}

func TestClaudeVertexAutoconf_AlreadyConfiguredInAgent(t *testing.T) {
	t.Parallel()

	v := "1"
	claudeCfg := &workspace.WorkspaceConfiguration{
		Environment: &[]workspace.EnvironmentVariable{
			{Name: "CLAUDE_CODE_USE_VERTEX", Value: &v},
		},
	}
	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: agentUpdater,
		AgentLoader:  &fakeAgentLoader{claudeCfg: claudeCfg},
		Confirm:      func(string) (bool, error) { return true, nil },
		SelectTarget: alwaysAgent,
	})

	var buf bytes.Buffer
	if err := runner.Run(&buf); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(agentUpdater.envVars) != 0 {
		t.Error("expected no updates when already configured in agent config")
	}
	if !strings.Contains(buf.String(), "already configured") {
		t.Errorf("expected 'already configured' in output, got: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "claude agent") {
		t.Errorf("expected 'claude agent' location in output, got: %q", buf.String())
	}
}

func TestClaudeVertexAutoconf_AlreadyConfiguredInWorkspace(t *testing.T) {
	t.Parallel()

	v := "1"
	wsCfg := &workspace.WorkspaceConfiguration{
		Environment: &[]workspace.EnvironmentVariable{
			{Name: "CLAUDE_CODE_USE_VERTEX", Value: &v},
		},
	}
	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:        &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater:    agentUpdater,
		WorkspaceConfig: &fakeVertexWorkspaceConfig{cfg: wsCfg},
		Confirm:         func(string) (bool, error) { return true, nil },
		SelectTarget:    alwaysAgent,
	})

	var buf bytes.Buffer
	if err := runner.Run(&buf); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(agentUpdater.envVars) != 0 {
		t.Error("expected no updates when already configured in workspace config")
	}
	if !strings.Contains(buf.String(), "local") {
		t.Errorf("expected 'local' location in output, got: %q", buf.String())
	}
}

func TestClaudeVertexAutoconf_LocalOptionNotOfferedWithoutUpdater(t *testing.T) {
	t.Parallel()

	var capturedOptions []ClaudeVertexConfigTargetOption
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: &fakeAgentUpdater{},
		// WorkspaceUpdater intentionally nil.
		Confirm: func(string) (bool, error) { return true, nil },
		SelectTarget: func(opts []ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error) {
			capturedOptions = opts
			return ClaudeVertexConfigTargetAgent, nil
		},
	})

	if err := runner.Run(&bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, opt := range capturedOptions {
		if opt.Target == ClaudeVertexConfigTargetLocal {
			t.Error("local target should not be offered when WorkspaceUpdater is nil")
		}
	}
}

func TestClaudeVertexAutoconf_SkipTarget(t *testing.T) {
	t.Parallel()

	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: agentUpdater,
		Confirm:      func(string) (bool, error) { return true, nil },
		SelectTarget: func(_ []ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error) {
			return 0, ErrSkipped
		},
	})

	var buf bytes.Buffer
	if err := runner.Run(&buf); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(agentUpdater.envVars) != 0 {
		t.Error("expected no updates when target is skipped")
	}
	if !strings.Contains(buf.String(), "Skipped") {
		t.Errorf("expected 'Skipped' in output, got: %q", buf.String())
	}
}

func TestClaudeVertexAutoconf_LocalTarget_NilWorkspaceUpdater_ReturnsError(t *testing.T) {
	t.Parallel()

	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: &fakeAgentUpdater{},
		// WorkspaceUpdater intentionally nil.
		Confirm: func(string) (bool, error) { return true, nil },
		SelectTarget: func(_ []ClaudeVertexConfigTargetOption) (ClaudeVertexConfigTarget, error) {
			return ClaudeVertexConfigTargetLocal, nil
		},
	})

	if err := runner.Run(&bytes.Buffer{}); err == nil {
		t.Error("expected error when local target selected without WorkspaceUpdater")
	}
}

func TestClaudeVertexAutoconf_EnvVarsWrittenInFixedOrder(t *testing.T) {
	t.Parallel()

	agentUpdater := &fakeAgentUpdater{}
	runner := NewClaudeVertexAutoconf(ClaudeVertexAutoconfOptions{
		Detector:     &fakeVertexDetector{cfg: detectedVertexCfg()},
		AgentUpdater: agentUpdater,
		Yes:          true,
		Confirm:      func(string) (bool, error) { return true, nil },
		SelectTarget: alwaysAgent,
	})

	if err := runner.Run(&bytes.Buffer{}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	wantOrder := vertexEnvVars
	if len(agentUpdater.envVars) != len(wantOrder) {
		t.Fatalf("expected %d env var calls, got %d", len(wantOrder), len(agentUpdater.envVars))
	}
	for i, want := range wantOrder {
		if agentUpdater.envVars[i].name != want {
			t.Errorf("env var[%d]: want %q, got %q", i, want, agentUpdater.envVars[i].name)
		}
	}
}
