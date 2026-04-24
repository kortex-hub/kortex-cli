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
	"strings"
	"testing"

	"github.com/openkaiden/kdn/pkg/runtime/podman/config"
)

func TestGenerateSudoers(t *testing.T) {
	t.Parallel()

	t.Run("generates sudoers with single ALLOWED alias", func(t *testing.T) {
		t.Parallel()

		sudoBinaries := []string{"/usr/bin/dnf", "/bin/kill", "/usr/bin/killall"}
		result := generateSudoers(sudoBinaries)

		// Check for ALLOWED alias
		if !strings.Contains(result, "Cmnd_Alias ALLOWED =") {
			t.Error("Expected sudoers to contain 'Cmnd_Alias ALLOWED ='")
		}

		// Check that all binaries are listed
		for _, binary := range sudoBinaries {
			if !strings.Contains(result, binary) {
				t.Errorf("Expected sudoers to contain %s", binary)
			}
		}

		// Check for the sudo rule
		if !strings.Contains(result, "agent ALL = !ALL, NOPASSWD: ALLOWED") {
			t.Error("Expected sudoers to contain correct sudo rule")
		}
	})

	t.Run("generates no-access sudoers when no binaries provided", func(t *testing.T) {
		t.Parallel()

		result := generateSudoers([]string{})

		// Should only have the deny-all rule
		if !strings.Contains(result, "agent ALL = !ALL") {
			t.Error("Expected sudoers to contain 'agent ALL = !ALL'")
		}

		// Should not have ALLOWED alias
		if strings.Contains(result, "ALLOWED") {
			t.Error("Expected sudoers to not contain ALLOWED alias when no binaries provided")
		}
	})

	t.Run("joins multiple binaries with comma separator", func(t *testing.T) {
		t.Parallel()

		sudoBinaries := []string{"/usr/bin/dnf", "/bin/kill"}
		result := generateSudoers(sudoBinaries)

		// Check for comma-separated list
		if !strings.Contains(result, "/usr/bin/dnf, /bin/kill") {
			t.Error("Expected binaries to be comma-separated")
		}
	})
}

