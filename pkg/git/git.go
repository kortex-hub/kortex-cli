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
	"errors"
	"path/filepath"
	"strings"
)

var (
	// ErrNotGitRepository is returned when a directory is not inside a git repository
	ErrNotGitRepository = errors.New("not a git repository")
)

// exitCoder is an interface for errors that have an exit code
type exitCoder interface {
	ExitCode() int
}

// RepositoryInfo contains information about a git repository
type RepositoryInfo struct {
	// RootDir is the absolute path to the repository root
	RootDir string
	// RemoteURL is the URL of the origin remote (empty if no remote)
	RemoteURL string
	// RelativePath is the relative path from repository root to the queried directory
	RelativePath string
}

// Detector provides methods for detecting and analyzing git repositories
type Detector interface {
	// DetectRepository checks if the given directory is inside a git repository
	// and returns information about the repository if found.
	// Returns ErrNotGitRepository if not inside a git repository.
	DetectRepository(ctx context.Context, dir string) (*RepositoryInfo, error)
}

// detector is the internal implementation of Detector
type detector struct {
	executor Executor
}

// Compile-time check to ensure detector implements Detector interface
var _ Detector = (*detector)(nil)

// NewDetector creates a new git repository detector
func NewDetector() Detector {
	return &detector{
		executor: NewExecutor(),
	}
}

// newDetectorWithExecutor creates a new detector with a custom executor (for testing)
func newDetectorWithExecutor(executor Executor) Detector {
	return &detector{
		executor: executor,
	}
}

// DetectRepository checks if the given directory is inside a git repository
func (d *detector) DetectRepository(ctx context.Context, dir string) (*RepositoryInfo, error) {
	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Check if inside a git repository by running git rev-parse --show-toplevel
	output, err := d.executor.Output(ctx, absDir, "rev-parse", "--show-toplevel")
	if err != nil {
		// Exit status 128 means not a git repository
		if exitErr, ok := err.(exitCoder); ok && exitErr.ExitCode() == 128 {
			return nil, ErrNotGitRepository
		}
		return nil, err
	}

	rootDir := strings.TrimSpace(string(output))
	if rootDir == "" {
		return nil, ErrNotGitRepository
	}

	// Calculate relative path from repository root to the queried directory
	relativePath, err := filepath.Rel(rootDir, absDir)
	if err != nil {
		return nil, err
	}
	// If at repository root, relative path will be "."
	if relativePath == "." {
		relativePath = ""
	}

	// Get the remote URL (try upstream first, then origin)
	remoteURL := ""
	output, err = d.executor.Output(ctx, rootDir, "remote", "get-url", "upstream")
	if err == nil {
		remoteURL = strings.TrimSpace(string(output))
		// Remove .git suffix if present
		remoteURL = strings.TrimSuffix(remoteURL, ".git")
	} else {
		// upstream not found, try origin
		output, err = d.executor.Output(ctx, rootDir, "remote", "get-url", "origin")
		if err == nil {
			remoteURL = strings.TrimSpace(string(output))
			// Remove .git suffix if present
			remoteURL = strings.TrimSuffix(remoteURL, ".git")
		}
		// Ignore errors - remote might not exist
	}

	return &RepositoryInfo{
		RootDir:      rootDir,
		RemoteURL:    remoteURL,
		RelativePath: relativePath,
	}, nil
}
