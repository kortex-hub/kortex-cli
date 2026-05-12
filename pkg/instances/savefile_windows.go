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

//go:build windows

package instances

import "os"

// writeStorageFile writes data to path directly.
// On Windows, MoveFileExW (used by os.Rename) fails with "Access is denied"
// if another process has the destination file open without FILE_SHARE_DELETE,
// which os.ReadFile does not request. Since withStorageLock already holds an
// exclusive LockFileEx preventing concurrent writers, a plain WriteFile is
// safe: only one writer runs at a time, and the write is fast enough that
// readers calling loadInstances concurrently are unlikely to observe a partial
// file.
func writeStorageFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}
