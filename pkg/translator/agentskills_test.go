// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

func TestBuildSkillMarkdown(t *testing.T) {
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

	manifestStruct, err := structpb.NewStruct(manifestMap)
	if err != nil {
		t.Fatalf("Failed to build manifest struct: %v", err)
	}

	markdown, err := BuildSkillMarkdown(manifestStruct, WithBody("Use this skill when handling PDFs."))
	if err != nil {
		t.Fatalf("BuildSkillMarkdown() error: %v", err)
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
		t.Fatalf("Expected version included in metadata")
	}

	if !strings.Contains(markdown, "Use this skill when handling PDFs.") {
		t.Fatalf("Expected body content")
	}
}

func TestBuildSkillMarkdownMissingFields(t *testing.T) {
	manifestStruct, err := structpb.NewStruct(map[string]any{
		"name": "missing-description",
	})
	if err != nil {
		t.Fatalf("Failed to build manifest struct: %v", err)
	}

	_, err = BuildSkillMarkdown(manifestStruct)
	if err == nil {
		t.Fatalf("Expected error when description is missing")
	}
}
