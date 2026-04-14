// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoder_test

import (
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

// --- JsonToProto ---

func TestJsonToProto_ValidObject(t *testing.T) {
	data := []byte(`{"name":"test","version":"1.0.0"}`)

	s, err := decoder.JsonToProto(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.GetFields()["name"].GetStringValue() != "test" {
		t.Errorf("expected name='test', got %q", s.GetFields()["name"].GetStringValue())
	}

	if s.GetFields()["version"].GetStringValue() != "1.0.0" {
		t.Errorf("expected version='1.0.0', got %q", s.GetFields()["version"].GetStringValue())
	}
}

func TestJsonToProto_Nested(t *testing.T) {
	data := []byte(`{"outer":{"inner":"value"}}`)

	s, err := decoder.JsonToProto(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inner := s.GetFields()["outer"].GetStructValue().GetFields()["inner"].GetStringValue()
	if inner != "value" {
		t.Errorf("expected inner='value', got %q", inner)
	}
}

func TestJsonToProto_InvalidJSON(t *testing.T) {
	_, err := decoder.JsonToProto([]byte(`not-json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestJsonToProto_EmptyObject(t *testing.T) {
	s, err := decoder.JsonToProto([]byte(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(s.GetFields()) != 0 {
		t.Errorf("expected empty fields, got %d", len(s.GetFields()))
	}
}

// --- StructToProto ---

func TestStructToProto_Map(t *testing.T) {
	input := map[string]any{
		"schema_version": "1.0.0",
		"name":           "my-agent",
	}

	s, err := decoder.StructToProto(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.GetFields()["schema_version"].GetStringValue() != "1.0.0" {
		t.Error("schema_version mismatch")
	}
}

func TestStructToProto_Struct(t *testing.T) {
	type MyRecord struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	s, err := decoder.StructToProto(MyRecord{Name: "agent", Version: "2.0.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s.GetFields()["name"].GetStringValue() != "agent" {
		t.Errorf("expected name='agent'")
	}
}

// --- ProtoToStruct ---

func TestProtoToStruct_RoundTrip(t *testing.T) {
	type Record struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	input := map[string]any{"name": "roundtrip", "version": "3.0.0"}

	proto, err := decoder.StructToProto(input)
	if err != nil {
		t.Fatalf("StructToProto error: %v", err)
	}

	result, err := decoder.ProtoToStruct[Record](proto)
	if err != nil {
		t.Fatalf("ProtoToStruct error: %v", err)
	}

	if result.Name != "roundtrip" {
		t.Errorf("expected name='roundtrip', got %q", result.Name)
	}

	if result.Version != "3.0.0" {
		t.Errorf("expected version='3.0.0', got %q", result.Version)
	}
}

func TestProtoToStruct_NestedFields(t *testing.T) {
	type Provider struct {
		Organization string `json:"organization"`
	}

	type Card struct {
		Name     string   `json:"name"`
		Provider Provider `json:"provider"`
	}

	proto, err := decoder.StructToProto(map[string]any{
		"name": "my-agent",
		"provider": map[string]any{
			"organization": "ACME",
		},
	})
	if err != nil {
		t.Fatalf("StructToProto error: %v", err)
	}

	result, err := decoder.ProtoToStruct[Card](proto)
	if err != nil {
		t.Fatalf("ProtoToStruct error: %v", err)
	}

	if result.Provider.Organization != "ACME" {
		t.Errorf("expected organization='ACME', got %q", result.Provider.Organization)
	}
}
