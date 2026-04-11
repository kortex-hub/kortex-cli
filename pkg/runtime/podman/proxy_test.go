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
)

func TestGenerateProxyContainerfile(t *testing.T) {
	t.Parallel()

	t.Run("includes squid and nftables installation", func(t *testing.T) {
		t.Parallel()

		content := generateProxyContainerfile("42")

		if !strings.Contains(content, "FROM registry.fedoraproject.org/fedora:42") {
			t.Error("Expected proxy Containerfile to use fedora:42 base image")
		}
		if !strings.Contains(content, "squid") {
			t.Error("Expected proxy Containerfile to install squid")
		}
		if !strings.Contains(content, "nftables") {
			t.Error("Expected proxy Containerfile to install nftables")
		}
		if !strings.Contains(content, "proxy-start.sh") {
			t.Error("Expected proxy Containerfile to reference proxy-start.sh")
		}
		if !strings.Contains(content, "/usr/local/bin/proxy-start.sh") {
			t.Error("Expected proxy Containerfile to install proxy-start.sh to /usr/local/bin")
		}
	})

	t.Run("uses provided version", func(t *testing.T) {
		t.Parallel()

		content := generateProxyContainerfile("latest")
		if !strings.Contains(content, "fedora:latest") {
			t.Errorf("Expected proxy Containerfile to use version 'latest', got:\n%s", content)
		}

		content2 := generateProxyContainerfile("40")
		if !strings.Contains(content2, "fedora:40") {
			t.Errorf("Expected proxy Containerfile to use version '40', got:\n%s", content2)
		}
	})
}

func TestGenerateProxyStartScript(t *testing.T) {
	t.Parallel()

	t.Run("contains nftables setup", func(t *testing.T) {
		t.Parallel()

		script := generateProxyStartScript()

		if !strings.Contains(script, "nft") {
			t.Error("Expected proxy start script to contain nft commands")
		}
		if !strings.Contains(script, "policy drop") {
			t.Error("Expected proxy start script to set drop policy")
		}
		if !strings.Contains(script, "established,related") {
			t.Error("Expected proxy start script to allow established/related traffic")
		}
		if !strings.Contains(script, "oif lo") {
			t.Error("Expected proxy start script to allow loopback traffic")
		}
		if !strings.Contains(script, "skuid squid") {
			t.Error("Expected proxy start script to allow squid user outbound traffic")
		}
	})

	t.Run("is idempotent - deletes table before recreating", func(t *testing.T) {
		t.Parallel()

		script := generateProxyStartScript()

		if !strings.Contains(script, "nft delete table ip filter") {
			t.Error("Expected proxy start script to delete existing filter table for idempotency")
		}
	})

	t.Run("cleans up stale PID file before starting squid", func(t *testing.T) {
		t.Parallel()

		script := generateProxyStartScript()

		if !strings.Contains(script, "rm -f /run/squid.pid") {
			t.Error("Expected proxy start script to remove stale PID file before starting squid")
		}
	})

	t.Run("starts squid in foreground", func(t *testing.T) {
		t.Parallel()

		script := generateProxyStartScript()

		if !strings.Contains(script, "exec squid -N") {
			t.Error("Expected proxy start script to start squid in foreground with -N flag")
		}
	})
}

func TestBuildProxyContainerArgs(t *testing.T) {
	t.Parallel()

	t.Run("produces correct args with CAP_NET_ADMIN", func(t *testing.T) {
		t.Parallel()

		args := buildProxyContainerArgs("kdn-myws", "kdn-myws-proxy", "kdn-myws-proxy")

		argsStr := strings.Join(args, " ")

		if !strings.Contains(argsStr, "create") {
			t.Error("Expected 'create' subcommand")
		}
		if !strings.Contains(argsStr, "--pod kdn-myws") {
			t.Error("Expected --pod flag with pod name")
		}
		if !strings.Contains(argsStr, "--name kdn-myws-proxy") {
			t.Error("Expected --name flag with proxy container name")
		}
		if !strings.Contains(argsStr, "--cap-add NET_ADMIN") {
			t.Error("Expected --cap-add NET_ADMIN for nftables support")
		}
		if !strings.Contains(argsStr, "kdn-myws-proxy") {
			t.Error("Expected proxy image name")
		}
	})

	t.Run("does not include sleep infinity", func(t *testing.T) {
		t.Parallel()

		args := buildProxyContainerArgs("kdn-myws", "kdn-myws-proxy", "kdn-myws-proxy")
		argsStr := strings.Join(args, " ")

		if strings.Contains(argsStr, "sleep") {
			t.Error("Proxy container should use its own CMD, not sleep infinity")
		}
	})
}
