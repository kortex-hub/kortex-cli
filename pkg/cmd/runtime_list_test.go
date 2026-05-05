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
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	api "github.com/openkaiden/kdn-api/cli/go"
	"github.com/openkaiden/kdn/pkg/cmd/testutil"
	"github.com/spf13/cobra"
)

func TestRuntimeListCmd(t *testing.T) {
	t.Parallel()

	cmd := NewRuntimeListCmd()
	if cmd == nil {
		t.Fatal("NewRuntimeListCmd() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got '%s'", cmd.Use)
	}
}

func TestRuntimeListCmd_PreRun(t *testing.T) {
	t.Parallel()

	t.Run("accepts empty output format", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{output: ""}
		cmd := &cobra.Command{}

		err := c.preRun(cmd, nil)
		if err != nil {
			t.Fatalf("preRun failed: %v", err)
		}
	})

	t.Run("accepts json output format", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{output: "json"}
		cmd := &cobra.Command{}

		err := c.preRun(cmd, nil)
		if err != nil {
			t.Fatalf("preRun failed: %v", err)
		}
	})

	t.Run("rejects unsupported output format", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{output: "yaml"}
		cmd := &cobra.Command{}

		err := c.preRun(cmd, nil)
		if err == nil {
			t.Fatal("Expected error for unsupported output format, got nil")
		}
		if !strings.Contains(err.Error(), "unsupported output format") {
			t.Errorf("Expected error about unsupported format, got: %s", err.Error())
		}
	})
}

func TestRuntimeListCmd_TextOutput(t *testing.T) {
	t.Parallel()

	t.Run("displays table with runtimes", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{
			listFn: func() []api.RuntimeInfo {
				return []api.RuntimeInfo{
					{Name: "podman", Description: "Container-based workspaces using Podman", Local: true},
				}
			},
		}

		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := c.run(cmd, nil)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "podman") {
			t.Errorf("Expected output to contain 'podman', got: %s", output)
		}
		if !strings.Contains(output, "Container-based workspaces using Podman") {
			t.Errorf("Expected output to contain description, got: %s", output)
		}
		if !strings.Contains(output, "yes") {
			t.Errorf("Expected output to contain 'yes' for local, got: %s", output)
		}
	})

	t.Run("displays remote runtime as no", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{
			listFn: func() []api.RuntimeInfo {
				return []api.RuntimeInfo{
					{Name: "k8s", Description: "Kubernetes-based workspaces", Local: false},
				}
			},
		}

		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := c.run(cmd, nil)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "no") {
			t.Errorf("Expected output to contain 'no' for remote runtime, got: %s", output)
		}
	})

	t.Run("displays message when no runtimes available", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{
			listFn: func() []api.RuntimeInfo {
				return nil
			},
		}

		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := c.run(cmd, nil)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "No runtimes available") {
			t.Errorf("Expected 'No runtimes available', got: %s", output)
		}
	})

	t.Run("displays multiple runtimes", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{
			listFn: func() []api.RuntimeInfo {
				return []api.RuntimeInfo{
					{Name: "podman", Description: "Container-based workspaces using Podman", Local: true},
					{Name: "k8s", Description: "Kubernetes-based workspaces", Local: false},
				}
			},
		}

		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := c.run(cmd, nil)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "podman") {
			t.Errorf("Expected output to contain 'podman', got: %s", output)
		}
		if !strings.Contains(output, "k8s") {
			t.Errorf("Expected output to contain 'k8s', got: %s", output)
		}
	})
}

func TestRuntimeListCmd_JSONOutput(t *testing.T) {
	t.Parallel()

	t.Run("outputs valid JSON with runtimes", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{
			output: "json",
			listFn: func() []api.RuntimeInfo {
				return []api.RuntimeInfo{
					{Name: "podman", Description: "Container-based workspaces using Podman", Local: true},
				}
			},
		}

		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := c.run(cmd, nil)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		var result api.RuntimesList
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v\nOutput: %s", err, buf.String())
		}

		if len(result.Items) != 1 {
			t.Fatalf("Expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].Name != "podman" {
			t.Errorf("Expected name 'podman', got %q", result.Items[0].Name)
		}
		if result.Items[0].Description != "Container-based workspaces using Podman" {
			t.Errorf("Expected description mismatch, got %q", result.Items[0].Description)
		}
		if !result.Items[0].Local {
			t.Error("Expected local to be true")
		}
	})

	t.Run("outputs empty items array when no runtimes", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{
			output: "json",
			listFn: func() []api.RuntimeInfo {
				return nil
			},
		}

		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := c.run(cmd, nil)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		var result api.RuntimesList
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v\nOutput: %s", err, buf.String())
		}

		if result.Items == nil {
			t.Fatal("Expected items to be an empty array, got nil")
		}
		if len(result.Items) != 0 {
			t.Errorf("Expected 0 items, got %d", len(result.Items))
		}
	})

	t.Run("outputs multiple runtimes with correct local flags", func(t *testing.T) {
		t.Parallel()

		c := &runtimeListCmd{
			output: "json",
			listFn: func() []api.RuntimeInfo {
				return []api.RuntimeInfo{
					{Name: "podman", Description: "Container-based workspaces", Local: true},
					{Name: "k8s", Description: "Kubernetes workspaces", Local: false},
				}
			},
		}

		cmd := &cobra.Command{}
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := c.run(cmd, nil)
		if err != nil {
			t.Fatalf("run failed: %v", err)
		}

		var result api.RuntimesList
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v\nOutput: %s", err, buf.String())
		}

		if len(result.Items) != 2 {
			t.Fatalf("Expected 2 items, got %d", len(result.Items))
		}
		if !result.Items[0].Local {
			t.Error("Expected first runtime to be local")
		}
		if result.Items[1].Local {
			t.Error("Expected second runtime to not be local")
		}
	})
}

func TestRuntimeListCmd_E2E(t *testing.T) {
	t.Parallel()

	t.Run("runtime list text output", func(t *testing.T) {
		t.Parallel()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"runtime", "list"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}
	})

	t.Run("runtime list json output", func(t *testing.T) {
		t.Parallel()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"runtime", "list", "--output", "json"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		var result api.RuntimesList
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v\nOutput: %s", err, buf.String())
		}

		if result.Items == nil {
			t.Fatal("Expected items array, got nil")
		}
	})

	t.Run("runtime list with short output flag", func(t *testing.T) {
		t.Parallel()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"runtime", "list", "-o", "json"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		var result api.RuntimesList
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal JSON: %v\nOutput: %s", err, buf.String())
		}
	})

	t.Run("runtime list with unsupported output format", func(t *testing.T) {
		t.Parallel()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"runtime", "list", "--output", "yaml"})

		err := rootCmd.Execute()
		if err == nil {
			t.Fatal("Expected error for unsupported output format")
		}
		if !strings.Contains(err.Error(), "unsupported output format") {
			t.Errorf("Expected error about unsupported format, got: %s", err.Error())
		}
	})
}

func TestRuntimeListCmd_Examples(t *testing.T) {
	t.Parallel()

	listCmd := NewRuntimeListCmd()

	if listCmd.Example == "" {
		t.Fatal("Example field should not be empty")
	}

	commands, err := testutil.ParseExampleCommands(listCmd.Example)
	if err != nil {
		t.Fatalf("Failed to parse examples: %v", err)
	}

	if len(commands) == 0 {
		t.Fatal("Expected at least one example command")
	}

	rootCmd := NewRootCmd()
	err = testutil.ValidateCommandExamples(rootCmd, listCmd.Example)
	if err != nil {
		t.Errorf("Example validation failed: %v", err)
	}
}
