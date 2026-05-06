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

package openshift

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	workspace "github.com/openkaiden/kdn-api/workspace-configuration/go"
)

const tokenKubeconfig = `apiVersion: v1
kind: Config
current-context: my-context
clusters:
- name: my-cluster
  cluster:
    server: https://api.cluster.example.com:6443
    certificate-authority-data: FAKECA==
contexts:
- name: my-context
  context:
    cluster: my-cluster
    user: my-user
    namespace: default
users:
- name: my-user
  user:
    token: sha256~real-token-value
`

const certKubeconfig = `apiVersion: v1
kind: Config
current-context: cert-context
clusters:
- name: cert-cluster
  cluster:
    server: https://api.cert.example.com:6443
contexts:
- name: cert-context
  context:
    cluster: cert-cluster
    user: cert-user
users:
- name: cert-user
  user:
    client-certificate-data: FAKECERT==
    client-key-data: FAKEKEY==
`

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

func TestOpenshiftCredential_Name(t *testing.T) {
	t.Parallel()
	if got := New().Name(); got != "openshift" {
		t.Errorf("Name() = %q, want %q", got, "openshift")
	}
}

func TestOpenshiftCredential_ContainerFilePath(t *testing.T) {
	t.Parallel()
	want := "/home/agent/.kube/config"
	if got := New().ContainerFilePath(); got != want {
		t.Errorf("ContainerFilePath() = %q, want %q", got, want)
	}
}

func TestOpenshiftCredential_Detect(t *testing.T) {
	t.Parallel()

	homeDir := t.TempDir()
	kubeDir := filepath.Join(homeDir, ".kube")
	if err := os.MkdirAll(kubeDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	kubeConfigPath := filepath.Join(kubeDir, "config")

	tests := []struct {
		name          string
		setup         func()
		mounts        []workspace.Mount
		wantNil       bool
		wantHostPath  string
		wantMountHost string
	}{
		{
			name:    "no mounts",
			mounts:  nil,
			wantNil: true,
		},
		{
			name: "unrelated mount",
			mounts: []workspace.Mount{
				{Host: "$HOME/projects", Target: "$HOME/projects"},
			},
			wantNil: true,
		},
		{
			name: "file mount token-based",
			setup: func() {
				if err := os.WriteFile(kubeConfigPath, []byte(tokenKubeconfig), 0600); err != nil {
					t.Fatalf("write kubeconfig: %v", err)
				}
			},
			mounts: []workspace.Mount{
				{Host: "$HOME/.kube/config", Target: "$HOME/.kube/config"},
			},
			wantNil:       false,
			wantHostPath:  kubeConfigPath,
			wantMountHost: "$HOME/.kube/config",
		},
		{
			name: "directory mount token-based",
			setup: func() {
				if err := os.WriteFile(kubeConfigPath, []byte(tokenKubeconfig), 0600); err != nil {
					t.Fatalf("write kubeconfig: %v", err)
				}
			},
			mounts: []workspace.Mount{
				{Host: "$HOME/.kube", Target: "$HOME/.kube"},
			},
			wantNil:       false,
			wantHostPath:  kubeConfigPath,
			wantMountHost: "$HOME/.kube",
		},
		{
			name: "file mount cert-based returns nil",
			setup: func() {
				if err := os.WriteFile(kubeConfigPath, []byte(certKubeconfig), 0600); err != nil {
					t.Fatalf("write kubeconfig: %v", err)
				}
			},
			mounts: []workspace.Mount{
				{Host: "$HOME/.kube/config", Target: "$HOME/.kube/config"},
			},
			wantNil: true,
		},
		{
			name: "file mount but kubeconfig missing",
			setup: func() {
				_ = os.Remove(kubeConfigPath)
			},
			mounts: []workspace.Mount{
				{Host: "$HOME/.kube/config", Target: "$HOME/.kube/config"},
			},
			wantNil: true,
		},
		{
			name: "absolute host path token-based",
			setup: func() {
				if err := os.WriteFile(kubeConfigPath, []byte(tokenKubeconfig), 0600); err != nil {
					t.Fatalf("write kubeconfig: %v", err)
				}
			},
			mounts: []workspace.Mount{
				{Host: kubeConfigPath, Target: "$HOME/.kube/config"},
			},
			wantNil:       false,
			wantHostPath:  kubeConfigPath,
			wantMountHost: kubeConfigPath,
		},
		{
			name: "absolute container target token-based",
			setup: func() {
				if err := os.WriteFile(kubeConfigPath, []byte(tokenKubeconfig), 0600); err != nil {
					t.Fatalf("write kubeconfig: %v", err)
				}
			},
			mounts: []workspace.Mount{
				{Host: "$HOME/.kube/config", Target: "/home/agent/.kube/config"},
			},
			wantNil:       false,
			wantHostPath:  kubeConfigPath,
			wantMountHost: "$HOME/.kube/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: cannot use t.Parallel() because tests share the kubeconfig file.
			if tt.setup != nil {
				tt.setup()
			}

			cred := New()
			gotPath, gotMount := cred.Detect(tt.mounts, homeDir)

			if tt.wantNil {
				if gotPath != "" || gotMount != nil {
					t.Errorf("Detect() = (%q, %+v), want (\"\", nil)", gotPath, gotMount)
				}
				return
			}
			if gotPath == "" || gotMount == nil {
				t.Fatal("Detect() returned empty result, want non-nil match")
			}
			if gotPath != tt.wantHostPath {
				t.Errorf("Detect() hostFilePath = %q, want %q", gotPath, tt.wantHostPath)
			}
			if gotMount.Host != tt.wantMountHost {
				t.Errorf("Detect() intercepted.Host = %q, want %q", gotMount.Host, tt.wantMountHost)
			}
		})
	}
}

