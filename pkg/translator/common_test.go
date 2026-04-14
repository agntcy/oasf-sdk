// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator_test

import (
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/types/known/structpb"
)

const schemaVersion100 = "1.0.0"

// --- WithVersion / validateMajorVersion (tested via A2AToRecord) ---

func minimalA2AInput(t *testing.T) *structpb.Struct {
	t.Helper()

	s, err := structpb.NewStruct(map[string]any{
		"a2aCard": map[string]any{
			"name":        "test-agent",
			"description": "test",
		},
	})
	if err != nil {
		t.Fatalf("failed to build a2a input: %v", err)
	}

	return s
}

func TestWithVersion_ValidVersion(t *testing.T) {
	record, err := translator.A2AToRecord(minimalA2AInput(t), translator.WithVersion(schemaVersion100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := record.GetFields()["schema_version"].GetStringValue(); got != schemaVersion100 {
		t.Errorf("expected schema_version %q, got %q", schemaVersion100, got)
	}
}

func TestWithVersion_AnotherValidVersion(t *testing.T) {
	record, err := translator.A2AToRecord(minimalA2AInput(t), translator.WithVersion("1.2.3"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := record.GetFields()["schema_version"].GetStringValue(); got != "1.2.3" {
		t.Errorf("expected schema_version '1.2.3', got %q", got)
	}
}

func TestWithVersion_UnsupportedMajor(t *testing.T) {
	_, err := translator.A2AToRecord(minimalA2AInput(t), translator.WithVersion("2.0.0"))
	if err == nil {
		t.Error("expected error for unsupported major version 2.0.0")
	}
}

func TestWithVersion_InvalidFormat(t *testing.T) {
	_, err := translator.A2AToRecord(minimalA2AInput(t), translator.WithVersion("not-a-version"))
	if err == nil {
		t.Error("expected error for malformed version string")
	}
}

func TestWithVersion_DefaultUsedWhenNotSet(t *testing.T) {
	record, err := translator.A2AToRecord(minimalA2AInput(t))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := record.GetFields()["schema_version"].GetStringValue()
	if got != translator.DefaultSchemaVersion {
		t.Errorf("expected default schema_version %q, got %q", translator.DefaultSchemaVersion, got)
	}
}

// --- normalizeServerName (tested indirectly via RecordToGHCopilot) ---

func TestNormalizeServerName_MCPSuffix(t *testing.T) {
	// Build a record whose MCP module has a server named "github-mcp-server"
	// and verify RecordToGHCopilot normalises the key to "github".
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": translator.MCPModuleName,
				"data": map[string]any{
					"name": "github-mcp-server",
					"connections": []any{
						map[string]any{
							"type":    "stdio",
							"command": "npx",
							"args":    []any{"-y", "@github/mcp-server"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	config, err := translator.RecordToGHCopilot(record)
	if err != nil {
		t.Fatalf("RecordToGHCopilot() error: %v", err)
	}

	if _, ok := config.Servers["github"]; !ok {
		t.Errorf("expected normalised server name 'github', got servers: %v", config.Servers)
	}
}

func TestNormalizeServerName_ServerSuffix(t *testing.T) {
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": translator.MCPModuleName,
				"data": map[string]any{
					"name": "filesystem-server",
					"connections": []any{
						map[string]any{
							"type":    "stdio",
							"command": "npx",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	config, err := translator.RecordToGHCopilot(record)
	if err != nil {
		t.Fatalf("RecordToGHCopilot() error: %v", err)
	}

	if _, ok := config.Servers["filesystem"]; !ok {
		t.Errorf("expected normalised server name 'filesystem', got: %v", config.Servers)
	}
}

// --- Constants ---

func TestConstants(t *testing.T) {
	if translator.MCPModuleName != "integration/mcp" {
		t.Errorf("MCPModuleName unexpected: %q", translator.MCPModuleName)
	}

	if translator.A2AModuleName != "integration/a2a" {
		t.Errorf("A2AModuleName unexpected: %q", translator.A2AModuleName)
	}

	if translator.DefaultSchemaVersion != schemaVersion100 {
		t.Errorf("DefaultSchemaVersion unexpected: %q", translator.DefaultSchemaVersion)
	}

	if translator.AgentSkillsModuleName != "core/language_model/agentskills" {
		t.Errorf("AgentSkillsModuleName unexpected: %q", translator.AgentSkillsModuleName)
	}
}
