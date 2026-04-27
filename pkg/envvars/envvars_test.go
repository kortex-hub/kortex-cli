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

package envvars_test

import (
	"testing"

	"github.com/openkaiden/kdn/pkg/envvars"
)

func TestIsTruthy(t *testing.T) {
	// Cannot use t.Parallel() on the parent because subtests use t.Setenv.

	const key = "TEST_ISTRUTHY_ENV"

	truthy := []string{"1", "true", "True", "TRUE", "yes", "Yes", "YES"}
	for _, v := range truthy {
		t.Run("truthy_"+v, func(t *testing.T) {
			t.Setenv(key, v)
			if !envvars.IsTruthy(key) {
				t.Errorf("expected IsTruthy(%q) to be true when %s=%q", key, key, v)
			}
		})
	}

	falsy := []string{"0", "false", "no", "", "random"}
	for _, v := range falsy {
		t.Run("falsy_"+v, func(t *testing.T) {
			t.Setenv(key, v)
			if envvars.IsTruthy(key) {
				t.Errorf("expected IsTruthy(%q) to be false when %s=%q", key, key, v)
			}
		})
	}
}
