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
	"os/exec"
	"path/filepath"
	"testing"
)

// fakeExecutor is a test implementation of Executor
type fakeExecutor struct {
	revParseOutput []byte
	revParseError  error
	upstreamOutput []byte
	upstreamError  error
	originOutput   []byte
	originError    error
}

func (f *fakeExecutor) Output(ctx context.Context, dir string, args ...string) ([]byte, error) {
	if len(args) == 0 {
		return nil, errors.New("no args provided")
	}

	switch args[0] {
	case "rev-parse":
		if f.revParseError != nil {
			return nil, f.revParseError
		}
		return f.revParseOutput, nil
	case "remote":
		if len(args) < 3 {
			return nil, errors.New("remote command requires subcommand and name")
		}
		// args[1] is "get-url", args[2] is remote name
		remoteName := args[2]
		switch remoteName {
		case "upstream":
			if f.upstreamError != nil {
				return nil, f.upstreamError
			}
			return f.upstreamOutput, nil
		case "origin":
			if f.originError != nil {
				return nil, f.originError
			}
			return f.originOutput, nil
		default:
			return nil, errors.New("unknown remote: " + remoteName)
		}
	default:
		return nil, errors.New("unknown command: " + args[0])
	}
}

func TestDetector_DetectRepository(t *testing.T) {
	t.Parallel()

	t.Run("detects git repository with origin remote", func(t *testing.T) {
		t.Parallel()

		testDir := t.TempDir()

		fake := &fakeExecutor{
			revParseOutput: []byte(testDir + "\n"),
			upstreamError:  &exec.ExitError{}, // upstream doesn't exist
			originOutput:   []byte("https://github.com/user/repo.git\n"),
		}

		detector := newDetectorWithExecutor(fake)
		info, err := detector.DetectRepository(context.Background(), testDir)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if info.RootDir != testDir {
			t.Errorf("Expected root dir %s, got %s", testDir, info.RootDir)
		}

		if info.RemoteURL != "https://github.com/user/repo" {
			t.Errorf("Expected remote URL https://github.com/user/repo (without .git), got %s", info.RemoteURL)
		}

		if info.RelativePath != "" {
			t.Errorf("Expected empty relative path at root, got %s", info.RelativePath)
		}
	})

	t.Run("detects git repository with upstream remote", func(t *testing.T) {
		t.Parallel()

		testDir := t.TempDir()

		fake := &fakeExecutor{
			revParseOutput: []byte(testDir + "\n"),
			upstreamOutput: []byte("https://github.com/upstream/repo.git\n"),
			originOutput:   []byte("https://github.com/user/fork.git\n"),
		}

		detector := newDetectorWithExecutor(fake)
		info, err := detector.DetectRepository(context.Background(), testDir)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if info.RootDir != testDir {
			t.Errorf("Expected root dir %s, got %s", testDir, info.RootDir)
		}

		if info.RemoteURL != "https://github.com/upstream/repo" {
			t.Errorf("Expected upstream remote URL https://github.com/upstream/repo (without .git), got %s", info.RemoteURL)
		}

		if info.RelativePath != "" {
			t.Errorf("Expected empty relative path at root, got %s", info.RelativePath)
		}
	})

	t.Run("detects git repository with remote in subdirectory", func(t *testing.T) {
		t.Parallel()

		repoRoot := t.TempDir()
		subDir := filepath.Join(repoRoot, "sub", "path")

		fake := &fakeExecutor{
			revParseOutput: []byte(repoRoot + "\n"),
			upstreamError:  &exec.ExitError{}, // upstream doesn't exist
			originOutput:   []byte("https://github.com/user/repo.git\n"),
		}

		detector := newDetectorWithExecutor(fake)
		info, err := detector.DetectRepository(context.Background(), subDir)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if info.RootDir != repoRoot {
			t.Errorf("Expected root dir %s, got %s", repoRoot, info.RootDir)
		}

		if info.RemoteURL != "https://github.com/user/repo" {
			t.Errorf("Expected remote URL https://github.com/user/repo (without .git), got %s", info.RemoteURL)
		}

		expectedRelPath := filepath.Join("sub", "path")
		if info.RelativePath != expectedRelPath {
			t.Errorf("Expected relative path %s, got %s", expectedRelPath, info.RelativePath)
		}
	})

	t.Run("detects git repository without any remote", func(t *testing.T) {
		t.Parallel()

		testDir := t.TempDir()

		fake := &fakeExecutor{
			revParseOutput: []byte(testDir + "\n"),
			upstreamError:  &exec.ExitError{},
			originError:    &exec.ExitError{},
		}

		detector := newDetectorWithExecutor(fake)
		info, err := detector.DetectRepository(context.Background(), testDir)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if info.RootDir != testDir {
			t.Errorf("Expected root dir %s, got %s", testDir, info.RootDir)
		}

		if info.RemoteURL != "" {
			t.Errorf("Expected empty remote URL, got %s", info.RemoteURL)
		}

		if info.RelativePath != "" {
			t.Errorf("Expected empty relative path, got %s", info.RelativePath)
		}
	})

	t.Run("returns error when not a git repository", func(t *testing.T) {
		t.Parallel()

		testDir := t.TempDir()

		fake := &fakeExecutor{
			revParseError: &exitError128{},
		}

		detector := newDetectorWithExecutor(fake)
		_, err := detector.DetectRepository(context.Background(), testDir)

		if !errors.Is(err, ErrNotGitRepository) {
			t.Errorf("Expected ErrNotGitRepository, got %v", err)
		}
	})

	t.Run("returns error on git command failure", func(t *testing.T) {
		t.Parallel()

		testDir := t.TempDir()

		fake := &fakeExecutor{
			revParseError: errors.New("git command failed"),
		}

		detector := newDetectorWithExecutor(fake)
		_, err := detector.DetectRepository(context.Background(), testDir)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if errors.Is(err, ErrNotGitRepository) {
			t.Error("Should not be ErrNotGitRepository")
		}
	})
}

// exitError128 is a test helper that simulates git exit code 128
type exitError128 struct{}

func (e *exitError128) ExitCode() int {
	return 128
}

func (e *exitError128) Error() string {
	return "exit status 128"
}
