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
	"errors"
	"strings"
	"testing"

	"github.com/openkaiden/kdn/pkg/cmd/testutil"
	"github.com/openkaiden/kdn/pkg/secret"
	"github.com/spf13/cobra"
)

// fakeListStore is a Store implementation that returns a fixed list for testing.
type fakeListStore struct {
	items []secret.ListItem
	err   error
}

var _ secret.Store = (*fakeListStore)(nil)

func (f *fakeListStore) Create(params secret.CreateParams) error { return nil }

func (f *fakeListStore) List() ([]secret.ListItem, error) {
	return f.items, f.err
}

func (f *fakeListStore) Remove(name string) error { return nil }

func TestSecretListCmd(t *testing.T) {
	t.Parallel()

	cmd := NewSecretListCmd()
	if cmd == nil {
		t.Fatal("NewSecretListCmd() returned nil")
	}
	if cmd.Use != "list" {
		t.Errorf("expected Use %q, got %q", "list", cmd.Use)
	}
}

func TestSecretListCmd_Examples(t *testing.T) {
	t.Parallel()

	cmd := NewSecretListCmd()
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

func TestSecretListCmd_PreRun(t *testing.T) {
	t.Parallel()

	c := &secretListCmd{}
	cmd := &cobra.Command{}
	cmd.Flags().String("storage", t.TempDir(), "")

	if err := c.preRun(cmd, []string{}); err != nil {
		t.Fatalf("preRun() failed: %v", err)
	}
	if c.store == nil {
		t.Error("expected store to be initialised")
	}
}

func TestSecretListCmd_PreRun_InvalidOutput(t *testing.T) {
	t.Parallel()

	c := &secretListCmd{output: "xml"}
	cmd := &cobra.Command{}
	cmd.Flags().String("storage", t.TempDir(), "")

	if err := c.preRun(cmd, []string{}); err == nil {
		t.Fatal("expected error for unsupported output format")
	}
}

func TestSecretListCmd_Run(t *testing.T) {
	t.Parallel()

	t.Run("displays empty message when no secrets", func(t *testing.T) {
		t.Parallel()

		c := &secretListCmd{store: &fakeListStore{}}
		root := &cobra.Command{}
		var out bytes.Buffer
		root.SetOut(&out)
		child := &cobra.Command{RunE: c.run}
		root.AddCommand(child)

		if err := child.RunE(child, []string{}); err != nil {
			t.Fatalf("run() failed: %v", err)
		}
		if !strings.Contains(out.String(), "No secrets found") {
			t.Errorf("expected 'No secrets found' in output, got: %s", out.String())
		}
	})

	t.Run("table output contains secret fields", func(t *testing.T) {
		t.Parallel()

		c := &secretListCmd{store: &fakeListStore{
			items: []secret.ListItem{
				{Name: "my-token", Type: "github", Description: "My GitHub token"},
			},
		}}
		root := &cobra.Command{}
		var out bytes.Buffer
		root.SetOut(&out)
		child := &cobra.Command{RunE: c.run}
		root.AddCommand(child)

		if err := child.RunE(child, []string{}); err != nil {
			t.Fatalf("run() failed: %v", err)
		}
		output := out.String()
		if !strings.Contains(output, "my-token") {
			t.Errorf("expected 'my-token' in output, got: %s", output)
		}
		if !strings.Contains(output, "github") {
			t.Errorf("expected 'github' in output, got: %s", output)
		}
		if !strings.Contains(output, "My GitHub token") {
			t.Errorf("expected description in output, got: %s", output)
		}
	})

	t.Run("json output contains all fields", func(t *testing.T) {
		t.Parallel()

		c := &secretListCmd{
			output: "json",
			store: &fakeListStore{
				items: []secret.ListItem{
					{
						Name:           "my-token",
						Type:           "other",
						Description:    "My token",
						Hosts:          []string{"api.example.com"},
						Path:           "/v1",
						Header:         "Authorization",
						HeaderTemplate: "Bearer ${value}",
						Envs:           []string{"MY_TOKEN"},
					},
				},
			},
		}
		root := &cobra.Command{}
		var out bytes.Buffer
		root.SetOut(&out)
		child := &cobra.Command{RunE: c.run}
		root.AddCommand(child)

		if err := child.RunE(child, []string{}); err != nil {
			t.Fatalf("run() failed: %v", err)
		}
		output := out.String()
		for _, want := range []string{`"items"`, `"my-token"`, `"other"`, `"My token"`, `"api.example.com"`, `"/v1"`, `"Authorization"`, `"Bearer ${value}"`, `"MY_TOKEN"`} {
			if !strings.Contains(output, want) {
				t.Errorf("expected %q in JSON output, got: %s", want, output)
			}
		}
	})

	t.Run("json output empty list returns items array", func(t *testing.T) {
		t.Parallel()

		c := &secretListCmd{output: "json", store: &fakeListStore{}}
		root := &cobra.Command{}
		var out bytes.Buffer
		root.SetOut(&out)
		child := &cobra.Command{RunE: c.run}
		root.AddCommand(child)

		if err := child.RunE(child, []string{}); err != nil {
			t.Fatalf("run() failed: %v", err)
		}
		if !strings.Contains(out.String(), `"items"`) {
			t.Errorf("expected JSON with items key, got: %s", out.String())
		}
	})

	t.Run("store error propagates", func(t *testing.T) {
		t.Parallel()

		sentinel := errors.New("store error")
		c := &secretListCmd{store: &fakeListStore{err: sentinel}}
		var out bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&out)
		err := c.run(cmd, []string{})
		if err == nil {
			t.Fatal("expected error when store fails")
		}
		if !errors.Is(err, sentinel) {
			t.Errorf("expected error to wrap sentinel, got: %v", err)
		}
	})
}
