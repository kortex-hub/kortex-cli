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
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	api "github.com/openkaiden/kdn-api/cli/go"
	"github.com/openkaiden/kdn/pkg/logger"
	"github.com/openkaiden/kdn/pkg/runtime"
	"github.com/openkaiden/kdn/pkg/runtime/podman/config"
	"github.com/openkaiden/kdn/pkg/runtime/podman/pods"
	"github.com/openkaiden/kdn/pkg/steplogger"
)

const defaultOnecliVersion = "1.17"

// podTemplateData holds the values used to render the pod YAML template.
type podTemplateData struct {
	Name          string
	OnecliWebPort int
	OnecliVersion string
}

// validateCreateParams validates the create parameters.
func (p *podmanRuntime) validateCreateParams(params runtime.CreateParams) error {
	if params.Name == "" {
		return fmt.Errorf("%w: name is required", runtime.ErrInvalidParams)
	}
	if params.SourcePath == "" {
		return fmt.Errorf("%w: source path is required", runtime.ErrInvalidParams)
	}
	if params.Agent == "" {
		return fmt.Errorf("%w: agent is required", runtime.ErrInvalidParams)
	}

	return nil
}

// createInstanceDirectory creates the working directory for a new instance.
func (p *podmanRuntime) createInstanceDirectory(name string) (string, error) {
	instanceDir := filepath.Join(p.storageDir, "instances", name)
	if err := os.MkdirAll(instanceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create instance directory: %w", err)
	}
	return instanceDir, nil
}

// createContainerfile creates a Containerfile in the instance directory using the provided configs.
// If settings is non-empty, the files are written to an agent-settings/ subdirectory of instanceDir
// so they can be embedded in the image via a COPY instruction.
func (p *podmanRuntime) createContainerfile(instanceDir string, imageConfig *config.ImageConfig, agentConfig *config.AgentConfig, settings map[string][]byte) error {
	// Generate sudoers content
	sudoersContent := generateSudoers(imageConfig.Sudo)
	sudoersPath := filepath.Join(instanceDir, "sudoers")
	if err := os.WriteFile(sudoersPath, []byte(sudoersContent), 0644); err != nil {
		return fmt.Errorf("failed to write sudoers: %w", err)
	}

	// Write agent settings files to the build context if provided
	if len(settings) > 0 {
		settingsDir := filepath.Join(instanceDir, "agent-settings")
		if err := os.MkdirAll(settingsDir, 0755); err != nil {
			return fmt.Errorf("failed to create agent settings dir: %w", err)
		}
		for relPath, content := range settings {
			destPath := filepath.Join(settingsDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
			}
			if err := os.WriteFile(destPath, content, 0600); err != nil {
				return fmt.Errorf("failed to write agent settings file %s: %w", relPath, err)
			}
		}
	}

	// Generate Containerfile content
	containerfileContent := generateContainerfile(imageConfig, agentConfig, len(settings) > 0)
	containerfilePath := filepath.Join(instanceDir, "Containerfile")
	if err := os.WriteFile(containerfilePath, []byte(containerfileContent), 0644); err != nil {
		return fmt.Errorf("failed to write Containerfile: %w", err)
	}

	return nil
}

// buildImage builds a podman image for the instance.
func (p *podmanRuntime) buildImage(ctx context.Context, imageName, instanceDir string) error {
	containerfilePath := filepath.Join(instanceDir, "Containerfile")

	// Get current user's UID and GID
	uid := p.system.Getuid()
	gid := p.system.Getgid()

	args := []string{
		"build",
		"--build-arg", fmt.Sprintf("UID=%d", uid),
		"--build-arg", fmt.Sprintf("GID=%d", gid),
		"-t", imageName,
		"-f", containerfilePath,
		instanceDir,
	}

	l := logger.FromContext(ctx)
	if err := p.executor.Run(ctx, l.Stdout(), l.Stderr(), args...); err != nil {
		return fmt.Errorf("failed to build podman image: %w", err)
	}
	return nil
}

// buildContainerArgs builds the arguments for creating the workspace container inside the pod.
func (p *podmanRuntime) buildContainerArgs(params runtime.CreateParams, imageName string) ([]string, error) {
	args := []string{"create", "--pod", params.Name, "--name", params.Name}

	// Add environment variables from workspace config
	if params.WorkspaceConfig != nil && params.WorkspaceConfig.Environment != nil {
		for _, env := range *params.WorkspaceConfig.Environment {
			if env.Value != nil {
				args = append(args, "-e", fmt.Sprintf("%s=%s", env.Name, *env.Value))
			} else if env.Secret != nil {
				secretArg := fmt.Sprintf("%s,type=env,target=%s", *env.Secret, env.Name)
				args = append(args, "--secret", secretArg)
			}
		}
	}

	// Mount the source directory at /workspace/sources
	args = append(args, "-v", fmt.Sprintf("%s:/workspace/sources:Z", params.SourcePath))

	// Mount additional directories if specified
	if params.WorkspaceConfig != nil && params.WorkspaceConfig.Mounts != nil {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		for _, m := range *params.WorkspaceConfig.Mounts {
			args = append(args, "-v", mountVolumeArg(m, params.SourcePath, homeDir))
		}
	}

	// Set working directory to /workspace/sources
	args = append(args, "-w", "/workspace/sources")

	// Add the image name
	args = append(args, imageName)

	// Add a default command to keep the container running
	args = append(args, "sleep", "infinity")

	return args, nil
}

