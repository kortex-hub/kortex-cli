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
	"os"
	"strings"
	"testing"

	"github.com/openkaiden/kdn/pkg/cmd/testutil"
	"github.com/spf13/cobra"
)

func TestSecretRemoveCmd(t *testing.T) {
	t.Parallel()

	cmd := NewSecretRemoveCmd()
	if cmd == nil {
		t.Fatal("NewSecretRemoveCmd() returned nil")
	}
	if cmd.Use != "remove <name>" {
		t.Errorf("expected Use %q, got %q", "remove <name>", cmd.Use)
	}
}

func TestSecretRemoveCmd_Examples(t *testing.T) {
	t.Parallel()

	cmd := NewSecretRemoveCmd()
	if cmd.Example == "" {
		t.Fatal("Example field should not be empty")
	}

	commands, err := testutil.ParseExampleCommands(cmd.Example)
	if err != nil {
		t.Fatalf("failed to parse examples: %v", err)
	}

	expectedCount := 3
	if len(commands) != expectedCount {
		t.Errorf("expected %d example commands, got %d", expectedCount, len(commands))
	}

	rootCmd := NewRootCmd()
	if err := testutil.ValidateCommandExamples(rootCmd, cmd.Example); err != nil {
		t.Errorf("example validation failed: %v", err)
	}
}

func TestSecretRemoveCmd_PreRun(t *testing.T) {
	t.Parallel()

	t.Run("rejects invalid output format", func(t *testing.T) {
		t.Parallel()

		c := &secretRemoveCmd{output: "xml"}
		cmd := &cobra.Command{}
		cmd.Flags().String("storage", t.TempDir(), "")

		err := c.preRun(cmd, []string{})
		if err == nil {
			t.Fatal("expected error for invalid output format")
		}
		if !strings.Contains(err.Error(), "unsupported output format") {
			t.Errorf("expected 'unsupported output format' error, got: %v", err)
		}
	})

	t.Run("accepts json output format", func(t *testing.T) {
		t.Parallel()

		c := &secretRemoveCmd{output: "json"}
		cmd := &cobra.Command{}
		cmd.Flags().String("storage", t.TempDir(), "")

		err := c.preRun(cmd, []string{})
		if err != nil {
			t.Fatalf("preRun() failed: %v", err)
		}
		if c.store == nil {
			t.Error("expected store to be set")
		}
	})
}

func TestSecretRemoveCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("calls store with correct name", func(t *testing.T) {
		t.Parallel()

		fs := &fakeStore{}
		c := &secretRemoveCmd{store: fs}

		root := &cobra.Command{}
		var out strings.Builder
		root.SetOut(&out)
		child := &cobra.Command{RunE: c.run}
		root.AddCommand(child)

		if err := child.RunE(child, []string{"my-secret"}); err != nil {
			t.Fatalf("run() failed: %v", err)
		}

		if fs.removeName != "my-secret" {
			t.Errorf("Name: want %q, got %q", "my-secret", fs.removeName)
		}
		if !strings.Contains(out.String(), "my-secret") {
			t.Errorf("expected success message to contain secret name, got: %q", out.String())
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		t.Parallel()

		fs := &fakeStore{removeErr: os.ErrPermission}
		c := &secretRemoveCmd{store: fs}

		cmd := &cobra.Command{}
		err := c.run(cmd, []string{"x"})
		if err == nil {
			t.Fatal("expected error when store fails")
		}
	})

	t.Run("outputs JSON on success", func(t *testing.T) {
		t.Parallel()

		fs := &fakeStore{}
		c := &secretRemoveCmd{store: fs, output: "json"}

		root := &cobra.Command{}
		var out strings.Builder
		root.SetOut(&out)
		child := &cobra.Command{RunE: c.run}
		root.AddCommand(child)

		if err := child.RunE(child, []string{"my-secret"}); err != nil {
			t.Fatalf("run() failed: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal([]byte(out.String()), &result); err != nil {
			t.Fatalf("failed to parse JSON output: %v", err)
		}
		if result["name"] != "my-secret" {
			t.Errorf("expected name %q in JSON, got: %v", "my-secret", result["name"])
		}
	})

	t.Run("outputs JSON error on store failure", func(t *testing.T) {
		t.Parallel()

		fs := &fakeStore{removeErr: os.ErrPermission}
		c := &secretRemoveCmd{store: fs, output: "json"}

		root := &cobra.Command{}
		var out strings.Builder
		root.SetOut(&out)
		child := &cobra.Command{RunE: c.run}
		root.AddCommand(child)

		err := child.RunE(child, []string{"x"})
		if err == nil {
			t.Fatal("expected error when store fails")
		}

		var result map[string]interface{}
		if jsonErr := json.Unmarshal([]byte(out.String()), &result); jsonErr != nil {
			t.Fatalf("failed to parse JSON error output: %v", jsonErr)
		}
		if _, ok := result["error"]; !ok {
			t.Errorf("expected 'error' key in JSON output, got: %v", result)
		}
	})
}
