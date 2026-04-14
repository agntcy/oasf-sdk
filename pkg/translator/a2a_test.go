// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator_test

import (
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/types/known/structpb"
)

// --- A2AToRecord ---

func TestA2AToRecord_WrappedFormat(t *testing.T) {
	input, err := structpb.NewStruct(map[string]any{
		"a2aCard": map[string]any{
			"name":        "my-agent",
			"description": "does things",
			"version":     "2.0.0",
			"provider": map[string]any{
				"organization": "ACME Corp",
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build input: %v", err)
	}

	record, err := translator.A2AToRecord(input)
	if err != nil {
		t.Fatalf("A2AToRecord() error: %v", err)
	}

	fields := record.GetFields()

	if fields["name"].GetStringValue() != "my-agent" {
		t.Errorf("expected name 'my-agent', got %q", fields["name"].GetStringValue())
	}

	if fields["description"].GetStringValue() != "does things" {
		t.Errorf("expected description 'does things'")
	}

	if fields["version"].GetStringValue() != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", fields["version"].GetStringValue())
	}

	authors := fields["authors"].GetListValue().GetValues()
	if len(authors) == 0 || authors[0].GetStringValue() != "ACME Corp" {
		t.Errorf("expected authors=['ACME Corp'], got %v", authors)
	}

	if fields["schema_version"].GetStringValue() != translator.DefaultSchemaVersion {
		t.Errorf("expected schema_version %q", translator.DefaultSchemaVersion)
	}

	if fields["created_at"].GetStringValue() == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestA2AToRecord_UnwrappedFormat(t *testing.T) {
	input, err := structpb.NewStruct(map[string]any{
		"name":        "unwrapped-agent",
		"description": "direct card",
	})
	if err != nil {
		t.Fatalf("failed to build input: %v", err)
	}

	record, err := translator.A2AToRecord(input)
	if err != nil {
		t.Fatalf("A2AToRecord() error: %v", err)
	}

	if record.GetFields()["name"].GetStringValue() != "unwrapped-agent" {
		t.Errorf("expected name 'unwrapped-agent'")
	}
}

func TestA2AToRecord_DefaultNameAndDescription(t *testing.T) {
	input, err := structpb.NewStruct(map[string]any{
		"a2aCard": map[string]any{},
	})
	if err != nil {
		t.Fatalf("failed to build input: %v", err)
	}

	record, err := translator.A2AToRecord(input)
	if err != nil {
		t.Fatalf("A2AToRecord() error: %v", err)
	}

	if record.GetFields()["name"].GetStringValue() == "" {
		t.Error("expected fallback name to be set")
	}

	if record.GetFields()["description"].GetStringValue() == "" {
		t.Error("expected fallback description to be set")
	}
}

func TestA2AToRecord_ContainsA2AModule(t *testing.T) {
	input, err := structpb.NewStruct(map[string]any{
		"a2aCard": map[string]any{"name": "agent", "description": "desc"},
	})
	if err != nil {
		t.Fatalf("failed to build input: %v", err)
	}

	record, err := translator.A2AToRecord(input)
	if err != nil {
		t.Fatalf("A2AToRecord() error: %v", err)
	}

	found := false

	for _, mod := range record.GetFields()["modules"].GetListValue().GetValues() {
		if mod.GetStructValue().GetFields()["name"].GetStringValue() == translator.A2AModuleName {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("expected module %q in record", translator.A2AModuleName)
	}
}

func TestA2AToRecord_InvalidWrappedCard(t *testing.T) {
	// "a2aCard" key exists but is not a struct value
	s := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"a2aCard": {Kind: &structpb.Value_StringValue{StringValue: "not-a-struct"}},
		},
	}

	_, err := translator.A2AToRecord(s)
	if err == nil {
		t.Error("expected error when a2aCard is not a struct")
	}
}

// --- RecordToA2A ---

func TestRecordToA2A_ValidRecord(t *testing.T) {
	cardData, err := structpb.NewStruct(map[string]any{
		"name":        "test-agent",
		"description": "an agent",
	})
	if err != nil {
		t.Fatalf("failed to build card data: %v", err)
	}

	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": "integration/a2a",
				"data": map[string]any{
					"card_data":           cardData.AsMap(),
					"card_schema_version": "v1.0.0",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	a2a, err := translator.RecordToA2A(record)
	if err != nil {
		t.Fatalf("RecordToA2A() error: %v", err)
	}

	if a2a.GetFields()["name"].GetStringValue() != "test-agent" {
		t.Errorf("expected name 'test-agent'")
	}
}

func TestRecordToA2A_MissingModule(t *testing.T) {
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules":        []any{},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	_, err = translator.RecordToA2A(record)
	if err == nil {
		t.Error("expected error when A2A module is missing")
	}
}

func TestRecordToA2A_FallbackToModuleData(t *testing.T) {
	// No card_data — should fall back to returning module data directly
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": "integration/a2a",
				"data": map[string]any{
					"name": "fallback-agent",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	a2a, err := translator.RecordToA2A(record)
	if err != nil {
		t.Fatalf("RecordToA2A() error: %v", err)
	}

	if a2a == nil {
		t.Error("expected non-nil result for fallback path")
	}
}
