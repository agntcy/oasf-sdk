// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoder

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestGetRecordModuleData(t *testing.T) {
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

	found, data := GetRecordModuleData(recordStruct, "agentskills")
	if !found {
		t.Fatalf("Expected agentskills module data to be found")
	}

	if data == nil {
		t.Fatalf("Expected agentskills module data to be non-nil")
	}

	if _, ok := data.GetFields()["skill_file"]; !ok {
		t.Fatalf("Expected skill_file field in agentskills data")
	}
}

func TestGetRecordModuleDataMissing(t *testing.T) {
	recordMap := map[string]any{
		"schema_version": "1.0.0",
		"modules":        []any{},
	}

	recordStruct, err := structpb.NewStruct(recordMap)
	if err != nil {
		t.Fatalf("Failed to build record struct: %v", err)
	}

	found, data := GetRecordModuleData(recordStruct, "agentskills")
	if found {
		t.Fatalf("Expected agentskills module data to be missing")
	}

	if data != nil {
		t.Fatalf("Expected agentskills module data to be nil")
	}
}
