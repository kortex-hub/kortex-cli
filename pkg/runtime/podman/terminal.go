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

package podman

import (
	"context"
	"fmt"

	"github.com/kortex-hub/kortex-cli/pkg/runtime"
)

// Ensure podmanRuntime implements runtime.Terminal at compile time.
var _ runtime.Terminal = (*podmanRuntime)(nil)

// Terminal starts an interactive terminal session inside a running instance.
func (p *podmanRuntime) Terminal(ctx context.Context, instanceID string, command []string) error {
	if instanceID == "" {
		return fmt.Errorf("%w: instance ID is required", runtime.ErrInvalidParams)
	}
	if len(command) == 0 {
		return fmt.Errorf("%w: command is required", runtime.ErrInvalidParams)
	}

	// Build podman exec -it <container> <command...>
	args := []string{"exec", "-it", instanceID}
	args = append(args, command...)

	return p.executor.RunInteractive(ctx, args...)
}
