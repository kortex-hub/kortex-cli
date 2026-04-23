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

package features_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkaiden/kdn/pkg/devcontainers/features"
)

// TestLocalFeature_Download_SrcDirNotFound covers the copyDir walk-error path
// (callback receives err != nil) and the "copying feature from" error wrapping
// in Download when the source directory does not exist.
func TestLocalFeature_Download_SrcDirNotFound(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{"./nonexistent-feature": nil},
		workspaceDir,
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	_, err = feats[0].Download(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for non-existent source directory, got nil")
	}
	if !strings.Contains(err.Error(), "copying feature from") {
		t.Errorf("error = %q, want to contain 'copying feature from'", err.Error())
	}
}

// TestLocalFeature_Download_DestIsFile covers the os.MkdirAll error path in
// Download when destDir already exists as a regular file.
func TestLocalFeature_Download_DestIsFile(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	makeLocalFeatureDir(t, workspaceDir, "my-feature", map[string]interface{}{})

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{"./my-feature": nil},
		workspaceDir,
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	destFile := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(destFile, []byte("file"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err = feats[0].Download(context.Background(), destFile)
	if err == nil {
		t.Error("expected error when destDir is a regular file, got nil")
	}
}

// TestLocalFeature_CopyDir_Symlink covers the symlink detection branch in copyDir.
func TestLocalFeature_CopyDir_Symlink(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	featureDir := filepath.Join(workspaceDir, "my-feature")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFeatureJSON(t, featureDir, map[string]interface{}{})

	targetFile := filepath.Join(workspaceDir, "symlink-target")
	if err := os.WriteFile(targetFile, []byte("target"), 0644); err != nil {
		t.Fatalf("WriteFile target: %v", err)
	}
	if err := os.Symlink(targetFile, filepath.Join(featureDir, "z-link")); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{"./my-feature": nil},
		workspaceDir,
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	_, err = feats[0].Download(context.Background(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for symlink in feature dir, got nil")
	}
	if !strings.Contains(err.Error(), "symlinks") {
		t.Errorf("error = %q, want to contain 'symlinks'", err.Error())
	}
}

// TestLocalFeature_Download_WithSubdirectory verifies that copyDir correctly
// creates subdirectories and copies files within them.
func TestLocalFeature_Download_WithSubdirectory(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	featureDir := filepath.Join(workspaceDir, "my-feature")
	subDir := filepath.Join(featureDir, "scripts")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFeatureJSON(t, featureDir, map[string]interface{}{
		"containerEnv": map[string]string{"FOO": "bar"},
	})
	if err := os.WriteFile(filepath.Join(subDir, "install.sh"), []byte("#!/bin/sh\necho hi"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{"./my-feature": nil},
		workspaceDir,
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	destDir := t.TempDir()
	meta, err := feats[0].Download(context.Background(), destDir)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got := meta.ContainerEnv()["FOO"]; got != "bar" {
		t.Errorf("ContainerEnv FOO = %q, want %q", got, "bar")
	}
	if _, err := os.Stat(filepath.Join(destDir, "scripts", "install.sh")); err != nil {
		t.Errorf("scripts/install.sh not found: %v", err)
	}
}

// TestLocalFeature_ParentRelativePath tests that a feature with a "../" prefix
// is resolved correctly relative to the workspace config directory.
func TestLocalFeature_ParentRelativePath(t *testing.T) {
	t.Parallel()

	parentDir := t.TempDir()
	workspaceDir := filepath.Join(parentDir, "subdir")
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		t.Fatalf("MkdirAll workspaceDir: %v", err)
	}
	featureDir := filepath.Join(parentDir, "my-feature")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("MkdirAll featureDir: %v", err)
	}
	writeFeatureJSON(t, featureDir, map[string]interface{}{})

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{"../my-feature": nil},
		workspaceDir,
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}
	if len(feats) != 1 {
		t.Fatalf("len(feats) = %d, want 1", len(feats))
	}
	if feats[0].ID() != "../my-feature" {
		t.Errorf("ID = %q, want %q", feats[0].ID(), "../my-feature")
	}

	_, err = feats[0].Download(context.Background(), t.TempDir())
	if err != nil {
		t.Errorf("Download: %v", err)
	}
}
