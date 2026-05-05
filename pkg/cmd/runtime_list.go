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

	"github.com/fatih/color"
	api "github.com/openkaiden/kdn-api/cli/go"
	"github.com/openkaiden/kdn/pkg/runtimesetup"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

// runtimeListCmd contains the configuration for the runtime list command.
type runtimeListCmd struct {
	output string
	listFn func() []api.RuntimeInfo
}

// preRun validates the parameters and flags.
func (r *runtimeListCmd) preRun(cmd *cobra.Command, args []string) error {
	if r.output != "" && r.output != "json" {
		return fmt.Errorf("unsupported output format: %s (supported: json)", r.output)
	}

	if r.output == "json" {
		cmd.SilenceErrors = true
	}

	return nil
}

// run executes the runtime list command logic.
func (r *runtimeListCmd) run(cmd *cobra.Command, args []string) error {
	runtimes := r.listFn()

	if r.output == "json" {
		return r.outputJSON(cmd, runtimes)
	}

	return r.displayTable(cmd, runtimes)
}

// displayTable displays the runtimes in a formatted table.
func (r *runtimeListCmd) displayTable(cmd *cobra.Command, runtimes []api.RuntimeInfo) error {
	out := cmd.OutOrStdout()
	if len(runtimes) == 0 {
		fmt.Fprintln(out, "No runtimes available")
		return nil
	}

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	tbl := table.New("NAME", "DESCRIPTION", "LOCAL")
	tbl.WithWriter(out)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, rt := range runtimes {
		local := "no"
		if rt.Local {
			local = "yes"
		}
		tbl.AddRow(rt.Name, rt.Description, local)
	}

	tbl.Print()

	return nil
}

// outputJSON outputs the runtimes as JSON.
func (r *runtimeListCmd) outputJSON(cmd *cobra.Command, runtimes []api.RuntimeInfo) error {
	items := runtimes
	if items == nil {
		items = []api.RuntimeInfo{}
	}

	list := api.RuntimesList{
		Items: items,
	}

	jsonData, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return outputErrorIfJSON(cmd, r.output, fmt.Errorf("failed to marshal runtimes to JSON: %w", err))
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(jsonData))
	return nil
}

func NewRuntimeListCmd() *cobra.Command {
	c := &runtimeListCmd{
		listFn: runtimesetup.ListRuntimes,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available runtimes",
		Long:  "List all runtime environments available for workspaces",
		Example: `# List all available runtimes
kdn runtime list

# List runtimes in JSON format
kdn runtime list --output json

# List using short flag
kdn runtime list -o json`,
		Args:    cobra.NoArgs,
		PreRunE: c.preRun,
		RunE:    c.run,
	}

	cmd.Flags().StringVarP(&c.output, "output", "o", "", "Output format (supported: json)")
	cmd.RegisterFlagCompletionFunc("output", newOutputFlagCompletion([]string{"json"}))

	return cmd
}
