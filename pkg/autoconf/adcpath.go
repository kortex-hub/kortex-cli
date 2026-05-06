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

package autoconf

import "path/filepath"

// adcDetectPath returns the absolute host path to the gcloud ADC file used
// for existence checking.
func adcDetectPath(homeDir, _ string) string {
	return filepath.Join(homeDir, ".config", "gcloud", "application_default_credentials.json")
}

// adcConfigHostPath returns the host path to write in a workspace mount entry.
// On Linux/macOS the ADC file lives under $HOME/.config/gcloud/, so the
// $HOME-variable form matches the actual file location.
func adcConfigHostPath(_, _ string) string {
	return ADCContainerPath
}
