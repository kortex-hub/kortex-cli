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
	"strings"
	"testing"
)

func TestRuntimeCmd(t *testing.T) {
	t.Parallel()

	cmd := NewRuntimeCmd()
	if cmd == nil {
		t.Fatal("NewRuntimeCmd() returned nil")
	}

	if cmd.Use != "runtime" {
		t.Errorf("Expected Use to be 'runtime', got '%s'", cmd.Use)
	}

	foundList := false
	for _, subCmd := range cmd.Commands() {
		if subCmd.Use == "list" {
			foundList = true
			break
		}
	}

	if !foundList {
		t.Error("Expected runtime command to have 'list' subcommand")
	}
}

func TestRuntimeCmd_UnknownCommand(t *testing.T) {
	t.Parallel()

	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"runtime", "foobar"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("Expected Execute() to return an error for unknown command")
	}

	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("Expected error to contain 'unknown command', got: %s", err.Error())
	}
	if !strings.Contains(err.Error(), "foobar") {
		t.Errorf("Expected error to contain 'foobar', got: %s", err.Error())
	}
}
