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
	"fmt"
	"sort"
	"strings"

	"github.com/openkaiden/kdn/pkg/runtime/podman/config"
	"github.com/openkaiden/kdn/pkg/runtime/podman/constants"
)

// featureInstallInfo holds the information needed to install a devcontainer feature in the image.
type featureInstallInfo struct {
	// dirName is the directory name under instanceDir/features/ (e.g. "feature-0").
	dirName string
	// options contains merged and normalized option env vars for the install.sh invocation.
	options map[string]string
	// envVars contains containerEnv entries to bake into the image after installation.
	envVars map[string]string
}

// buildFeatureInstallCmd builds the RUN instruction body for installing a devcontainer feature.
// Options are passed as inline environment variable assignments before the install.sh invocation.
// Values are always double-quoted to handle spaces and special characters.
func buildFeatureInstallCmd(options map[string]string, installPath string) string {
	scriptPath := installPath + "/install.sh"

	if len(options) == 0 {
		return fmt.Sprintf("chmod +x %s && %s", scriptPath, scriptPath)
	}

	keys := make([]string, 0, len(options))
	for k := range options {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var envParts []string
	for _, k := range keys {
		// Escape embedded double quotes in the value.
		v := strings.ReplaceAll(options[k], `"`, `\"`)
		envParts = append(envParts, fmt.Sprintf(`%s="%s"`, k, v))
	}

	return fmt.Sprintf("chmod +x %s && %s %s", scriptPath, strings.Join(envParts, " "), scriptPath)
}

// generateSudoers generates the sudoers file content from a list of allowed binaries.
// It creates a single ALLOWED command alias and sets up sudo rules.
func generateSudoers(sudoBinaries []string) string {
	if len(sudoBinaries) == 0 {
		// No sudo access if no binaries are specified
		return fmt.Sprintf("%s ALL = !ALL\n", constants.ContainerUser)
	}

	var lines []string

	// Create single ALLOWED command alias
	lines = append(lines, fmt.Sprintf("Cmnd_Alias ALLOWED = %s", strings.Join(sudoBinaries, ", ")))
	lines = append(lines, "")

	// Create sudo rule
	lines = append(lines, fmt.Sprintf("%s ALL = !ALL, NOPASSWD: ALLOWED", constants.ContainerUser))

	return strings.Join(lines, "\n") + "\n"
}

// generateContainerfile generates the Containerfile content from image and agent configurations.
// If hasAgentSettings is true, a COPY instruction is added to embed the agent-settings
// directory (written to the build context) into the agent user's home directory.
// If featureInfos is non-empty, features are installed as root after user creation so
// that install scripts can chown files, write to /home/agent, and su to the target user.
// _REMOTE_USER and _REMOTE_USER_HOME are exported before the feature block so install
// scripts can resolve the target user by name. USER agent:agent is set after all features.
func generateContainerfile(imageConfig *config.ImageConfig, agentConfig *config.AgentConfig, hasAgentSettings bool, featureInfos []featureInstallInfo) string {
	if imageConfig == nil {
		return ""
	}
	if agentConfig == nil {
		return ""
	}

	var lines []string

	// FROM line with base image
	baseImage := fmt.Sprintf("%s:%s", constants.BaseImageRegistry, imageConfig.Version)
	lines = append(lines, fmt.Sprintf("FROM %s", baseImage))
	lines = append(lines, "")

	// Build args for UID/GID declared early so they are available for user creation
	// and remain in scope for the rest of the stage.
	lines = append(lines, "ARG UID=1000")
	lines = append(lines, "ARG GID=1000")
	lines = append(lines, "")

	// Merge packages from image and agent configs
	allPackages := append([]string{}, imageConfig.Packages...)
	allPackages = append(allPackages, agentConfig.Packages...)

	// Install packages if any
	if len(allPackages) > 0 {
		lines = append(lines, fmt.Sprintf("RUN dnf install -y %s", strings.Join(allPackages, " ")))
		lines = append(lines, "")
	}

	// User and group setup — done before feature installation so the agent user and
	// /home/agent exist when feature install scripts run.
	lines = append(lines, `RUN GROUPNAME=$(grep $GID /etc/group | cut -d: -f1); [ -n "$GROUPNAME" ] && groupdel $GROUPNAME || true`)
	lines = append(lines, fmt.Sprintf(`RUN groupadd -g "${GID}" %s && useradd -u "${UID}" -g "${GID}" -m %s`, constants.ContainerGroup, constants.ContainerUser))
	lines = append(lines, fmt.Sprintf("COPY sudoers /etc/sudoers.d/%s", constants.ContainerUser))
	lines = append(lines, fmt.Sprintf("RUN chmod 0440 /etc/sudoers.d/%s", constants.ContainerUser))
	lines = append(lines, "")

	// Devcontainer feature installation section.
	// Features still run as root (USER switch comes after) so they can install
	// system-wide tools. The agent user and /home/agent now exist so install scripts
	// can chown files, write to the home directory, and su to the target user.
	// _REMOTE_USER/_REMOTE_USER_HOME tell scripts which user to configure.
	// containerEnv vars from each feature are set immediately after installation so
	// subsequent features can reference them during their own install.
	if len(featureInfos) > 0 {
		lines = append(lines, fmt.Sprintf(`ENV _REMOTE_USER="%s"`, constants.ContainerUser))
		lines = append(lines, fmt.Sprintf(`ENV _REMOTE_USER_HOME="/home/%s"`, constants.ContainerUser))
		lines = append(lines, "")
		for _, f := range featureInfos {
			installPath := fmt.Sprintf("/tmp/feature-install/%s", f.dirName)
			lines = append(lines, fmt.Sprintf("COPY features/%s/ %s/", f.dirName, installPath))
			lines = append(lines, fmt.Sprintf("RUN %s", buildFeatureInstallCmd(f.options, installPath)))
			// Set containerEnv vars sorted for deterministic output.
			if len(f.envVars) > 0 {
				envKeys := make([]string, 0, len(f.envVars))
				for k := range f.envVars {
					envKeys = append(envKeys, k)
				}
				sort.Strings(envKeys)
				for _, k := range envKeys {
					v := strings.ReplaceAll(f.envVars[k], `"`, `\"`)
					lines = append(lines, fmt.Sprintf(`ENV %s="%s"`, k, v))
				}
			}
		}
		lines = append(lines, "")
	}

	// Switch to the agent user for all subsequent steps.
	lines = append(lines, fmt.Sprintf("USER %s:%s", constants.ContainerUser, constants.ContainerGroup))
	lines = append(lines, "")

	// Environment PATH — prepend agent-local and system bins while preserving any
	// PATH additions made by devcontainer features installed above.
	lines = append(lines, fmt.Sprintf("ENV PATH=/home/%s/.local/bin:/usr/local/bin:/usr/bin:$PATH", constants.ContainerUser))
	lines = append(lines, "")

	// Copy Containerfile to home directory for reference
	lines = append(lines, fmt.Sprintf("COPY Containerfile /home/%s/Containerfile", constants.ContainerUser))

	// Copy agent default settings into home directory before the RUN commands,
	// so that agent install scripts can read and build upon the defaults.
	if hasAgentSettings {
		lines = append(lines, fmt.Sprintf("COPY --chown=%s:%s agent-settings/. /home/%s/",
			constants.ContainerUser, constants.ContainerGroup, constants.ContainerUser))
	}

	// Custom RUN commands from image config
	for _, cmd := range imageConfig.RunCommands {
		lines = append(lines, fmt.Sprintf("RUN %s", cmd))
	}

	// Custom RUN commands from agent config
	for _, cmd := range agentConfig.RunCommands {
		lines = append(lines, fmt.Sprintf("RUN %s", cmd))
	}

	// Add final newline if there are RUN commands
	if len(imageConfig.RunCommands) > 0 || len(agentConfig.RunCommands) > 0 {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}
