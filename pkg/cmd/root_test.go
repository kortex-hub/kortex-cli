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
	"os"
	"testing"
)

func TestRootCmd_Initialization(t *testing.T) {
	rootCmd := NewRootCmd()
	if rootCmd.Use != "kortex-cli" {
		t.Errorf("Expected Use to be 'kortex-cli', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Expected Short description to be set")
	}
}

func TestExecute_WithHelp(t *testing.T) {
	// Save original os.Args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set os.Args to call help
	os.Args = []string{"kortex-cli", "--help"}

	// Redirect output to avoid cluttering test output
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	// Call Execute() - test passes if it doesn't panic
	_ = rootCmd.Execute()
}

func TestExecute_NoArgs(t *testing.T) {
	// Save original os.Args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set os.Args with no subcommand
	os.Args = []string{"kortex-cli"}

	// Redirect output to avoid cluttering test output
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)

	// Call Execute() - test passes if it doesn't panic
	_ = rootCmd.Execute()
}
