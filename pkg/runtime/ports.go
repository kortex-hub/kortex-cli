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

package runtime

import (
	"fmt"
	"net"
)

// CollectPorts returns the deduplicated list of container ports to forward,
// combining workspace configuration ports with the agent's default ports.
func CollectPorts(params CreateParams) []int {
	seen := make(map[int]bool)
	var ports []int

	if params.WorkspaceConfig != nil && params.WorkspaceConfig.Ports != nil {
		for _, p := range *params.WorkspaceConfig.Ports {
			if !seen[p] {
				seen[p] = true
				ports = append(ports, p)
			}
		}
	}

	for _, p := range params.DefaultPorts {
		if !seen[p] {
			seen[p] = true
			ports = append(ports, p)
		}
	}

	return ports
}

// IsPortFree checks whether a specific TCP port on 127.0.0.1 is available.
func IsPortFree(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	l.Close()
	return true
}
