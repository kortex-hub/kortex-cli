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

package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	api "github.com/openkaiden/kdn-api/cli/go"
	"github.com/openkaiden/kdn/pkg/secret"
	"github.com/openkaiden/kdn/pkg/secretservicesetup"
	"github.com/spf13/cobra"
)

type secretCreateCmd struct {
	secretType     string
	value          string
	description    string
	hosts          []string
	path           string
	header         string
	headerTemplate string
	envs           []string
	output         string
	store          secret.Store
	validTypes     []string
}

func (s *secretCreateCmd) isValidType(t string) bool {
	for _, v := range s.validTypes {
		if t == v {
			return true
		}
	}
	return false
}

func (s *secretCreateCmd) preRun(cmd *cobra.Command, args []string) error {
	if s.output != "" && s.output != "json" {
		return fmt.Errorf("unsupported output format: %s (supported: json)", s.output)
	}
	if s.output == "json" {
		cmd.SilenceErrors = true
	}

	if s.secretType == "" {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--type is required"))
	}
	if !s.isValidType(s.secretType) {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("invalid --type %q: must be one of %s", s.secretType, strings.Join(s.validTypes, ", ")))
	}
	if s.value == "" {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--value is required"))
	}

	if s.secretType == secret.TypeOther {
		if len(s.hosts) == 0 {
			return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--host is required when --type=%s", secret.TypeOther))
		}
		if !cmd.Flags().Changed("header") {
			return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--header is required when --type=%s", secret.TypeOther))
		}
	} else {
		// Descriptor flags are not valid for named types
		if len(s.hosts) > 0 {
			return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--host is only valid when --type=%s", secret.TypeOther))
		}
		if cmd.Flags().Changed("path") {
			return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--path is only valid when --type=%s", secret.TypeOther))
		}
		if cmd.Flags().Changed("header") {
			return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--header is only valid when --type=%s", secret.TypeOther))
		}
		if cmd.Flags().Changed("headerTemplate") {
			return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--headerTemplate is only valid when --type=%s", secret.TypeOther))
		}
		if len(s.envs) > 0 {
			return outputErrorIfJSON(cmd, s.output, fmt.Errorf("--env is only valid when --type=%s", secret.TypeOther))
		}
	}

	storageDir, err := cmd.Flags().GetString("storage")
	if err != nil {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("failed to read --storage flag: %w", err))
	}

	absStorageDir, err := filepath.Abs(storageDir)
	if err != nil {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("failed to resolve storage directory path: %w", err))
	}

	s.store = secret.NewStore(absStorageDir)
	return nil
}

func (s *secretCreateCmd) run(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := s.store.Create(secret.CreateParams{
		Name:           name,
		Type:           s.secretType,
		Value:          s.value,
		Description:    s.description,
		Hosts:          s.hosts,
		Path:           s.path,
		Header:         s.header,
		HeaderTemplate: s.headerTemplate,
		Envs:           s.envs,
	}); err != nil {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("failed to create secret: %w", err))
	}

	if s.output == "json" {
		return s.outputJSON(cmd, name)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Secret %q created successfully\n", name)
	return nil
}

func (s *secretCreateCmd) outputJSON(cmd *cobra.Command, name string) error {
	response := api.SecretName{Name: name}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("failed to marshal secret to JSON: %w", err))
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
	return nil
}

func NewSecretCreateCmd() *cobra.Command {
	registeredTypes := secretservicesetup.ListAvailable()
	sort.Strings(registeredTypes)
	validTypes := append(registeredTypes, secret.TypeOther)
	typesStr := strings.Join(validTypes, ", ")

	c := &secretCreateCmd{validTypes: validTypes}

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new secret",
		Long: fmt.Sprintf(`Create a new secret and store its value in the system keychain.

The secret value is stored securely in the system keychain (GNOME Keyring on
Linux, Keychain on macOS, DPAPI on Windows). Non-sensitive metadata (type,
hosts, path, header template, envs) is persisted in the kdn storage directory.

Accepted types: %s.

When --type=other, --host and --header are required; --path, --headerTemplate,
and --env are optional. For any other type, these flags must not be
specified.`, typesStr),
		Example: `# Create a GitHub token secret
kdn secret create my-github-token --type github --value ghp_mytoken

# Create a custom secret (type=other) with all descriptor flags
kdn secret create my-api-key --type other --value secret123 --host api.example.com --host dev.example.com --path /api/v1 --header Authorization --headerTemplate "Bearer ${value}" --env MY_API_KEY --env API_KEY

# Create a custom secret (type=other) with only required flags
kdn secret create my-api-key --type other --value secret123 --host api.example.com --header Authorization

# Create a GitHub token secret with JSON output
kdn secret create my-github-token --type github --value ghp_mytoken --output json`,
		Args:    cobra.ExactArgs(1),
		PreRunE: c.preRun,
		RunE:    c.run,
	}

	cmd.Flags().StringVar(&c.secretType, "type", "", fmt.Sprintf("Type of secret (%s)", typesStr))
	cmd.RegisterFlagCompletionFunc("type", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return validTypes, cobra.ShellCompDirectiveNoFileComp
	})
	cmd.Flags().StringVar(&c.value, "value", "", "Secret value to store in the system keychain")
	cmd.Flags().StringVar(&c.description, "description", "", "Optional human-readable description of the secret")
	cmd.Flags().StringArrayVar(&c.hosts, "host", nil, "Host pattern (required for --type=other, can be specified multiple times)")
	cmd.Flags().StringVar(&c.path, "path", "", "URL path restriction (optional for --type=other)")
	cmd.Flags().StringVar(&c.header, "header", "", "HTTP header name (required for --type=other)")
	cmd.Flags().StringVar(&c.headerTemplate, "headerTemplate", "", "HTTP header value template using ${value} as placeholder (optional for --type=other)")
	cmd.Flags().StringArrayVar(&c.envs, "env", nil, "Environment variable name to expose the secret value (optional for --type=other, can be specified multiple times)")
	cmd.Flags().StringVarP(&c.output, "output", "o", "", "Output format (supported: json)")
	cmd.RegisterFlagCompletionFunc("output", newOutputFlagCompletion([]string{"json"}))

	return cmd
}
