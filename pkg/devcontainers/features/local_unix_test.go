//go:build !windows

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
	"syscall"
	"testing"

	"github.com/openkaiden/kdn/pkg/devcontainers/features"
)

// TestLocalFeature_CopyFile_UnreadableSource covers the os.Open error path in
// copyFile by including a file with no read permission in the feature directory.
// Skipped when running as root since permissions have no effect.
func TestLocalFeature_CopyFile_UnreadableSource(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("running as root, file permissions have no effect")
	}

	workspaceDir := t.TempDir()
	featureDir := filepath.Join(workspaceDir, "my-feature")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFeatureJSON(t, featureDir, map[string]interface{}{})

	// Write a file that cannot be read (0000 permissions).
	secretFile := filepath.Join(featureDir, "z-secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret"), 0000); err != nil {
		t.Fatalf("WriteFile: %v", err)
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
		t.Error("expected error for unreadable source file, got nil")
	}
}

// TestLocalFeature_CopyFile_ReadOnlyDest covers the os.OpenFile error path in
// copyFile when the destination directory is read-only.
// Skipped when running as root since permissions have no effect.
func TestLocalFeature_CopyFile_ReadOnlyDest(t *testing.T) {
	t.Parallel()

	if os.Getuid() == 0 {
		t.Skip("running as root, file permissions have no effect")
	}

	workspaceDir := t.TempDir()
	makeLocalFeatureDir(t, workspaceDir, "my-feature", map[string]interface{}{})

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{"./my-feature": nil},
		workspaceDir,
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	destDir := t.TempDir()
	if err := os.Chmod(destDir, 0555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(destDir, 0755) })

	_, err = feats[0].Download(context.Background(), destDir)
	if err == nil {
		t.Error("expected error for read-only destDir, got nil")
	}
}

// TestLocalFeature_CopyDir_NonRegularFile covers the non-regular-file branch in
// copyDir using a named pipe, which is neither a directory nor a symlink nor a
// regular file.
func TestLocalFeature_CopyDir_NonRegularFile(t *testing.T) {
	t.Parallel()

	workspaceDir := t.TempDir()
	featureDir := filepath.Join(workspaceDir, "my-feature")
	if err := os.MkdirAll(featureDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	writeFeatureJSON(t, featureDir, map[string]interface{}{})

	// Named pipe — not a regular file, not a symlink, not a directory.
	// "z-pipe" sorts after devcontainer-feature.json so WalkDir visits it last.
	pipePath := filepath.Join(featureDir, "z-pipe")
	if err := syscall.Mkfifo(pipePath, 0600); err != nil {
		t.Skipf("cannot create named pipe: %v", err)
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
		t.Fatal("expected error for non-regular file in feature dir, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported non-regular file") {
		t.Errorf("error = %q, want to contain 'unsupported non-regular file'", err.Error())
	}
}
