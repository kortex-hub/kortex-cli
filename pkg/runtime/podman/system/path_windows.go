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

//go:build windows

package system

import (
	"path/filepath"
	"strings"
)

// HostPathToMachinePath converts a Windows host path to its Podman machine
// (WSL2) equivalent, e.g. C:\Users\foo → /mnt/c/Users/foo.
func HostPathToMachinePath(p string) string {
	p = filepath.ToSlash(p)
	if len(p) >= 3 && p[1] == ':' && p[2] == '/' {
		drive := strings.ToLower(string(p[0]))
		return "/mnt/" + drive + p[2:]
	}
	return p
}

// MachinePathToHostPath converts a Podman machine (WSL2) path back to a
// Windows host path, e.g. /mnt/c/Users/foo → C:\Users\foo.
func MachinePathToHostPath(p string) string {
	// /mnt/<drive>/... → <DRIVE>:\...
	if strings.HasPrefix(p, "/mnt/") && len(p) >= 7 && p[6] == '/' {
		drive := strings.ToUpper(string(p[5]))
		return drive + ":" + filepath.FromSlash(p[6:])
	}
	return filepath.FromSlash(p)
}
