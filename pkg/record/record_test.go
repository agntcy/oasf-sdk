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

func TestGetModule_Found(t *testing.T) {
	rec := makeRecord(t, []any{
		map[string]any{
			"name": "integration/mcp",
			"data": map[string]any{"key": "value"},
		},
	})

	found, mod := record.GetModule(rec, "integration/mcp")
	if !found {
		t.Fatal("expected module to be found")
	}

	if mod == nil {
		t.Fatal("expected non-nil module")
	}

	data := mod.GetFields()["data"].GetStructValue()
	if data == nil {
		t.Fatal("expected non-nil data in module")
	}

	if data.GetFields()["key"].GetStringValue() != "value" {
		t.Errorf("unexpected data field value: %v", data.GetFields()["key"])
	}
}

func TestGetModule_ReturnsWholeModule(t *testing.T) {
	rec := makeRecord(t, []any{
		map[string]any{
			"name":     "integration/mcp",
			"id":       float64(42),
			"data":     map[string]any{"key": "value"},
			"artifact": map[string]any{"media_type": "application/json"},
		},
	})

	found, mod := record.GetModule(rec, "integration/mcp")
	if !found {
		t.Fatal("expected module to be found")
	}

	if mod.GetFields()["name"].GetStringValue() != "integration/mcp" {
		t.Error("expected name field in module")
	}

	if mod.GetFields()["id"].GetNumberValue() != 42 {
		t.Error("expected id field in module")
	}

	if mod.GetFields()["artifact"].GetStructValue() == nil {
		t.Error("expected artifact field in module")
	}
}

func TestGetModule_NotFound(t *testing.T) {
	rec := makeRecord(t, []any{
		map[string]any{
			"name": "integration/a2a",
			"data": map[string]any{},
		},
	})

	found, mod := record.GetModule(rec, "integration/mcp")
	if found {
		t.Error("expected module not to be found")
	}

	if mod != nil {
		t.Error("expected nil module when not found")
	}
}

func TestGetModule_EmptyModules(t *testing.T) {
	rec := makeRecord(t, []any{})

	found, mod := record.GetModule(rec, "integration/mcp")
	if found {
		t.Error("expected no match in empty modules list")
	}

	if mod != nil {
		t.Error("expected nil module for empty modules list")
	}
}

func TestGetModule_NilRecord(t *testing.T) {
	found, mod := record.GetModule(nil, "integration/mcp")
	if found {
		t.Error("expected false for nil record")
	}

	if mod != nil {
		t.Error("expected nil module for nil record")
	}
}

func TestGetModule_MultipleModules(t *testing.T) {
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

	found, mod := record.GetModule(rec, "integration/mcp")
	if !found {
		t.Fatal("expected mcp module to be found")
	}

	data := mod.GetFields()["data"].GetStructValue()
	if data.GetFields()["mcp_field"].GetStringValue() != "mcp_value" {
		t.Error("returned wrong module")
	}
}

func TestGetModule_NoModulesField(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{"schema_version": "1.0.0"})
	if err != nil {
		t.Fatalf("failed to build struct: %v", err)
	}

	found, mod := record.GetModule(s, "integration/mcp")
	if found {
		t.Error("expected false when modules field is absent")
	}

	if mod != nil {
		t.Error("expected nil module when modules field is absent")
	}
}
