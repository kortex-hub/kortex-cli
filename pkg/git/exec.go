// Copyright 2026 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package git

import (
	"context"
	"os/exec"
)

// Executor provides an interface for executing git commands
type Executor interface {
	// Output executes a git command in the specified directory and returns its standard output
	Output(ctx context.Context, dir string, args ...string) ([]byte, error)
}

// executor is the internal implementation of Executor
type executor struct{}

// Compile-time check to ensure executor implements Executor interface
var _ Executor = (*executor)(nil)

// NewExecutor creates a new git command executor
func NewExecutor() Executor {
	return &executor{}
}

// Output executes a git command in the specified directory and returns its standard output
func (e *executor) Output(ctx context.Context, dir string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	return cmd.Output()
}
