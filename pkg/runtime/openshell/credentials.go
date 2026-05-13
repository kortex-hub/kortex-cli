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

package openshell

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	workspace "github.com/openkaiden/kdn-api/workspace-configuration/go"
	"github.com/openkaiden/kdn/pkg/logger"
	"github.com/openkaiden/kdn/pkg/runtime"
)

// containerADCPath is the ADC file path inside the OpenShell sandbox.
var containerADCPath = path.Join(containerHome, ".config/gcloud/application_default_credentials.json")

// interceptCredentials detects file-based credential mounts (e.g. gcloud ADC)
// in the workspace config and returns a function that uploads the real
// credential file into the sandbox after it is created. OpenShell sandboxes
// are isolated so the real file can be uploaded directly.
//
// The intercepted mount is removed from the workspace config and the
// credential's host patterns are added to the network allow list.
func (r *openshellRuntime) interceptCredentials(ctx context.Context, params runtime.CreateParams) (func(context.Context, string) error, error) {
	wsCfg := params.WorkspaceConfig
	if r.credentialRegistry == nil || wsCfg == nil || wsCfg.Mounts == nil || len(*wsCfg.Mounts) == 0 {
		return nil, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determining home directory: %w", err)
	}

	l := logger.FromContext(ctx)

	var uploadFn func(context.Context, string) error

	for _, cred := range r.credentialRegistry.List() {
		hostPath, intercepted := cred.Detect(*wsCfg.Mounts, homeDir)
		if intercepted == nil {
			continue
		}

		if cred.Name() != "gcloud" {
			continue
		}

		if _, err := os.Stat(hostPath); os.IsNotExist(err) {
			fmt.Fprintf(l.Stderr(), "Gcloud ADC file not found at %s, skipping credential interception\n", hostPath)
			continue
		}

		removeMountFromConfig(wsCfg, intercepted)
		addHostsToNetworkConfig(wsCfg, expandWildcardHosts(cred.HostPatterns(hostPath)))

		realPath := hostPath
		uploadFn = func(ctx context.Context, sandboxName string) error {
			l := logger.FromContext(ctx)
			return r.executor.Run(ctx, l.Stdout(), l.Stderr(),
				"sandbox", "upload", sandboxName, realPath, containerADCPath,
			)
		}
	}

	return uploadFn, nil
}

// removeMountFromConfig removes the intercepted mount from the workspace config.
func removeMountFromConfig(wsCfg *workspace.WorkspaceConfiguration, intercepted *workspace.Mount) {
	if wsCfg.Mounts == nil {
		return
	}
	filtered := make([]workspace.Mount, 0, len(*wsCfg.Mounts))
	for i := range *wsCfg.Mounts {
		if &(*wsCfg.Mounts)[i] != intercepted {
			filtered = append(filtered, (*wsCfg.Mounts)[i])
		}
	}
	*wsCfg.Mounts = filtered
}

// vertexAIRegions lists the Vertex AI regional endpoint prefixes.
// OpenShell policy does not support "*-" wildcards (only "*." or "**."),
// so we enumerate them explicitly.
var vertexAIRegions = []string{
	"us-central1",
	"us-east4",
	"us-east5",
	"us-west1",
	"us-west4",
	"europe-west1",
	"europe-west2",
	"europe-west4",
	"europe-west9",
	"asia-southeast1",
	"asia-northeast1",
	"asia-northeast3",
	"me-west1",
	"northamerica-northeast1",
}

// extraVertexAIHosts lists additional Google Cloud hosts required by the
// Vertex AI authentication and model serving flow beyond the regional
// aiplatform endpoints.
var extraVertexAIHosts = []string{
	"storage.googleapis.com",
}

// expandWildcardHosts replaces unsupported wildcard patterns like
// "*-aiplatform.googleapis.com" with explicit regional endpoints and
// appends additional hosts required by the Vertex AI flow.
func expandWildcardHosts(hosts []string) []string {
	var result []string
	expanded := false
	for _, h := range hosts {
		if h == "*-aiplatform.googleapis.com" {
			for _, region := range vertexAIRegions {
				result = append(result, region+"-aiplatform.googleapis.com")
			}
			expanded = true
			continue
		}
		result = append(result, h)
	}
	if expanded {
		result = append(result, extraVertexAIHosts...)
	}
	return result
}

// addHostsToNetworkConfig appends hosts to the workspace network allow list.
func addHostsToNetworkConfig(wsCfg *workspace.WorkspaceConfiguration, hosts []string) {
	if len(hosts) == 0 {
		return
	}
	if wsCfg.Network == nil {
		wsCfg.Network = &workspace.NetworkConfiguration{}
	}
	if wsCfg.Network.Hosts == nil {
		hostsList := make([]string, 0, len(hosts))
		wsCfg.Network.Hosts = &hostsList
	}

	seen := make(map[string]bool, len(*wsCfg.Network.Hosts))
	for _, h := range *wsCfg.Network.Hosts {
		seen[strings.ToLower(h)] = true
	}
	for _, h := range hosts {
		if !seen[strings.ToLower(h)] {
			*wsCfg.Network.Hosts = append(*wsCfg.Network.Hosts, h)
		}
	}
}
