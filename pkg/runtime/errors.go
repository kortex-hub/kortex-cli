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

package runtime

import "errors"

var (
	// ErrRuntimeNotFound is returned when a requested runtime type is not registered.
	ErrRuntimeNotFound = errors.New("runtime not found")

	// ErrInstanceNotFound is returned when a requested runtime instance does not exist.
	ErrInstanceNotFound = errors.New("runtime instance not found")

	// ErrRuntimeUnavailable is returned when a runtime exists but cannot be used
	// (e.g., podman binary not found, docker daemon not running).
	ErrRuntimeUnavailable = errors.New("runtime unavailable")

	// ErrInvalidParams is returned when create parameters are invalid or incomplete.
	ErrInvalidParams = errors.New("invalid runtime parameters")
)
