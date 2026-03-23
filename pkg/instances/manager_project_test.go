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

package instances

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kortex-hub/kortex-cli/pkg/git"
)

func TestManager_detectProject(t *testing.T) {
	t.Parallel()

	t.Run("returns source directory for non-git directory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		sourceDir := filepath.Join(tmpDir, "workspace")

		// Create fake git detector that returns ErrNotGitRepository
		gitDetector := newFakeGitDetector()

		m, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), newTestRegistry(tmpDir), gitDetector)
		mgr := m.(*manager)

		result := mgr.detectProject(context.Background(), sourceDir)

		if result != sourceDir {
			t.Errorf("detectProject() = %v, want %v", result, sourceDir)
		}
	})

	t.Run("returns remote URL with trailing slash for git repository at root", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		repoRoot := filepath.Join(tmpDir, "repo")

		// Create fake git detector that returns repository info
		gitDetector := newFakeGitDetectorWithRepo(
			repoRoot,
			"https://github.com/user/repo",
			"", // at root, relative path is empty
		)

		m, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), newTestRegistry(tmpDir), gitDetector)
		mgr := m.(*manager)

		result := mgr.detectProject(context.Background(), repoRoot)

		expected := "https://github.com/user/repo/"
		if result != expected {
			t.Errorf("detectProject() = %v, want %v", result, expected)
		}
	})

	t.Run("returns remote URL with subdirectory path for git repository in subdirectory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		repoRoot := filepath.Join(tmpDir, "repo")
		subDir := filepath.Join(repoRoot, "pkg", "git")

		// Create fake git detector that returns repository info with relative path
		gitDetector := newFakeGitDetectorWithRepo(
			repoRoot,
			"https://github.com/user/repo",
			filepath.Join("pkg", "git"),
		)

		m, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), newTestRegistry(tmpDir), gitDetector)
		mgr := m.(*manager)

		result := mgr.detectProject(context.Background(), subDir)

		expected := "https://github.com/user/repo/pkg/git"
		if result != expected {
			t.Errorf("detectProject() = %v, want %v", result, expected)
		}
	})

	t.Run("returns local repository root for git repository without remote", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		repoRoot := filepath.Join(tmpDir, "local-repo")

		// Create fake git detector that returns repository info without remote URL
		gitDetector := newFakeGitDetectorWithRepo(
			repoRoot,
			"", // no remote URL
			"", // at root
		)

		m, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), newTestRegistry(tmpDir), gitDetector)
		mgr := m.(*manager)

		result := mgr.detectProject(context.Background(), repoRoot)

		if result != repoRoot {
			t.Errorf("detectProject() = %v, want %v", result, repoRoot)
		}
	})

	t.Run("returns local repository root with relative path for git repository without remote in subdirectory", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		repoRoot := filepath.Join(tmpDir, "local-repo")
		subDir := filepath.Join(repoRoot, "pkg", "utils")

		// Create fake git detector that returns repository info without remote URL
		gitDetector := newFakeGitDetectorWithRepo(
			repoRoot,
			"", // no remote URL
			filepath.Join("pkg", "utils"),
		)

		m, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), newTestRegistry(tmpDir), gitDetector)
		mgr := m.(*manager)

		result := mgr.detectProject(context.Background(), subDir)

		expected := filepath.Join(repoRoot, "pkg", "utils")
		if result != expected {
			t.Errorf("detectProject() = %v, want %v", result, expected)
		}
	})

	t.Run("handles remote URL that already has trailing slash", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		repoRoot := filepath.Join(tmpDir, "repo")

		// Create fake git detector with URL that already has trailing slash
		gitDetector := newFakeGitDetectorWithRepo(
			repoRoot,
			"https://github.com/user/repo/",
			"",
		)

		m, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), newTestRegistry(tmpDir), gitDetector)
		mgr := m.(*manager)

		result := mgr.detectProject(context.Background(), repoRoot)

		expected := "https://github.com/user/repo/"
		if result != expected {
			t.Errorf("detectProject() = %v, want %v", result, expected)
		}
	})

	t.Run("preserves URL scheme and slashes when appending relative path", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		repoRoot := filepath.Join(tmpDir, "repo")

		// Create fake git detector with HTTPS URL
		gitDetector := newFakeGitDetectorWithRepo(
			repoRoot,
			"https://github.com/upstream/repo",
			filepath.Join("pkg", "cmd"),
		)

		m, _ := newManagerWithFactory(tmpDir, fakeInstanceFactory, newFakeGenerator(), newTestRegistry(tmpDir), gitDetector)
		mgr := m.(*manager)

		result := mgr.detectProject(context.Background(), repoRoot)

		// Should preserve https:// in the URL
		expected := "https://github.com/upstream/repo/pkg/cmd"
		if result != expected {
			t.Errorf("detectProject() = %v, want %v", result, expected)
		}
	})
}

// newFakeGitDetectorWithRepo creates a fake git detector that returns repository info
func newFakeGitDetectorWithRepo(rootDir, remoteURL, relativePath string) *fakeGitDetector {
	return &fakeGitDetector{
		repoInfo: &git.RepositoryInfo{
			RootDir:      rootDir,
			RemoteURL:    remoteURL,
			RelativePath: relativePath,
		},
	}
}