// createContainer creates a podman container and returns its ID.
func (p *podmanRuntime) createContainer(ctx context.Context, args []string) (string, error) {
	l := logger.FromContext(ctx)
	output, err := p.executor.Output(ctx, l.Stderr(), args...)
	if err != nil {
		return "", fmt.Errorf("failed to create podman container: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// findFreePorts returns n free TCP ports on 127.0.0.1.
// Each port is obtained by binding to :0 and immediately closing the listener.
func findFreePorts(n int) ([]int, error) {
	ports := make([]int, 0, n)
	for range n {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return nil, fmt.Errorf("failed to find free port: %w", err)
		}
		port := l.Addr().(*net.TCPAddr).Port
		l.Close()
		ports = append(ports, port)
	}
	return ports, nil
}

// renderPodYAML renders the embedded pod YAML template with the given data.
func renderPodYAML(data podTemplateData) ([]byte, error) {
	tmpl, err := template.New("pod").Parse(string(pods.OnecliPodYAML))
	if err != nil {
		return nil, fmt.Errorf("failed to parse pod template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to render pod template: %w", err)
	}
	return buf.Bytes(), nil
}

// Create creates a new Podman runtime instance.
// It uses kube play to create a pod with onecli services from the embedded YAML template,
// then adds the workspace container to the same pod.
func (p *podmanRuntime) Create(ctx context.Context, params runtime.CreateParams) (runtime.RuntimeInfo, error) {
	stepLogger := steplogger.FromContext(ctx)
	defer stepLogger.Complete()

	// Validate parameters
	if err := p.validateCreateParams(params); err != nil {
		return runtime.RuntimeInfo{}, err
	}

	// Create instance directory
	stepLogger.Start("Creating temporary build directory", "Temporary build directory created")
	instanceDir, err := p.createInstanceDirectory(params.Name)
	if err != nil {
		stepLogger.Fail(err)
		return runtime.RuntimeInfo{}, err
	}
	defer os.RemoveAll(instanceDir)

	// Load configurations
	imageConfig, err := p.config.LoadImage()
	if err != nil {
		return runtime.RuntimeInfo{}, fmt.Errorf("failed to load image config: %w", err)
	}

	agentConfig, err := p.config.LoadAgent(params.Agent)
	if err != nil {
		return runtime.RuntimeInfo{}, fmt.Errorf("failed to load agent config: %w", err)
	}

	// Create Containerfile
	stepLogger.Start("Generating Containerfile", "Containerfile generated")
	if err := p.createContainerfile(instanceDir, imageConfig, agentConfig, params.AgentSettings); err != nil {
		stepLogger.Fail(err)
		return runtime.RuntimeInfo{}, err
	}

	// Build image
	imageName := fmt.Sprintf("kdn-%s", params.Name)
	stepLogger.Start(fmt.Sprintf("Building container image: %s", imageName), "Container image built")
	if err := p.buildImage(ctx, imageName, instanceDir); err != nil {
		stepLogger.Fail(err)
		return runtime.RuntimeInfo{}, err
	}

	// Allocate random free ports for the pod
	freePorts, err := findFreePorts(1)
	if err != nil {
		return runtime.RuntimeInfo{}, fmt.Errorf("failed to allocate free ports: %w", err)
	}

	// Render the pod YAML template
	tmplData := podTemplateData{
		Name:          params.Name,
		OnecliWebPort: freePorts[0],
		OnecliVersion: defaultOnecliVersion,
	}

	tmpPodDir := filepath.Join(instanceDir, "pod")
	if err := os.MkdirAll(tmpPodDir, 0755); err != nil {
		return runtime.RuntimeInfo{}, fmt.Errorf("failed to create temp pod directory: %w", err)
	}
	tmpYAMLPath := filepath.Join(tmpPodDir, podYAMLFile)
	if err := writePodYAMLFile(tmpYAMLPath, tmplData); err != nil {
		return runtime.RuntimeInfo{}, err
	}

	// Create the pod with onecli services via kube play (--start=false keeps all containers stopped)
	stepLogger.Start("Creating onecli services", "Onecli services created")
	l := logger.FromContext(ctx)
	if err := p.executor.Run(ctx, l.Stdout(), l.Stderr(), "kube", "play", "--start=false", tmpYAMLPath); err != nil {
		stepLogger.Fail(err)
		return runtime.RuntimeInfo{}, fmt.Errorf("failed to create pod via kube play: %w", err)
	}

	// Add the workspace container to the pod
	stepLogger.Start(fmt.Sprintf("Creating workspace container: %s", params.Name), "Workspace container created")
	createArgs, err := p.buildContainerArgs(params, imageName)
	if err != nil {
		return runtime.RuntimeInfo{}, err
	}
	containerID, err := p.createContainer(ctx, createArgs)
	if err != nil {
		stepLogger.Fail(err)
		return runtime.RuntimeInfo{}, err
	}

	// Persist pod files keyed by the workspace container ID
	if err := p.writePodFiles(containerID, tmplData); err != nil {
		return runtime.RuntimeInfo{}, fmt.Errorf("failed to persist pod files: %w", err)
	}

	// Return RuntimeInfo
	info := map[string]string{
		"container_id": containerID,
		"image_name":   imageName,
		"source_path":  params.SourcePath,
		"agent":        params.Agent,
	}

	return runtime.RuntimeInfo{
		ID:    containerID,
		State: api.WorkspaceStateStopped,
		Info:  info,
	}, nil
}

// writePodYAMLFile renders and writes the pod YAML template to the given path.
func writePodYAMLFile(path string, data podTemplateData) error {
	content, err := renderPodYAML(data)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("failed to write pod YAML: %w", err)
	}
	return nil
}
