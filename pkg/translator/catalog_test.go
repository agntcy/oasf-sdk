// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator_test

import (
	"slices"
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/types/known/structpb"
)

// moduleMap builds a minimal OASF module object: name + structured data.
func moduleMap(name string, data map[string]any) map[string]any {
	m := map[string]any{"name": name}
	if data != nil {
		m["data"] = data
	}

	return m
}

// taxonomyMap builds a {"name": ...} taxonomy entry (skill/domain).
func taxonomyMap(name string) map[string]any {
	return map[string]any{"name": name}
}

func mustStruct(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()

	s, err := structpb.NewStruct(m)
	if err != nil {
		t.Fatalf("failed to build struct: %v", err)
	}

	return s
}

func entryString(t *testing.T, entry *structpb.Struct, field string) string {
	t.Helper()

	return entry.GetFields()[field].GetStringValue()
}

func entryTags(entry *structpb.Struct) []string {
	list := entry.GetFields()["tags"].GetListValue()
	if list == nil {
		return nil
	}

	out := make([]string, 0, len(list.GetValues()))
	for _, v := range list.GetValues() {
		out = append(out, v.GetStringValue())
	}

	return out
}

const (
	testCID     = "baeareibxiiy45pg4bjwhbijgh35epzjhnh6lvaxts2qggcgssn3glzdh64"
	testBaseURN = "urn:ai:org.agntcy:cid:" + testCID
)

func TestRecordToCatalog_MCPLeaf(t *testing.T) {
	moduleData := map[string]any{
		"name": "io.github.github/github-mcp-server",
		"connections": []any{
			map[string]any{"type": "stdio", "command": "docker"},
		},
	}

	record := mustStruct(t, map[string]any{
		"name":           "io.github.github/github-mcp-server",
		"version":        "1.0.4",
		"description":    "Connect AI assistants to GitHub.",
		"schema_version": "1.0.0",
		"created_at":     "2026-05-12T00:40:01Z",
		"skills": []any{
			taxonomyMap("retrieval_augmented_generation/document_or_database_question_answering"),
			taxonomyMap("retrieval_augmented_generation/retrieval_of_information"),
		},
		"domains": []any{taxonomyMap("technology/cloud_computing")},
		"modules": []any{moduleMap(translator.MCPModuleName, moduleData)},
	})

	entry, err := translator.RecordToCatalog(record, translator.WithCatalogCID(testCID))
	if err != nil {
		t.Fatalf("RecordToCatalog() error: %v", err)
	}

	if got, want := entryString(t, entry, "identifier"), testBaseURN; got != want {
		t.Errorf("identifier = %q, want %q", got, want)
	}

	if got := entryString(t, entry, "media_type"); got != translator.MCPCatalogMediaType {
		t.Errorf("media_type = %q, want %q", got, translator.MCPCatalogMediaType)
	}

	if got := entryString(t, entry, "display_name"); got != "io.github.github/github-mcp-server" {
		t.Errorf("display_name = %q", got)
	}

	if got := entryString(t, entry, "version"); got != "1.0.4" {
		t.Errorf("version = %q, want 1.0.4", got)
	}

	if got := entryString(t, entry, "updated_at"); got != "2026-05-12T00:40:01Z" {
		t.Errorf("updated_at = %q", got)
	}

	// data must equal the module's structured data, not the artifact descriptor.
	data := entry.GetFields()["data"].GetStructValue()
	if data == nil {
		t.Fatal("expected inline data on leaf entry")
	}

	if got := data.GetFields()["name"].GetStringValue(); got != "io.github.github/github-mcp-server" {
		t.Errorf("data.name = %q", got)
	}

	wantTags := []string{
		"oasf:v1.0.0:skills:retrieval_augmented_generation/document_or_database_question_answering",
		"oasf:v1.0.0:skills:retrieval_augmented_generation/retrieval_of_information",
		"oasf:v1.0.0:domains:technology/cloud_computing",
	}
	if got := entryTags(entry); !slices.Equal(got, wantTags) {
		t.Errorf("tags = %v, want %v", got, wantTags)
	}
}

