// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package record_test

import (
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/record"
	"google.golang.org/protobuf/types/known/structpb"
)

func makeRecord(t *testing.T, modules []any) *structpb.Struct {
	t.Helper()

	s, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules":        modules,
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	return s
}

func TestGetModuleData_Found(t *testing.T) {
	rec := makeRecord(t, []any{
		map[string]any{
			"name": "integration/mcp",
			"data": map[string]any{"key": "value"},
		},
	})

	found, data := record.GetModuleData(rec, "integration/mcp")
	if !found {
		t.Fatal("expected module to be found")
	}

	if data == nil {
		t.Fatal("expected non-nil module data")
	}

	if data.GetFields()["key"].GetStringValue() != "value" {
		t.Errorf("unexpected data field value: %v", data.GetFields()["key"])
	}
}

func TestGetModuleData_NotFound(t *testing.T) {
	rec := makeRecord(t, []any{
		map[string]any{
			"name": "integration/a2a",
			"data": map[string]any{},
		},
	})

	found, data := record.GetModuleData(rec, "integration/mcp")
	if found {
		t.Error("expected module not to be found")
	}

	if data != nil {
		t.Error("expected nil data when module not found")
	}
}

func TestGetModuleData_EmptyModules(t *testing.T) {
	rec := makeRecord(t, []any{})

	found, data := record.GetModuleData(rec, "integration/mcp")
	if found {
		t.Error("expected no match in empty modules list")
	}

	if data != nil {
		t.Error("expected nil data for empty modules list")
	}
}

func TestGetModuleData_NilRecord(t *testing.T) {
	found, data := record.GetModuleData(nil, "integration/mcp")
	if found {
		t.Error("expected false for nil record")
	}

	if data != nil {
		t.Error("expected nil data for nil record")
	}
}

func TestGetModuleData_MultipleModules(t *testing.T) {
	rec := makeRecord(t, []any{
		map[string]any{
			"name": "integration/a2a",
			"data": map[string]any{"a2a_field": "a2a_value"},
		},
		map[string]any{
			"name": "integration/mcp",
			"data": map[string]any{"mcp_field": "mcp_value"},
		},
	})

	found, data := record.GetModuleData(rec, "integration/mcp")
	if !found {
		t.Fatal("expected mcp module to be found")
	}

	if data.GetFields()["mcp_field"].GetStringValue() != "mcp_value" {
		t.Error("returned wrong module data")
	}
}

func TestGetModuleData_NoModulesField(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{"schema_version": "1.0.0"})
	if err != nil {
		t.Fatalf("failed to build struct: %v", err)
	}

	found, data := record.GetModuleData(s, "integration/mcp")
	if found {
		t.Error("expected false when modules field is absent")
	}

	if data != nil {
		t.Error("expected nil data when modules field is absent")
	}
}
