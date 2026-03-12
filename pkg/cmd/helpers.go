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

package cmd

import "strings"

// AdaptExampleForAlias replaces the original command with the alias command
// in the example string, but only in command lines (not in comments).
// This is useful for alias commands that want to inherit examples from
// their original commands while showing the alias syntax.
//
// Example:
//
//	original := `# List all workspaces
//	kortex-cli workspace list
//
//	# List in JSON format
//	kortex-cli workspace list --output json`
//
//	adapted := AdaptExampleForAlias(original, "workspace list", "list")
//	// Result:
//	// `# List all workspaces
//	// kortex-cli list
//	//
//	// # List in JSON format
//	// kortex-cli list --output json`
func AdaptExampleForAlias(example, originalCmd, aliasCmd string) string {
	lines := strings.Split(example, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// Only replace in command lines (starting with kortex-cli), not in comments
		if strings.HasPrefix(trimmed, "kortex-cli ") {
			line = strings.Replace(line, originalCmd, aliasCmd, 1)
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
