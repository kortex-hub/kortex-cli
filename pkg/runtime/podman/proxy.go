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
	"path/filepath"

	"github.com/openkaiden/kdn/pkg/logger"
	"github.com/openkaiden/kdn/pkg/runtime/podman/constants"
)

// proxyImageName returns the proxy image name for a given pod name.
func proxyImageName(podN string) string {
	return fmt.Sprintf("%s-proxy", podN)
}

// proxyContainerName returns the proxy container name for a given pod name.
func proxyContainerName(podN string) string {
	return fmt.Sprintf("%s-proxy", podN)
}

// generateProxyContainerfile generates the Containerfile content for the proxy sidecar image.
// The proxy runs as root so that nftables setup works; Squid drops privileges to the squid user.
func generateProxyContainerfile(version string) string {
	return fmt.Sprintf(`FROM %s:%s

RUN dnf install -y squid nftables

RUN printf 'http_port 3128\nacl all src all\nhttp_access allow all\ncoredump_dir /var/spool/squid\n' \
    > /etc/squid/squid.conf

COPY proxy-start.sh /usr/local/bin/proxy-start.sh
RUN chmod +x /usr/local/bin/proxy-start.sh

CMD ["/usr/local/bin/proxy-start.sh"]
`, constants.BaseImageRegistry, version)
}

// generateProxyStartScript returns the content of the proxy startup script.
// The script sets nftables OUTPUT chain rules in the shared pod network namespace,
// then starts Squid. The chain policy is drop, so only explicitly accepted traffic passes.
//
// The table is deleted before recreation so the script is idempotent across pod
// stop/start cycles: Podman preserves the pod network namespace (and any nftables
// rules in it) between stop and start, only destroying it on pod removal.
func generateProxyStartScript() string {
	return `#!/bin/bash
set -e

# Restrict outbound network in the shared pod network namespace.
# Delete and recreate the filter table so rules are clean on every start
# (the pod network namespace persists across stop/start, not just pod rm).
nft delete table ip filter 2>/dev/null || true
nft add table ip filter
nft add chain ip filter output '{ type filter hook output priority 0; policy drop; }'
# ESTABLISHED/RELATED allows return packets for squid's own connections.
nft add rule ip filter output ct state established,related accept
# Localhost is always allowed (workspace container -> squid on :3128).
nft add rule ip filter output oif lo accept
# Squid's own outbound traffic must reach the internet.
nft add rule ip filter output meta skuid squid accept

# Remove a stale PID file if present (left by a previous run after pod stop/start).
rm -f /run/squid.pid

exec squid -N -f /etc/squid/squid.conf
`
}

// buildProxyContainerArgs builds the podman create arguments for the proxy sidecar container.
// CAP_NET_ADMIN is required for nft to configure the shared pod network namespace.
func buildProxyContainerArgs(podN, proxyContainer, proxyImage string) []string {
	return []string{
		"create",
		"--pod", podN,
		"--name", proxyContainer,
		"--cap-add", "NET_ADMIN",
		proxyImage,
	}
}

// buildProxyImage builds the proxy sidecar container image.
// Unlike the workspace image, the proxy image does not need UID/GID build args.
func (p *podmanRuntime) buildProxyImage(ctx context.Context, imageName, instanceDir string) error {
	containerfilePath := filepath.Join(instanceDir, "Containerfile.proxy")

	l := logger.FromContext(ctx)
	args := []string{
		"build",
		"-t", imageName,
		"-f", containerfilePath,
		instanceDir,
	}

	if err := p.executor.Run(ctx, l.Stdout(), l.Stderr(), args...); err != nil {
		return fmt.Errorf("failed to build proxy image: %w", err)
	}
	return nil
}
