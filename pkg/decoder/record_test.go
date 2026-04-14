// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoder

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetRecordModule(t *testing.T) {
	recordMap := map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": "agentskills",
				"data": map[string]any{
					"skill_file": "SKILL.md",
					"skill_manifest": map[string]any{
						"name": "example-skill",
					},
				},
			},
		},
	}

	recordStruct, err := structpb.NewStruct(recordMap)
	if err != nil {
		t.Fatalf("Failed to build record struct: %v", err)
	}

	found, mod := GetRecordModule(recordStruct, "agentskills")
	if !found {
		t.Fatalf("Expected agentskills module to be found")
	}

	if mod == nil {
		t.Fatalf("Expected agentskills module to be non-nil")
	}

	data := mod.GetFields()["data"].GetStructValue()
	if data == nil {
		t.Fatalf("Expected non-nil data in module")
	}

	if _, ok := data.GetFields()["skill_file"]; !ok {
		t.Fatalf("Expected skill_file field in agentskills data")
	}
}

func TestGetRecordModuleMissing(t *testing.T) {
	recordMap := map[string]any{
		"schema_version": "1.0.0",
		"modules":        []any{},
	}

	recordStruct, err := structpb.NewStruct(recordMap)
	if err != nil {
		t.Fatalf("Failed to build record struct: %v", err)
	}

	found, mod := GetRecordModule(recordStruct, "agentskills")
	if found {
		t.Fatalf("Expected agentskills module to be missing")
	}

	if mod != nil {
		t.Fatalf("Expected nil module when not found")
	}
}
