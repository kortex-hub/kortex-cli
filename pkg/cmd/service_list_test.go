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

	"github.com/openkaiden/kdn/pkg/cmd/testutil"
	"github.com/spf13/cobra"
)

func TestServiceListCmd(t *testing.T) {
	t.Parallel()

	cmd := NewServiceListCmd()
	if cmd == nil {
		t.Fatal("NewServiceListCmd() returned nil")
	}

	if cmd.Use != "list" {
		t.Errorf("Expected Use to be 'list', got '%s'", cmd.Use)
	}
}

func TestServiceListCmd_Examples(t *testing.T) {
	t.Parallel()

	cmd := NewServiceListCmd()

	if cmd.Example == "" {
		t.Fatal("Example field should not be empty")
	}

	commands, err := testutil.ParseExampleCommands(cmd.Example)
	if err != nil {
		t.Fatalf("Failed to parse examples: %v", err)
	}

	expectedCount := 3
	if len(commands) != expectedCount {
		t.Errorf("Expected %d example commands, got %d", expectedCount, len(commands))
	}

	rootCmd := NewRootCmd()
	err = testutil.ValidateCommandExamples(rootCmd, cmd.Example)
	if err != nil {
		t.Errorf("Example validation failed: %v", err)
	}
}

func TestServiceListCmd_PreRun(t *testing.T) {
	t.Parallel()

	t.Run("accepts empty output flag", func(t *testing.T) {
		t.Parallel()

		c := &serviceListCmd{}
		cmd := &cobra.Command{}

		err := c.preRun(cmd, []string{})
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}
	})

	t.Run("accepts json output format", func(t *testing.T) {
		t.Parallel()

		c := &serviceListCmd{output: "json"}
		cmd := &cobra.Command{}

		err := c.preRun(cmd, []string{})
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}
	})

	t.Run("rejects invalid output format", func(t *testing.T) {
		t.Parallel()

		c := &serviceListCmd{output: "xml"}
		cmd := &cobra.Command{}

		err := c.preRun(cmd, []string{})
		if err == nil {
			t.Fatal("Expected error for invalid output format")
		}

		if !strings.Contains(err.Error(), "unsupported output format") {
			t.Errorf("Expected 'unsupported output format' error, got: %v", err)
		}
	})
}

func TestServiceListCmd_E2E(t *testing.T) {
	t.Parallel()

	t.Run("table output contains expected headers and data", func(t *testing.T) {
		t.Parallel()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"service", "list"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "NAME") {
			t.Errorf("Expected output to contain 'NAME' header, got: %s", output)
		}
		if !strings.Contains(output, "HOST PATTERN") {
			t.Errorf("Expected output to contain 'HOST PATTERN' header, got: %s", output)
		}
		if !strings.Contains(output, "HEADER") {
			t.Errorf("Expected output to contain 'HEADER' header, got: %s", output)
		}
		if !strings.Contains(output, "PATH") {
			t.Errorf("Expected output to contain 'PATH' header, got: %s", output)
		}
		if !strings.Contains(output, "HEADER TEMPLATE") {
			t.Errorf("Expected output to contain 'HEADER TEMPLATE' header, got: %s", output)
		}
		if !strings.Contains(output, "ENV VARS") {
			t.Errorf("Expected output to contain 'ENV VARS' header, got: %s", output)
		}
		if !strings.Contains(output, "DESCRIPTION") {
			t.Errorf("Expected output to contain 'DESCRIPTION' header, got: %s", output)
		}
		if !strings.Contains(output, "Bearer ${value}") {
			t.Errorf("Expected output to contain 'Bearer ${value}', got: %s", output)
		}
		if !strings.Contains(output, "github") {
			t.Errorf("Expected output to contain 'github' service, got: %s", output)
		}
	})

	t.Run("json output has expected structure", func(t *testing.T) {
		t.Parallel()

		rootCmd := NewRootCmd()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetArgs([]string{"service", "list", "-o", "json"})

		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("Execute() failed: %v", err)
		}

		var response servicesList
		if err := json.Unmarshal(buf.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if len(response.Items) == 0 {
			t.Fatal("Expected at least one service in JSON output")
		}

		// Find github service
		found := false
		for _, svc := range response.Items {
			if svc.Name == "github" {
				found = true
				if svc.Description == "" {
					t.Error("Expected non-empty Description for github service")
				}
				if len(svc.HostsPatterns) == 0 || svc.HostsPatterns[0] != "api.github.com" {
					t.Errorf("Expected HostsPatterns %v, got %v", []string{"api.github.com"}, svc.HostsPatterns)
				}
				if svc.HeaderName != "Authorization" {
					t.Errorf("Expected HeaderName %q, got %q", "Authorization", svc.HeaderName)
				}
				if len(svc.EnvVars) != 2 {
					t.Errorf("Expected 2 env vars, got %d", len(svc.EnvVars))
				}
				break
			}
		}

		if !found {
			t.Error("Expected to find 'github' service in JSON output")
		}
	})
}
