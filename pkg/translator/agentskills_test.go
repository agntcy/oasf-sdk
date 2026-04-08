// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func buildAgentSkillsRecord(t *testing.T, manifestMap map[string]any, body string) *structpb.Struct {
	t.Helper()

	manifestStruct, err := structpb.NewStruct(manifestMap)
	if err != nil {
		t.Fatalf("Failed to build manifest struct: %v", err)
	}

	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": AgentSkillsModuleName,
				"data": map[string]any{
					"skill_file":     "SKILL.md",
					"skill_manifest": manifestStruct.AsMap(),
					"skill_body":     body,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to build record struct: %v", err)
	}

	return record
}

func TestRecordToSkillMarkdown(t *testing.T) {
	manifestMap := map[string]any{
		"name":          "pdf-processing",
		"description":   "Extract PDF text and merge files.",
		"license":       "Apache-2.0",
		"compatibility": "Requires python3",
		"version":       "1.0.0",
		"allowed_tools": []any{"Read", "Bash(jq:*)"},
		"frontmatter_metadata": map[string]any{
			"author": "example-org",
		},
	}

	record := buildAgentSkillsRecord(t, manifestMap, "Use this skill when handling PDFs.")

	markdown, err := RecordToSkillMarkdown(record, WithBody("Use this skill when handling PDFs."))
	if err != nil {
		t.Fatalf("RecordToSkillMarkdown() error: %v", err)
	}

	if !strings.Contains(markdown, "name: pdf-processing") {
		t.Fatalf("Expected name in frontmatter")
	}

	if !strings.Contains(markdown, "description: Extract PDF text and merge files.") {
		t.Fatalf("Expected description in frontmatter")
	}

	if !strings.Contains(markdown, "license: Apache-2.0") {
		t.Fatalf("Expected license in frontmatter")
	}

	if !strings.Contains(markdown, "allowed-tools: Read Bash(jq:*)") {
		t.Fatalf("Expected allowed-tools in frontmatter")
	}

	if !strings.Contains(markdown, "metadata:") {
		t.Fatalf("Expected metadata section")
	}

	if !strings.Contains(markdown, "version: 1.0.0") {
		t.Fatalf("Expected version included")
	}

	if !strings.Contains(markdown, "Use this skill when handling PDFs.") {
		t.Fatalf("Expected body content")
	}
}

func TestRecordToSkillMarkdownMissingFields(t *testing.T) {
	record := buildAgentSkillsRecord(t, map[string]any{
		"name": "missing-description",
	}, "")

	_, err := RecordToSkillMarkdown(record)
	if err == nil {
		t.Fatalf("Expected error when description is missing")
	}
}

func TestRecordToSkillMarkdownNoModule(t *testing.T) {
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules":        []any{},
	})
	if err != nil {
		t.Fatalf("Failed to build record struct: %v", err)
	}

	_, err = RecordToSkillMarkdown(record)
	if err == nil {
		t.Fatalf("Expected error when agentskills module is missing")
	}
}

func TestSkillMarkdownToRecord(t *testing.T) {
	skillMD := `---
name: pdf-processing
description: Extract PDF text and merge files.
license: Apache-2.0
compatibility: Requires python3
version: 1.0.0
allowed-tools: Read Bash(jq:*)
metadata:
  author: example-org
---
Use this skill when handling PDFs.
`

	input, err := structpb.NewStruct(map[string]any{
		"skillMarkdown": skillMD,
	})
	if err != nil {
		t.Fatalf("Failed to build input struct: %v", err)
	}

	record, err := SkillMarkdownToRecord(input)
	if err != nil {
		t.Fatalf("SkillMarkdownToRecord() error: %v", err)
	}

	if record == nil {
		t.Fatalf("Expected non-nil record")
	}

	// Verify top-level fields.
	fields := record.GetFields()
	if fields["name"].GetStringValue() != "pdf-processing" {
		t.Errorf("Expected record name 'pdf-processing', got %q", fields["name"].GetStringValue())
	}

	if fields["description"].GetStringValue() != "Extract PDF text and merge files." {
		t.Errorf("Expected record description, got %q", fields["description"].GetStringValue())
	}

	if fields["version"].GetStringValue() != "1.0.0" {
		t.Errorf("Expected record version '1.0.0', got %q", fields["version"].GetStringValue())
	}

	// Verify agentskills module.
	found, moduleData := findModule(record, AgentSkillsModuleName)
	if !found || moduleData == nil {
		t.Fatalf("Expected agentskills module in record")
	}

	manifestVal := moduleData.GetFields()["skill_manifest"]
	if manifestVal == nil {
		t.Fatalf("Expected skill_manifest in module data")
	}

	manifest := manifestVal.GetStructValue()
	if manifest == nil {
		t.Fatalf("Expected skill_manifest to be a struct")
	}

	manifestFields := manifest.GetFields()

	if manifestFields["name"].GetStringValue() != "pdf-processing" {
		t.Errorf("Expected manifest name 'pdf-processing'")
	}

	if manifestFields["license"].GetStringValue() != "Apache-2.0" {
		t.Errorf("Expected manifest license 'Apache-2.0'")
	}

	if manifestFields["version"].GetStringValue() != "1.0.0" {
		t.Errorf("Expected manifest version '1.0.0'")
	}
}

func TestSkillMarkdownToRecordMissingName(t *testing.T) {
	skillMD := `---
description: Only description, no name.
---
`

	input, err := structpb.NewStruct(map[string]any{
		"skillMarkdown": skillMD,
	})
	if err != nil {
		t.Fatalf("Failed to build input: %v", err)
	}

	_, err = SkillMarkdownToRecord(input)
	if err == nil {
		t.Fatalf("Expected error for missing name")
	}
}

func TestSkillMarkdownToRecordMissingWrapper(t *testing.T) {
	input, err := structpb.NewStruct(map[string]any{
		"notSkillMarkdown": "---\nname: x\ndescription: y\n---\n",
	})
	if err != nil {
		t.Fatalf("Failed to build input: %v", err)
	}

	_, err = SkillMarkdownToRecord(input)
	if err == nil {
		t.Fatalf("Expected error for missing 'skillMarkdown' key")
	}
}

// findModule is a test helper to locate a named module in a record's modules list.
func findModule(record *structpb.Struct, name string) (bool, *structpb.Struct) {
	modulesVal, ok := record.GetFields()["modules"]
	if !ok {
		return false, nil
	}

	for _, modVal := range modulesVal.GetListValue().GetValues() {
		mod := modVal.GetStructValue()
		if mod == nil {
			continue
		}

		if mod.GetFields()["name"].GetStringValue() == name {
			return true, mod.GetFields()["data"].GetStructValue()
		}
	}

	return false, nil
}