func TestGenerateContainerfile(t *testing.T) {
	t.Parallel()

	t.Run("generates containerfile with default configs", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:     "latest",
			Packages:    []string{"which", "procps-ng"},
			Sudo:        []string{"/usr/bin/dnf"},
			RunCommands: []string{},
		}

		agentConfig := &config.AgentConfig{
			Packages:        []string{},
			RunCommands:     []string{"curl -fsSL https://claude.ai/install.sh | bash"},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		// Check for FROM line with correct base image
		expectedFrom := "FROM registry.fedoraproject.org/fedora:latest"
		if !strings.Contains(result, expectedFrom) {
			t.Errorf("Expected FROM line: %s", expectedFrom)
		}

		// Check for package installation
		if !strings.Contains(result, "RUN dnf install -y which procps-ng") {
			t.Error("Expected package installation line")
		}

		// Check for user/group setup
		if !strings.Contains(result, "ARG UID=1000") {
			t.Error("Expected UID argument")
		}
		if !strings.Contains(result, "ARG GID=1000") {
			t.Error("Expected GID argument")
		}
		if !strings.Contains(result, "USER agent:agent") {
			t.Error("Expected USER line")
		}

		// Check for sudoers copy
		if !strings.Contains(result, "COPY sudoers /etc/sudoers.d/agent") {
			t.Error("Expected COPY sudoers line")
		}

		// Check for sudoers chmod
		if !strings.Contains(result, "RUN chmod 0440 /etc/sudoers.d/agent") {
			t.Error("Expected RUN chmod for sudoers")
		}

		// Check for PATH environment — must preserve $PATH so feature additions are not lost
		if !strings.Contains(result, "ENV PATH=/home/agent/.local/bin:/usr/local/bin:/usr/bin:$PATH") {
			t.Error("Expected PATH environment variable preserving $PATH")
		}

		// Check for Containerfile copy
		if !strings.Contains(result, "COPY Containerfile /home/agent/Containerfile") {
			t.Error("Expected COPY Containerfile line")
		}

		// Check for agent RUN commands
		if !strings.Contains(result, "RUN curl -fsSL https://claude.ai/install.sh | bash") {
			t.Error("Expected agent RUN command")
		}
	})

	t.Run("uses custom fedora version", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:  "40",
			Packages: []string{},
			Sudo:     []string{},
		}

		agentConfig := &config.AgentConfig{
			Packages:        []string{},
			RunCommands:     []string{},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		expectedFrom := "FROM registry.fedoraproject.org/fedora:40"
		if !strings.Contains(result, expectedFrom) {
			t.Errorf("Expected FROM line with custom version: %s", expectedFrom)
		}
	})

	t.Run("merges packages from image and agent configs", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:  "latest",
			Packages: []string{"package1", "package2"},
			Sudo:     []string{},
		}

		agentConfig := &config.AgentConfig{
			Packages:        []string{"package3", "package4"},
			RunCommands:     []string{},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		// Should have all packages in a single RUN command
		if !strings.Contains(result, "RUN dnf install -y package1 package2 package3 package4") {
			t.Error("Expected merged package installation with all packages")
		}
	})

	t.Run("omits package installation when no packages", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:  "latest",
			Packages: []string{},
			Sudo:     []string{},
		}

		agentConfig := &config.AgentConfig{
			Packages:        []string{},
			RunCommands:     []string{},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		// Should not have dnf install line
		if strings.Contains(result, "dnf install") {
			t.Error("Expected no dnf install line when no packages specified")
		}
	})

	t.Run("includes custom RUN commands from both configs", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:     "latest",
			Packages:    []string{},
			Sudo:        []string{},
			RunCommands: []string{"echo 'image setup'"},
		}

		agentConfig := &config.AgentConfig{
			Packages:        []string{},
			RunCommands:     []string{"echo 'agent setup'"},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		// Should have both RUN commands
		if !strings.Contains(result, "RUN echo 'image setup'") {
			t.Error("Expected image RUN command")
		}
		if !strings.Contains(result, "RUN echo 'agent setup'") {
			t.Error("Expected agent RUN command")
		}
	})

	t.Run("adds COPY instruction for agent settings when hasAgentSettings is true", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:     "latest",
			Packages:    []string{},
			Sudo:        []string{},
			RunCommands: []string{"echo 'image setup'"},
		}
		agentConfig := &config.AgentConfig{
			Packages:        []string{},
			RunCommands:     []string{"echo 'agent setup'"},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, true, nil)

		expected := "COPY --chown=agent:agent agent-settings/. /home/agent/"
		if !strings.Contains(result, expected) {
			t.Errorf("Expected Containerfile to contain %q, got:\n%s", expected, result)
		}

		// Verify the COPY comes before all RUN commands so agent install scripts can
		// read and build upon the defaults.
		settingsPos := strings.Index(result, expected)
		imageRunPos := strings.Index(result, "RUN echo 'image setup'")
		agentRunPos := strings.Index(result, "RUN echo 'agent setup'")

		if settingsPos > imageRunPos || settingsPos > agentRunPos {
			t.Error("Expected COPY agent-settings to appear before all RUN commands")
		}
	})

	t.Run("no agent-settings COPY when hasAgentSettings is false", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:     "latest",
			Packages:    []string{},
			Sudo:        []string{},
			RunCommands: []string{},
		}
		agentConfig := &config.AgentConfig{
			Packages:        []string{},
			RunCommands:     []string{},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		if strings.Contains(result, "agent-settings") {
			t.Errorf("Expected no agent-settings COPY line, got:\n%s", result)
		}
	})

	t.Run("image RUN commands come before agent RUN commands", func(t *testing.T) {
		t.Parallel()

		imageConfig := &config.ImageConfig{
			Version:     "latest",
			Packages:    []string{},
			Sudo:        []string{},
			RunCommands: []string{"echo 'image'"},
		}

		agentConfig := &config.AgentConfig{
			Packages:        []string{},
			RunCommands:     []string{"echo 'agent'"},
			TerminalCommand: []string{"claude"},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		// Find positions
		imagePos := strings.Index(result, "RUN echo 'image'")
		agentPos := strings.Index(result, "RUN echo 'agent'")

		if imagePos == -1 || agentPos == -1 {
			t.Fatal("Both RUN commands should be present")
		}

		if imagePos > agentPos {
			t.Error("Image RUN commands should come before agent RUN commands")
		}
	})
}

