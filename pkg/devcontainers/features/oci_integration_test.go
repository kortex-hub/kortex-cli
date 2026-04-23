//go:build integration

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
	"sync"
	"testing"
	"time"

	"github.com/openkaiden/kdn/pkg/devcontainers/features"
)

const (
	nodeFeatureID        = "ghcr.io/devcontainers/features/node:1"
	awsCliFeatureID      = "ghcr.io/devcontainers/features/aws-cli:1"
	commonUtilsFeatureID = "ghcr.io/devcontainers/features/common-utils:2"
)

func TestIntegration_OCIFeature_DownloadNode(t *testing.T) {
	t.Parallel()

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{nodeFeatureID: nil},
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}
	if len(feats) != 1 {
		t.Fatalf("expected 1 feature, got %d", len(feats))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	destDir := t.TempDir()
	meta, err := feats[0].Download(ctx, destDir)
	if err != nil {
		t.Fatalf("Download(%q): %v", nodeFeatureID, err)
	}

	// install.sh must be present — it is the entry point for every feature.
	if _, err := os.Stat(filepath.Join(destDir, "install.sh")); err != nil {
		t.Errorf("install.sh not found in extracted feature: %v", err)
	}

	if meta == nil {
		t.Fatal("Download returned nil metadata")
	}

	// ContainerEnv should include NVM_DIR, which the node feature always sets.
	env := meta.ContainerEnv()
	if _, ok := env["NVM_DIR"]; !ok {
		t.Errorf("ContainerEnv missing NVM_DIR; got %v", env)
	}

	// The node feature always declares a "version" string option with a default.
	opts := meta.Options()
	if opts == nil {
		t.Fatal("Options() returned nil")
	}
	merged, err := opts.Merge(nil)
	if err != nil {
		t.Fatalf("Merge(nil): %v", err)
	}
	if _, ok := merged["VERSION"]; !ok {
		t.Errorf("merged options missing VERSION key; got %v", merged)
	}
}

func TestIntegration_OCIFeature_MergeUserOptions(t *testing.T) {
	t.Parallel()

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{nodeFeatureID: nil},
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	meta, err := feats[0].Download(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("Download: %v", err)
	}

	// Override the version option; the key must be normalised to VERSION.
	merged, err := meta.Options().Merge(map[string]interface{}{
		"version": "20",
	})
	if err != nil {
		t.Fatalf("Merge with version=20: %v", err)
	}
	if got := merged["VERSION"]; got != "20" {
		t.Errorf("VERSION = %q, want %q", got, "20")
	}
}

func TestIntegration_OCIFeature_InvalidOptionRejected(t *testing.T) {
	t.Parallel()

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{nodeFeatureID: nil},
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	meta, err := feats[0].Download(ctx, t.TempDir())
	if err != nil {
		t.Fatalf("Download: %v", err)
	}

	// Supplying an integer for a string option must return an error.
	_, err = meta.Options().Merge(map[string]interface{}{
		"version": 9000,
	})
	if err == nil {
		t.Error("expected error for wrong type (int for string option), got nil")
	}
}

// TestIntegration_OCIFeature_OrderAWSCLIAfterCommonUtils verifies that Order
// places common-utils:2 before aws-cli:1.
//
// aws-cli:1 declares installsAfter: ["ghcr.io/devcontainers/features/common-utils"]
// (versionless, per spec). Without version-aware matching this dependency would
// be missed because the registered ID carries a ":2" tag — and the naive
// alphabetical order ("aws-cli" < "common-utils") would produce the wrong result.
// This test therefore exercises both the version-stripped ID matching in Order
// and the network download path.
func TestIntegration_OCIFeature_OrderAWSCLIAfterCommonUtils(t *testing.T) {
	t.Parallel()

	feats, _, err := features.FromMap(
		map[string]map[string]interface{}{
			awsCliFeatureID:      nil,
			commonUtilsFeatureID: nil,
		},
		t.TempDir(),
	)
	if err != nil {
		t.Fatalf("FromMap: %v", err)
	}
	if len(feats) != 2 {
		t.Fatalf("expected 2 features, got %d", len(feats))
	}

	// Verify that FromMap pre-sorts by ID: aws-cli < common-utils alphabetically.
	if feats[0].ID() != awsCliFeatureID || feats[1].ID() != commonUtilsFeatureID {
		t.Fatalf("FromMap sort: got [%s, %s], want [%s, %s]",
			feats[0].ID(), feats[1].ID(), awsCliFeatureID, commonUtilsFeatureID)
	}

	// Download both features in parallel.
	type downloadResult struct {
		id   string
		meta features.FeatureMetadata
		err  error
	}
	results := make([]downloadResult, len(feats))
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var wg sync.WaitGroup
	for i, feat := range feats {
		wg.Add(1)
		go func(i int, feat features.Feature) {
			defer wg.Done()
			meta, err := feat.Download(ctx, t.TempDir())
			results[i] = downloadResult{id: feat.ID(), meta: meta, err: err}
		}(i, feat)
	}
	wg.Wait()

	metadata := make(map[string]features.FeatureMetadata, len(feats))
	for _, r := range results {
		if r.err != nil {
			t.Fatalf("Download(%q): %v", r.id, r.err)
		}
		metadata[r.id] = r.meta
	}

	t.Logf("aws-cli installsAfter:      %v", metadata[awsCliFeatureID].InstallsAfter())
	t.Logf("common-utils installsAfter: %v", metadata[commonUtilsFeatureID].InstallsAfter())

	ordered, err := features.Order(feats, metadata)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	if len(ordered) != 2 {
		t.Fatalf("Order returned %d features, want 2", len(ordered))
	}

	// common-utils must come first because aws-cli installsAfter it.
	// This is the opposite of alphabetical order, proving the dependency was detected.
	if ordered[0].ID() != commonUtilsFeatureID || ordered[1].ID() != awsCliFeatureID {
		t.Errorf("Order: got [%s, %s], want [%s, %s]",
			ordered[0].ID(), ordered[1].ID(), commonUtilsFeatureID, awsCliFeatureID)
	}
}
