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
	"testing"

	"github.com/openkaiden/kdn/pkg/cmd/testutil"
)

func TestAutoconfCmd(t *testing.T) {
	t.Parallel()

	cmd := NewAutoconfCmd()
	if cmd == nil {
		t.Fatal("NewAutoconfCmd() returned nil")
	}
	if cmd.Use != "autoconf" {
		t.Errorf("expected Use %q, got %q", "autoconf", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected non-empty Short description")
	}
}

func TestAutoconfCmd_Examples(t *testing.T) {
	t.Parallel()

	cmd := NewAutoconfCmd()
	if cmd.Example == "" {
		t.Fatal("Example field should not be empty")
	}

	commands, err := testutil.ParseExampleCommands(cmd.Example)
	if err != nil {
		t.Fatalf("failed to parse examples: %v", err)
	}

	expectedCount := 4
	if len(commands) != expectedCount {
		t.Errorf("expected %d example commands, got %d", expectedCount, len(commands))
	}

	rootCmd := NewRootCmd()
	if err := testutil.ValidateCommandExamples(rootCmd, cmd.Example); err != nil {
		t.Errorf("example validation failed: %v", err)
	}
}
