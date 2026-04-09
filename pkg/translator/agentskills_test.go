// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// buildAgentSkillsRecord constructs a minimal OASF record with an agentskills module
// containing the provided manifest fields. No skill_body — it is not in the schema.
func buildAgentSkillsRecord(t *testing.T, manifestMap map[string]any) *structpb.Struct {
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
				"id":   agentSkillsModuleID,
				"data": map[string]any{
					"skill_file":     "SKILL.md",
					"skill_manifest": manifestStruct.AsMap(),
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
		"version":       "1.0",
		"compatibility": []any{"Requires python3"},
		"allowed_tools": []any{"Read", "Bash(jq:*)"},
		"frontmatter_metadata": map[string]any{
			"author": "example-org",
		},
	}

	record := buildAgentSkillsRecord(t, manifestMap)

	markdown, err := RecordToSkillMarkdown(record)
	if err != nil {
		t.Fatalf("RecordToSkillMarkdown() error: %v", err)
	}

	checks := []struct {
		contains bool
		fragment string
		label    string
	}{
		{true, "name: pdf-processing", "name"},
		{true, "description: Extract PDF text and merge files.", "description"},
		{true, "license: Apache-2.0", "license"},
		{true, "compatibility: Requires python3", "compatibility (joined)"},
		{true, "allowed-tools: Read Bash(jq:*)", "allowed-tools"},
		{true, "metadata:", "metadata section"},
		{true, "author: example-org", "metadata author"},
		// version must appear in metadata, not as a top-level frontmatter key (per spec).
		{true, "version: 1.0", "version in metadata"},
		{false, "\nversion: 1.0\n", "version as top-level key (not allowed by spec)"},
	}

	for _, c := range checks {
		got := strings.Contains(markdown, c.fragment)
		if got != c.contains {
			if c.contains {
				t.Errorf("Expected %s in output.\nmarkdown:\n%s", c.label, markdown)
			} else {
				t.Errorf("Expected %s NOT in output.\nmarkdown:\n%s", c.label, markdown)
			}
		}
	}
}

func TestRecordToSkillMarkdownMissingDescription(t *testing.T) {
	record := buildAgentSkillsRecord(t, map[string]any{
		"name": "missing-description",
	})

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
	// version is NOT a top-level frontmatter key per the spec; it lives in metadata.
	skillMD := `---
name: pdf-processing
description: Extract PDF text and merge files.
license: Apache-2.0
compatibility: Requires python3
allowed-tools: Read Bash(jq:*)
metadata:
  author: example-org
  version: "1.0"
---
Use this skill when handling PDFs.
`

	input, err := structpb.NewStruct(map[string]any{"skillMarkdown": skillMD})
	if err != nil {
		t.Fatalf("Failed to build input struct: %v", err)
	}

	record, err := SkillMarkdownToRecord(input)
	if err != nil {
		t.Fatalf("SkillMarkdownToRecord() error: %v", err)
	}

	fields := record.GetFields()

	if fields["name"].GetStringValue() != "pdf-processing" {
		t.Errorf("Expected record name 'pdf-processing', got %q", fields["name"].GetStringValue())
	}

	// version comes from metadata["version"].
	if fields["version"].GetStringValue() != "1.0" {
		t.Errorf("Expected record version '1.0', got %q", fields["version"].GetStringValue())
	}

	// authors derived from metadata.author.
	authorVals := fields["authors"].GetListValue().GetValues()
	if len(authorVals) != 1 || authorVals[0].GetStringValue() != "example-org" {
		t.Errorf("Expected authors = [\"example-org\"], got %v", authorVals)
	}

	found, moduleData := findModule(record, AgentSkillsModuleName)
	if !found || moduleData == nil {
		t.Fatalf("Expected agentskills module %q in record", AgentSkillsModuleName)
	}

	manifest := moduleData.GetFields()["skill_manifest"].GetStructValue()
	if manifest == nil {
		t.Fatalf("Expected skill_manifest to be a struct")
	}

	mf := manifest.GetFields()

	if mf["name"].GetStringValue() != "pdf-processing" {
		t.Errorf("Expected manifest name 'pdf-processing'")
	}

	if mf["license"].GetStringValue() != "Apache-2.0" {
		t.Errorf("Expected manifest license 'Apache-2.0'")
	}

	// version is a top-level manifest field per the agentskills_manifest schema.
	if mf["version"].GetStringValue() != "1.0" {
		t.Errorf("Expected manifest version '1.0', got %q", mf["version"].GetStringValue())
	}

	// compatibility must be stored as []string per the agentskills_manifest schema.
	compatItems := mf["compatibility"].GetListValue().GetValues()
	if len(compatItems) != 1 || compatItems[0].GetStringValue() != "Requires python3" {
		t.Errorf("Expected compatibility = [\"Requires python3\"], got %v", compatItems)
	}

	// skill_body must NOT be stored (not in agentskills_data schema).
	if _, hasBody := moduleData.GetFields()["skill_body"]; hasBody {
		t.Errorf("skill_body must not be stored in the record: not defined in agentskills_data schema")
	}
}

func TestSkillMarkdownToRecordVersionFallback(t *testing.T) {
	// No version in frontmatter and no version in metadata → defaultVersion.
	skillMD := `---
name: simple-skill
description: A simple skill.
---
`

	input, err := structpb.NewStruct(map[string]any{"skillMarkdown": skillMD})
	if err != nil {
		t.Fatalf("Failed to build input: %v", err)
	}

	record, err := SkillMarkdownToRecord(input)
	if err != nil {
		t.Fatalf("SkillMarkdownToRecord() error: %v", err)
	}

	if record.GetFields()["version"].GetStringValue() != defaultVersion {
		t.Errorf("Expected default version %q, got %q", defaultVersion, record.GetFields()["version"].GetStringValue())
	}
}

func TestSkillMarkdownToRecordVersionFromMetadata(t *testing.T) {
	// No top-level version → falls back to metadata["version"].
	skillMD := `---
name: simple-skill
description: A simple skill.
metadata:
  version: "2.0"
---
`

	input, err := structpb.NewStruct(map[string]any{"skillMarkdown": skillMD})
	if err != nil {
		t.Fatalf("Failed to build input: %v", err)
	}

	record, err := SkillMarkdownToRecord(input)
	if err != nil {
		t.Fatalf("SkillMarkdownToRecord() error: %v", err)
	}

	if record.GetFields()["version"].GetStringValue() != "2.0" {
		t.Errorf("Expected version '2.0' from metadata, got %q", record.GetFields()["version"].GetStringValue())
	}
}

func TestSkillMarkdownToRecordMissingName(t *testing.T) {
	input, err := structpb.NewStruct(map[string]any{
		"skillMarkdown": "---\ndescription: Only description, no name.\n---\n",
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