func TestRecordToCatalog_A2ALeaf(t *testing.T) {
	record := mustStruct(t, map[string]any{
		"name":           "Langraph Planner Agent",
		"version":        "1.0.0",
		"schema_version": "1.0.0",
		"modules": []any{
			moduleMap(translator.A2AModuleName, map[string]any{
				"card_schema_version": "v1.0.0",
				"card_data":           map[string]any{"name": "Langraph Planner Agent"},
			}),
		},
	})

	entry, err := translator.RecordToCatalog(record, translator.WithCatalogCID(testCID))
	if err != nil {
		t.Fatalf("RecordToCatalog() error: %v", err)
	}

	if got := entryString(t, entry, "media_type"); got != translator.A2ACatalogMediaType {
		t.Errorf("media_type = %q, want %q", got, translator.A2ACatalogMediaType)
	}

	if got, want := entryString(t, entry, "identifier"), testBaseURN; got != want {
		t.Errorf("identifier = %q, want %q", got, want)
	}
}

func TestRecordToCatalog_SkillLeaf(t *testing.T) {
	record := mustStruct(t, map[string]any{
		"name":           "brand-guidelines",
		"version":        "1.0.0",
		"schema_version": "1.0.0",
		"domains": []any{
			taxonomyMap("technology/internet_of_things"),
			taxonomyMap("technology/software_engineering"),
		},
		"modules": []any{
			moduleMap(translator.AgentSkillsModuleName, map[string]any{
				"skill_file": "SKILL.md",
			}),
		},
	})

	entry, err := translator.RecordToCatalog(record, translator.WithCatalogCID(testCID))
	if err != nil {
		t.Fatalf("RecordToCatalog() error: %v", err)
	}

	if got := entryString(t, entry, "media_type"); got != translator.AgentSkillsCatalogMediaType {
		t.Errorf("media_type = %q, want %q", got, translator.AgentSkillsCatalogMediaType)
	}

	wantTags := []string{
		"oasf:v1.0.0:domains:technology/internet_of_things",
		"oasf:v1.0.0:domains:technology/software_engineering",
	}
	if got := entryTags(entry); !slices.Equal(got, wantTags) {
		t.Errorf("tags = %v, want %v", got, wantTags)
	}
}

func TestRecordToCatalog_MultiModuleContainer(t *testing.T) {
	record := mustStruct(t, map[string]any{
		"name":           "multi-agent",
		"version":        "2.0.0",
		"schema_version": "1.0.0",
		"skills":         []any{taxonomyMap("nlp/generation")},
		"modules": []any{
			moduleMap(translator.MCPModuleName, map[string]any{"name": "mcp-part"}),
			moduleMap(translator.A2AModuleName, map[string]any{"card_data": map[string]any{}}),
		},
	})

	entry, err := translator.RecordToCatalog(record, translator.WithCatalogCID(testCID))
	if err != nil {
		t.Fatalf("RecordToCatalog() error: %v", err)
	}

	if got := entryString(t, entry, "media_type"); got != translator.CatalogContainerMediaType {
		t.Errorf("media_type = %q, want %q", got, translator.CatalogContainerMediaType)
	}

	if got, want := entryString(t, entry, "identifier"), testBaseURN; got != want {
		t.Errorf("identifier = %q, want %q", got, want)
	}

	// Container carries the record-level tags.
	if got := entryTags(entry); !slices.Equal(got, []string{"oasf:v1.0.0:skills:nlp/generation"}) {
		t.Errorf("container tags = %v", got)
	}

	nestedCatalog := entry.GetFields()["data"].GetStructValue()
	if nestedCatalog == nil {
		t.Fatal("expected nested catalog data")
	}

	if got := nestedCatalog.GetFields()["specVersion"].GetStringValue(); got != translator.DefaultCatalogSpecVersion {
		t.Errorf("specVersion = %q, want %q", got, translator.DefaultCatalogSpecVersion)
	}

	entries := nestedCatalog.GetFields()["entries"].GetListValue()
	if entries == nil || len(entries.GetValues()) != 2 {
		t.Fatalf("expected 2 nested entries, got %v", entries)
	}

	// Modules are sorted by name: "integration/a2a" < "integration/mcp".
	// Nested entries use camelCase keys since they live inside a
	// google.protobuf.Value that is serialized verbatim.
	first := entries.GetValues()[0].GetStructValue()
	if got, want := first.GetFields()["identifier"].GetStringValue(), testBaseURN+":a2a"; got != want {
		t.Errorf("first nested identifier = %q, want %q", got, want)
	}

	if got := first.GetFields()["displayName"].GetStringValue(); got != "multi-agent (A2A)" {
		t.Errorf("first nested displayName = %q, want %q", got, "multi-agent (A2A)")
	}

	if got := first.GetFields()["mediaType"].GetStringValue(); got != translator.A2ACatalogMediaType {
		t.Errorf("first nested mediaType = %q", got)
	}

	// Nested entries deliberately carry no tags.
	if _, ok := first.GetFields()["tags"]; ok {
		t.Error("nested entry should not carry tags")
	}

	second := entries.GetValues()[1].GetStructValue()
	if got, want := second.GetFields()["identifier"].GetStringValue(), testBaseURN+":mcp"; got != want {
		t.Errorf("second nested identifier = %q, want %q", got, want)
	}
}

