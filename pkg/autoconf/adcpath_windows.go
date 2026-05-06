//go:build windows

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

package autoconf

import (
	"path"
	"path/filepath"
	"strings"
)

// adcDetectPath returns the absolute host path to the gcloud ADC file used
// for existence checking. On Windows gcloud stores credentials under %APPDATA%
// rather than under the user home directory.
func adcDetectPath(_, appDataDir string) string {
	if appDataDir == "" {
		return ""
	}
	return filepath.Join(appDataDir, "gcloud", "application_default_credentials.json")
}

// adcConfigHostPath returns the host path to write in a workspace mount entry.
// On Windows %APPDATA% is typically a subdirectory of the user home (e.g.
// $HOME\AppData\Roaming), so the path is expressed as $HOME/<rel>/gcloud/...
// so that it starts with $HOME as required by the mount config format.
// Returns "" if %APPDATA% is unset or is not located under homeDir.
func adcConfigHostPath(homeDir, appDataDir string) string {
	if appDataDir == "" {
		return ""
	}
	rel, err := filepath.Rel(homeDir, appDataDir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return ""
	}
	return path.Join("$HOME", filepath.ToSlash(rel), "gcloud", "application_default_credentials.json")
}