func TestBuildFeatureInstallCmd(t *testing.T) {
	t.Parallel()

	t.Run("no options produces simple chmod and run", func(t *testing.T) {
		t.Parallel()

		result := buildFeatureInstallCmd(nil, "/tmp/feature-install/feature-0")
		expected := "chmod +x /tmp/feature-install/feature-0/install.sh && /tmp/feature-install/feature-0/install.sh"
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("single option is inlined before script", func(t *testing.T) {
		t.Parallel()

		result := buildFeatureInstallCmd(map[string]string{"VERSION": "1.0"}, "/tmp/feature-install/feature-0")
		expected := `chmod +x /tmp/feature-install/feature-0/install.sh && VERSION="1.0" /tmp/feature-install/feature-0/install.sh`
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("multiple options are sorted alphabetically", func(t *testing.T) {
		t.Parallel()

		result := buildFeatureInstallCmd(
			map[string]string{"VERSION": "1.0", "INSTALL": "true"},
			"/tmp/feature-install/feature-0",
		)
		expected := `chmod +x /tmp/feature-install/feature-0/install.sh && INSTALL="true" VERSION="1.0" /tmp/feature-install/feature-0/install.sh`
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})

	t.Run("double quotes in value are escaped", func(t *testing.T) {
		t.Parallel()

		result := buildFeatureInstallCmd(map[string]string{"KEY": `val"ue`}, "/tmp/feature-install/feature-0")
		expected := `chmod +x /tmp/feature-install/feature-0/install.sh && KEY="val\"ue" /tmp/feature-install/feature-0/install.sh`
		if result != expected {
			t.Errorf("expected %q, got %q", expected, result)
		}
	})
}

func TestGenerateContainerfile_WithFeatures(t *testing.T) {
	t.Parallel()

	imageConfig := &config.ImageConfig{
		Version:     "latest",
		Packages:    []string{},
		Sudo:        []string{},
		RunCommands: []string{"echo 'image setup'"},
	}
	agentConfig := &config.AgentConfig{
		Packages:        []string{},
		RunCommands:     []string{"echo 'agent setup'"},
		TerminalCommand: []string{"claude"},
	}

	t.Run("single feature without options or containerEnv", func(t *testing.T) {
		t.Parallel()

		infos := []featureInstallInfo{
			{dirName: "feature-0", options: nil, envVars: nil},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, infos)

		if !strings.Contains(result, "COPY features/feature-0/ /tmp/feature-install/feature-0/") {
			t.Error("Expected COPY instruction for feature-0")
		}
		if !strings.Contains(result, "RUN chmod +x /tmp/feature-install/feature-0/install.sh && /tmp/feature-install/feature-0/install.sh") {
			t.Error("Expected RUN install.sh without options")
		}

		// No USER root switch — features still run as root but after user creation.
		if strings.Contains(result, "USER root") {
			t.Error("Expected no 'USER root' instruction")
		}

		// Feature COPY appears before USER agent:agent (features run as root).
		featureCopyPos := strings.Index(result, "COPY features/feature-0/")
		userAgentPos := strings.Index(result, "USER agent:agent")
		if featureCopyPos == -1 || userAgentPos == -1 {
			t.Fatal("Expected both feature COPY and 'USER agent:agent'")
		}
		if featureCopyPos > userAgentPos {
			t.Error("Expected feature COPY to appear before 'USER agent:agent'")
		}

		// User must be created before features so install scripts can reference the account.
		useradd := "useradd"
		useraddPos := strings.Index(result, useradd)
		if useraddPos == -1 {
			t.Fatal("Expected useradd instruction")
		}
		if useraddPos > featureCopyPos {
			t.Error("Expected useradd to appear before feature COPY")
		}
	})

	t.Run("feature with options includes inline env assignments", func(t *testing.T) {
		t.Parallel()

		infos := []featureInstallInfo{
			{dirName: "feature-0", options: map[string]string{"VERSION": "1.21", "INSTALL": "true"}, envVars: nil},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, infos)

		expected := `RUN chmod +x /tmp/feature-install/feature-0/install.sh && INSTALL="true" VERSION="1.21" /tmp/feature-install/feature-0/install.sh`
		if !strings.Contains(result, expected) {
			t.Errorf("Expected RUN with sorted options\nWant: %q\nGot:\n%s", expected, result)
		}
	})

	t.Run("feature with containerEnv sets ENV instructions", func(t *testing.T) {
		t.Parallel()

		infos := []featureInstallInfo{
			{
				dirName: "feature-0",
				options: nil,
				envVars: map[string]string{"GOPATH": "/home/agent/go", "GOROOT": "/usr/local/go"},
			},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, infos)

		if !strings.Contains(result, `ENV GOPATH="/home/agent/go"`) {
			t.Errorf("Expected ENV GOPATH instruction\nGot:\n%s", result)
		}
		if !strings.Contains(result, `ENV GOROOT="/usr/local/go"`) {
			t.Errorf("Expected ENV GOROOT instruction\nGot:\n%s", result)
		}
	})

	t.Run("feature section appears before custom RUN commands", func(t *testing.T) {
		t.Parallel()

		infos := []featureInstallInfo{
			{dirName: "feature-0", options: nil, envVars: nil},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, infos)

		featureCopyPos := strings.Index(result, "COPY features/feature-0/")
		imageRunPos := strings.Index(result, "RUN echo 'image setup'")
		agentRunPos := strings.Index(result, "RUN echo 'agent setup'")

		if featureCopyPos == -1 {
			t.Fatal("Expected feature COPY instruction")
		}
		if featureCopyPos > imageRunPos || featureCopyPos > agentRunPos {
			t.Error("Expected feature COPY to appear before image/agent RUN commands")
		}
	})

	t.Run("multiple features installed in declaration order with containerEnv between them", func(t *testing.T) {
		t.Parallel()

		infos := []featureInstallInfo{
			{dirName: "feature-0", options: nil, envVars: map[string]string{"FIRST_VAR": "first"}},
			{dirName: "feature-1", options: nil, envVars: nil},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, infos)

		feature0Pos := strings.Index(result, "COPY features/feature-0/")
		feature1Pos := strings.Index(result, "COPY features/feature-1/")
		envPos := strings.Index(result, `ENV FIRST_VAR="first"`)

		if feature0Pos == -1 || feature1Pos == -1 {
			t.Fatal("Expected both feature COPY instructions")
		}
		if feature0Pos > feature1Pos {
			t.Error("Expected feature-0 to be installed before feature-1")
		}
		// containerEnv from feature-0 should appear before feature-1 install
		if envPos == -1 || envPos > feature1Pos {
			t.Error("Expected containerEnv from feature-0 to appear before feature-1 COPY")
		}
	})

	t.Run("no feature instructions when featureInfos is nil", func(t *testing.T) {
		t.Parallel()

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		if strings.Contains(result, "COPY features/") {
			t.Errorf("Expected no feature COPY instructions\nGot:\n%s", result)
		}
		if strings.Contains(result, "/tmp/feature-install/") {
			t.Errorf("Expected no feature install paths\nGot:\n%s", result)
		}
	})

	t.Run("_REMOTE_USER and _REMOTE_USER_HOME are set before feature installation", func(t *testing.T) {
		t.Parallel()

		infos := []featureInstallInfo{
			{dirName: "feature-0", options: nil, envVars: nil},
		}

		result := generateContainerfile(imageConfig, agentConfig, false, infos)

		if !strings.Contains(result, `ENV _REMOTE_USER="agent"`) {
			t.Errorf("Expected ENV _REMOTE_USER=\"agent\"\nGot:\n%s", result)
		}
		if !strings.Contains(result, `ENV _REMOTE_USER_HOME="/home/agent"`) {
			t.Errorf("Expected ENV _REMOTE_USER_HOME=\"/home/agent\"\nGot:\n%s", result)
		}

		// Both vars must appear before the feature COPY instruction.
		remoteUserPos := strings.Index(result, `ENV _REMOTE_USER="agent"`)
		remoteUserHomePos := strings.Index(result, `ENV _REMOTE_USER_HOME="/home/agent"`)
		featureCopyPos := strings.Index(result, "COPY features/feature-0/")

		if remoteUserPos == -1 || remoteUserHomePos == -1 || featureCopyPos == -1 {
			t.Fatal("Expected _REMOTE_USER, _REMOTE_USER_HOME, and feature COPY to be present")
		}
		if remoteUserPos > featureCopyPos {
			t.Error("Expected _REMOTE_USER to appear before feature COPY")
		}
		if remoteUserHomePos > featureCopyPos {
			t.Error("Expected _REMOTE_USER_HOME to appear before feature COPY")
		}
	})

	t.Run("_REMOTE_USER and _REMOTE_USER_HOME are not set when no features", func(t *testing.T) {
		t.Parallel()

		result := generateContainerfile(imageConfig, agentConfig, false, nil)

		if strings.Contains(result, "_REMOTE_USER") {
			t.Errorf("Expected no _REMOTE_USER when no features\nGot:\n%s", result)
		}
	})

	t.Run("feature section appears before agent-settings COPY", func(t *testing.T) {
		t.Parallel()

		infos := []featureInstallInfo{
			{dirName: "feature-0", options: nil, envVars: nil},
		}

		result := generateContainerfile(imageConfig, agentConfig, true, infos)

		featureCopyPos := strings.Index(result, "COPY features/feature-0/")
		agentSettingsPos := strings.Index(result, "COPY --chown=agent:agent agent-settings/.")

		if featureCopyPos == -1 || agentSettingsPos == -1 {
			t.Fatal("Expected both feature COPY and agent-settings COPY")
		}
		if featureCopyPos > agentSettingsPos {
			t.Error("Expected feature COPY to appear before agent-settings COPY")
		}
	})
}