func TestRecordToCatalog_NoKnownModules(t *testing.T) {
	record := mustStruct(t, map[string]any{
		"name":    "no-modules",
		"modules": []any{moduleMap("runtime/framework", map[string]any{"x": "y"})},
	})

	if _, err := translator.RecordToCatalog(record, translator.WithCatalogCID(testCID)); err == nil {
		t.Error("expected error for record without known catalog modules")
	}
}

func TestRecordToCatalog_MissingCID(t *testing.T) {
	record := mustStruct(t, map[string]any{
		"name":    "needs-cid",
		"modules": []any{moduleMap(translator.MCPModuleName, map[string]any{"x": "y"})},
	})

	// No CID, and a whitespace-only CID, must both error.
	if _, err := translator.RecordToCatalog(record); err == nil {
		t.Error("expected error when no CID is provided")
	}

	if _, err := translator.RecordToCatalog(record, translator.WithCatalogCID("  ")); err == nil {
		t.Error("expected error when a blank CID is provided")
	}
}

func TestRecordToCatalog_NilRecord(t *testing.T) {
	if _, err := translator.RecordToCatalog(nil); err == nil {
		t.Error("expected error for nil record")
	}
}

func TestRecordToCatalog_HostOverrideAndModuleWithoutData(t *testing.T) {
	record := mustStruct(t, map[string]any{
		"name":           "no-data-agent",
		"schema_version": "1.0.0",
		"modules":        []any{moduleMap(translator.MCPModuleName, nil)},
	})

	entry, err := translator.RecordToCatalog(record,
		translator.WithCatalogHost("example.com"),
		translator.WithCatalogCID("cid123"),
	)
	if err != nil {
		t.Fatalf("RecordToCatalog() error: %v", err)
	}

	wantURN := "urn:ai:example.com:cid:cid123"
	if got := entryString(t, entry, "identifier"); got != wantURN {
		t.Errorf("identifier = %q, want %q", got, wantURN)
	}

	// The artifact is always carried as inline data (empty when the module
	// has none); url is never set.
	if _, ok := entry.GetFields()["data"]; !ok {
		t.Error("expected inline data to always be present")
	}

	if _, ok := entry.GetFields()["url"]; ok {
		t.Error("expected url never to be set")
	}
}

func TestRecordToCatalog_AnnotationsAsTags(t *testing.T) {
	record := mustStruct(t, map[string]any{
		"name":           "annotated",
		"schema_version": "1.0.0",
		"annotations": map[string]any{
			"team":     "platform",
			"featured": "",
		},
		"modules": []any{moduleMap(translator.MCPModuleName, map[string]any{"x": "y"})},
	})

	entry, err := translator.RecordToCatalog(record, translator.WithCatalogCID(testCID))
	if err != nil {
		t.Fatalf("RecordToCatalog() error: %v", err)
	}

	// Annotation tags are key-sorted: "featured" (bare) before "team=platform".
	wantTags := []string{"featured", "team=platform"}
	if got := entryTags(entry); !slices.Equal(got, wantTags) {
		t.Errorf("tags = %v, want %v", got, wantTags)
	}
}
