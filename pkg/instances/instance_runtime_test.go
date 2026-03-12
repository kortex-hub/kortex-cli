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

package instances

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestInstanceData_SerializeWithRuntime(t *testing.T) {
	t.Parallel()

	data := InstanceData{
		ID:   "test-id",
		Name: "test-instance",
		Paths: InstancePaths{
			Source:        "/path/to/source",
			Configuration: "/path/to/config",
		},
		Runtime: RuntimeData{
			Type:       "fake",
			InstanceID: "fake-001",
			State:      "running",
			Info: map[string]string{
				"created_at": "2026-03-11T10:00:00Z",
			},
		},
	}

	// Serialize to JSON
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal InstanceData: %v", err)
	}

	// Deserialize back
	var deserialized InstanceData
	err = json.Unmarshal(jsonBytes, &deserialized)
	if err != nil {
		t.Fatalf("Failed to unmarshal InstanceData: %v", err)
	}

	// Verify all fields
	if deserialized.ID != data.ID {
		t.Errorf("ID = %v, want %v", deserialized.ID, data.ID)
	}
	if deserialized.Name != data.Name {
		t.Errorf("Name = %v, want %v", deserialized.Name, data.Name)
	}
	if deserialized.Paths.Source != data.Paths.Source {
		t.Errorf("Paths.Source = %v, want %v", deserialized.Paths.Source, data.Paths.Source)
	}
	if deserialized.Paths.Configuration != data.Paths.Configuration {
		t.Errorf("Paths.Configuration = %v, want %v", deserialized.Paths.Configuration, data.Paths.Configuration)
	}
	if deserialized.Runtime.Type != data.Runtime.Type {
		t.Errorf("Runtime.Type = %v, want %v", deserialized.Runtime.Type, data.Runtime.Type)
	}
	if deserialized.Runtime.InstanceID != data.Runtime.InstanceID {
		t.Errorf("Runtime.InstanceID = %v, want %v", deserialized.Runtime.InstanceID, data.Runtime.InstanceID)
	}
	if deserialized.Runtime.State != data.Runtime.State {
		t.Errorf("Runtime.State = %v, want %v", deserialized.Runtime.State, data.Runtime.State)
	}
	if deserialized.Runtime.Info["created_at"] != data.Runtime.Info["created_at"] {
		t.Errorf("Runtime.Info[created_at] = %v, want %v", deserialized.Runtime.Info["created_at"], data.Runtime.Info["created_at"])
	}
}

func TestInstance_GetRuntimeType(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	data := InstanceData{
		ID:   "test-id",
		Name: "test-instance",
		Paths: InstancePaths{
			Source:        filepath.Join(tmpDir, "source"),
			Configuration: filepath.Join(tmpDir, "config"),
		},
		Runtime: RuntimeData{
			Type:       "fake",
			InstanceID: "fake-001",
			State:      "running",
		},
	}

	inst, err := NewInstanceFromData(data)
	if err != nil {
		t.Fatalf("NewInstanceFromData() failed: %v", err)
	}

	if inst.GetRuntimeType() != "fake" {
		t.Errorf("GetRuntimeType() = %v, want 'fake'", inst.GetRuntimeType())
	}
}

func TestInstance_GetRuntimeData(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	runtimeData := RuntimeData{
		Type:       "fake",
		InstanceID: "fake-001",
		State:      "running",
		Info: map[string]string{
			"created_at": "2026-03-11T10:00:00Z",
			"started_at": "2026-03-11T10:01:00Z",
		},
	}

	data := InstanceData{
		ID:   "test-id",
		Name: "test-instance",
		Paths: InstancePaths{
			Source:        filepath.Join(tmpDir, "source"),
			Configuration: filepath.Join(tmpDir, "config"),
		},
		Runtime: runtimeData,
	}

	inst, err := NewInstanceFromData(data)
	if err != nil {
		t.Fatalf("NewInstanceFromData() failed: %v", err)
	}

	info := inst.GetRuntimeData()

	if info.Type != runtimeData.Type {
		t.Errorf("GetRuntimeData().Type = %v, want %v", info.Type, runtimeData.Type)
	}
	if info.InstanceID != runtimeData.InstanceID {
		t.Errorf("GetRuntimeData().InstanceID = %v, want %v", info.InstanceID, runtimeData.InstanceID)
	}
	if info.State != runtimeData.State {
		t.Errorf("GetRuntimeData().State = %v, want %v", info.State, runtimeData.State)
	}
	if info.Info["created_at"] != runtimeData.Info["created_at"] {
		t.Errorf("GetRuntimeData().Info[created_at] = %v, want %v", info.Info["created_at"], runtimeData.Info["created_at"])
	}
	if info.Info["started_at"] != runtimeData.Info["started_at"] {
		t.Errorf("GetRuntimeData().Info[started_at] = %v, want %v", info.Info["started_at"], runtimeData.Info["started_at"])
	}
}

func TestInstance_DumpIncludesRuntime(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	runtimeData := RuntimeData{
		Type:       "fake",
		InstanceID: "fake-001",
		State:      "running",
		Info: map[string]string{
			"key": "value",
		},
	}

	data := InstanceData{
		ID:   "test-id",
		Name: "test-instance",
		Paths: InstancePaths{
			Source:        filepath.Join(tmpDir, "source"),
			Configuration: filepath.Join(tmpDir, "config"),
		},
		Runtime: runtimeData,
	}

	inst, err := NewInstanceFromData(data)
	if err != nil {
		t.Fatalf("NewInstanceFromData() failed: %v", err)
	}

	dumped := inst.Dump()

	if dumped.Runtime.Type != runtimeData.Type {
		t.Errorf("Dump().Runtime.Type = %v, want %v", dumped.Runtime.Type, runtimeData.Type)
	}
	if dumped.Runtime.InstanceID != runtimeData.InstanceID {
		t.Errorf("Dump().Runtime.InstanceID = %v, want %v", dumped.Runtime.InstanceID, runtimeData.InstanceID)
	}
	if dumped.Runtime.State != runtimeData.State {
		t.Errorf("Dump().Runtime.State = %v, want %v", dumped.Runtime.State, runtimeData.State)
	}
	if dumped.Runtime.Info["key"] != runtimeData.Info["key"] {
		t.Errorf("Dump().Runtime.Info[key] = %v, want %v", dumped.Runtime.Info["key"], runtimeData.Info["key"])
	}
}
