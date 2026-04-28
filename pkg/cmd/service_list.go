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
	"strings"

	"github.com/fatih/color"
	"github.com/openkaiden/kdn/pkg/secretservice"
	"github.com/openkaiden/kdn/pkg/secretservicesetup"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

// serviceListCmd contains the configuration for the service list command
type serviceListCmd struct {
	registry secretservice.Registry
	output   string
}

// registryRegistrar adapts a secretservice.Registry to satisfy
// the secretservicesetup.SecretServiceRegistrar interface.
type registryRegistrar struct {
	registry secretservice.Registry
}

func (r *registryRegistrar) RegisterSecretService(service secretservice.SecretService) error {
	return r.registry.Register(service)
}

// serviceDetail represents a secret service in JSON output
type serviceDetail struct {
	Name           string   `json:"name"`
	Description    string   `json:"description,omitempty"`
	HostsPatterns  []string `json:"hostsPatterns"`
	HeaderName     string   `json:"headerName"`
	HeaderTemplate string   `json:"headerTemplate,omitempty"`
	Path           string   `json:"path,omitempty"`
	EnvVars        []string `json:"envVars,omitempty"`
}

// servicesList represents the JSON output for the service list command
type servicesList struct {
	Items []serviceDetail `json:"items"`
}

// preRun validates the parameters and flags
func (s *serviceListCmd) preRun(cmd *cobra.Command, args []string) error {
	// Validate output format if specified
	if s.output != "" && s.output != "json" {
		return fmt.Errorf("unsupported output format: %s (supported: json)", s.output)
	}

	// Silence Cobra's default error output to stderr when JSON mode is enabled,
	// because we write the error in the JSON response to stdout instead
	if s.output == "json" {
		cmd.SilenceErrors = true
	}

	// Create registry and register secret services
	registry := secretservice.NewRegistry()
	if err := secretservicesetup.RegisterAll(&registryRegistrar{registry: registry}); err != nil {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("failed to register secret services: %w", err))
	}

	s.registry = registry

	return nil
}

// run executes the service list command logic
func (s *serviceListCmd) run(cmd *cobra.Command, args []string) error {
	names := s.registry.List()
	services := make([]secretservice.SecretService, 0, len(names))
	for _, name := range names {
		svc, err := s.registry.Get(name)
		if err == nil {
			services = append(services, svc)
		}
	}

	// Handle JSON output format
	if s.output == "json" {
		return s.outputJSON(cmd, services)
	}

	// Display the services in table format
	return s.displayTable(cmd, services)
}

// displayTable displays the services in a formatted table
func (s *serviceListCmd) displayTable(cmd *cobra.Command, services []secretservice.SecretService) error {
	out := cmd.OutOrStdout()
	if len(services) == 0 {
		fmt.Fprintln(out, "No services registered")
		return nil
	}

	// Create table with headers and formatters
	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("NAME", "HOST PATTERNS", "PATH", "HEADER", "HEADER TEMPLATE", "ENV VARS", "DESCRIPTION")
	tbl.WithWriter(out)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	// Add each service as a row
	for _, svc := range services {
		hostsPatterns := strings.Join(svc.HostsPatterns(), ", ")
		envVars := strings.Join(svc.EnvVars(), ", ")
		tbl.AddRow(svc.Name(), hostsPatterns, svc.Path(), svc.HeaderName(), svc.HeaderTemplate(), envVars, svc.Description())
	}

	// Print the table
	tbl.Print()

	return nil
}

// outputJSON converts services to JSON format and outputs them
func (s *serviceListCmd) outputJSON(cmd *cobra.Command, services []secretservice.SecretService) error {
	items := make([]serviceDetail, 0, len(services))
	for _, svc := range services {
		items = append(items, serviceDetail{
			Name:           svc.Name(),
			Description:    svc.Description(),
			HostsPatterns:  svc.HostsPatterns(),
			HeaderName:     svc.HeaderName(),
			HeaderTemplate: svc.HeaderTemplate(),
			Path:           svc.Path(),
			EnvVars:        svc.EnvVars(),
		})
	}

	response := servicesList{Items: items}

	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return outputErrorIfJSON(cmd, s.output, fmt.Errorf("failed to marshal services to JSON: %w", err))
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
	return nil
}

func NewServiceListCmd() *cobra.Command {
	c := &serviceListCmd{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered services",
		Long:  "List all secret services available for workspace configuration",
		Example: `# List all services
kdn service list

# List services in JSON format
kdn service list --output json

# List using short flag
kdn service list -o json`,
		Args:    cobra.NoArgs,
		PreRunE: c.preRun,
		RunE:    c.run,
	}

	cmd.Flags().StringVarP(&c.output, "output", "o", "", "Output format (supported: json)")
	cmd.RegisterFlagCompletionFunc("output", newOutputFlagCompletion([]string{"json"}))

	return cmd
}