func TestOpenshiftCredential_FakeFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	kubeconfigPath := writeFile(t, dir, "config", tokenKubeconfig)

	cred := New()
	content, err := cred.FakeFile(kubeconfigPath)
	if err != nil {
		t.Fatalf("FakeFile() error = %v", err)
	}
	if len(content) == 0 {
		t.Fatal("FakeFile() returned empty content")
	}

	s := string(content)

	// Must contain placeholder token, not the real one.
	if !strings.Contains(s, tokenPlaceholder) {
		t.Errorf("FakeFile() does not contain placeholder %q", tokenPlaceholder)
	}
	if strings.Contains(s, "sha256~real-token-value") {
		t.Error("FakeFile() contains real token value")
	}

	// Must preserve the cluster server and current-context.
	if !strings.Contains(s, "api.cluster.example.com") {
		t.Error("FakeFile() does not contain cluster server hostname")
	}
	if !strings.Contains(s, "my-context") {
		t.Error("FakeFile() does not contain current context name")
	}

	// Must not contain entries from other contexts/users/clusters that weren't in this kubeconfig.
	// (Since tokenKubeconfig only has one context, this verifies pruning is consistent.)
	if strings.Contains(s, "cert-context") {
		t.Error("FakeFile() contains unrelated context")
	}
}

func TestOpenshiftCredential_FakeFile_MissingFile(t *testing.T) {
	t.Parallel()

	cred := New()
	_, err := cred.FakeFile("/nonexistent/path/config")
	if err == nil {
		t.Fatal("FakeFile() error = nil, want error for missing file")
	}
}

func TestOpenshiftCredential_HostPatterns(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	kubeconfigPath := writeFile(t, dir, "config", tokenKubeconfig)

	cred := New()
	patterns := cred.HostPatterns(kubeconfigPath)
	if len(patterns) == 0 {
		t.Fatal("HostPatterns() returned empty slice")
	}
	if patterns[0] != "api.cluster.example.com" {
		t.Errorf("HostPatterns()[0] = %q, want %q", patterns[0], "api.cluster.example.com")
	}
}

func TestOpenshiftCredential_HostPatterns_Missing(t *testing.T) {
	t.Parallel()

	cred := New()
	patterns := cred.HostPatterns("/nonexistent/config")
	if len(patterns) != 0 {
		t.Errorf("HostPatterns() = %v, want empty for missing file", patterns)
	}
}

func TestOpenshiftCredential_EnvVars(t *testing.T) {
	t.Parallel()

	cred := New()
	vars := cred.EnvVars("")
	if vars != nil {
		t.Errorf("EnvVars() = %v, want nil", vars)
	}
}

func TestLoadKubeConfig(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for missing file", func(t *testing.T) {
		t.Parallel()

		cfg, err := loadKubeConfig("/nonexistent/config")
		if err != nil {
			t.Fatalf("loadKubeConfig() error = %v, want nil", err)
		}
		if cfg != nil {
			t.Errorf("loadKubeConfig() = %+v, want nil", cfg)
		}
	})

	t.Run("parses token-based kubeconfig", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := writeFile(t, dir, "config", tokenKubeconfig)

		cfg, err := loadKubeConfig(path)
		if err != nil {
			t.Fatalf("loadKubeConfig() error = %v", err)
		}
		if cfg == nil {
			t.Fatal("loadKubeConfig() = nil, want non-nil")
		}
		if cfg.CurrentContext != "my-context" {
			t.Errorf("CurrentContext = %q, want %q", cfg.CurrentContext, "my-context")
		}
		if len(cfg.Clusters) != 1 {
			t.Fatalf("Clusters len = %d, want 1", len(cfg.Clusters))
		}
		if cfg.Clusters[0].Cluster.Server != "https://api.cluster.example.com:6443" {
			t.Errorf("Server = %q, want %q", cfg.Clusters[0].Cluster.Server, "https://api.cluster.example.com:6443")
		}
		user := findUser(cfg, "my-user")
		if user == nil {
			t.Fatal("findUser() = nil, want non-nil")
		}
		if user.Token != "sha256~real-token-value" {
			t.Errorf("Token = %q, want %q", user.Token, "sha256~real-token-value")
		}
	})

	t.Run("returns error for invalid YAML", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := writeFile(t, dir, "config", "}{invalid yaml")

		_, err := loadKubeConfig(path)
		if err == nil {
			t.Fatal("loadKubeConfig() error = nil, want error for invalid YAML")
		}
	})
}

func TestIsTokenBased(t *testing.T) {
	t.Parallel()

	t.Run("token-based returns true", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := writeFile(t, dir, "config", tokenKubeconfig)
		cfg, _ := loadKubeConfig(path)
		if !isTokenBased(cfg) {
			t.Error("isTokenBased() = false, want true for token-based kubeconfig")
		}
	})

	t.Run("cert-based returns false", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := writeFile(t, dir, "config", certKubeconfig)
		cfg, _ := loadKubeConfig(path)
		if isTokenBased(cfg) {
			t.Error("isTokenBased() = true, want false for cert-based kubeconfig")
		}
	})
}
