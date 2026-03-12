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

import "testing"

func TestAdaptExampleForAlias(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		example     string
		originalCmd string
		aliasCmd    string
		want        string
	}{
		{
			name: "replaces command in simple example",
			example: `# List all workspaces
kortex-cli workspace list`,
			originalCmd: "workspace list",
			aliasCmd:    "list",
			want: `# List all workspaces
kortex-cli list`,
		},
		{
			name: "replaces command with flags",
			example: `# List workspaces in JSON format
kortex-cli workspace list --output json`,
			originalCmd: "workspace list",
			aliasCmd:    "list",
			want: `# List workspaces in JSON format
kortex-cli list --output json`,
		},
		{
			name: "replaces multiple occurrences",
			example: `# List all workspaces
kortex-cli workspace list

# List in JSON format
kortex-cli workspace list --output json

# List using short flag
kortex-cli workspace list -o json`,
			originalCmd: "workspace list",
			aliasCmd:    "list",
			want: `# List all workspaces
kortex-cli list

# List in JSON format
kortex-cli list --output json

# List using short flag
kortex-cli list -o json`,
		},
		{
			name: "does not replace in comments",
			example: `# Use workspace list to see all workspaces
kortex-cli workspace list`,
			originalCmd: "workspace list",
			aliasCmd:    "list",
			want: `# Use workspace list to see all workspaces
kortex-cli list`,
		},
		{
			name: "replaces remove command",
			example: `# Remove workspace by ID
kortex-cli workspace remove abc123`,
			originalCmd: "workspace remove",
			aliasCmd:    "remove",
			want: `# Remove workspace by ID
kortex-cli remove abc123`,
		},
		{
			name:        "handles empty example",
			example:     ``,
			originalCmd: "workspace list",
			aliasCmd:    "list",
			want:        ``,
		},
		{
			name: "preserves indentation",
			example: `# List all workspaces
kortex-cli workspace list

# Another example
	kortex-cli workspace list --output json`,
			originalCmd: "workspace list",
			aliasCmd:    "list",
			want: `# List all workspaces
kortex-cli list

# Another example
	kortex-cli list --output json`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := AdaptExampleForAlias(tt.example, tt.originalCmd, tt.aliasCmd)
			if got != tt.want {
				t.Errorf("AdaptExampleForAlias() mismatch:\nGot:\n%s\n\nWant:\n%s", got, tt.want)
			}
		})
	}
}
