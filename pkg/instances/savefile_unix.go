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

//go:build !windows

package instances

import (
	"os"
	"path/filepath"
)

// writeStorageFile writes data to path atomically using a temp file + rename.
// On POSIX systems rename(2) is atomic: readers always see either the old file
// or the new file, never a partial write, and the rename succeeds even if
// another process currently has the destination open.
func writeStorageFile(path string, data []byte) error {
	tmpFile, err := os.CreateTemp(filepath.Dir(path), ".instances-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // no-op once Rename succeeds

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}
