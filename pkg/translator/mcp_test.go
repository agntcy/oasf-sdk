// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator_test

import (
	"maps"
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/types/known/structpb"
)

func minimalMCPInput(t *testing.T, extras map[string]any) *structpb.Struct {
	t.Helper()

	server := map[string]any{
		"name":        "io.github.example/my-server",
		"description": "A test MCP server",
		"version":     "1.0.0",
		"packages": []any{
			map[string]any{
				"registryType": "npm",
				"identifier":   "@example/my-server",
				"version":      "1.0.0",
				"transport":    map[string]any{"type": "stdio"},
			},
		},
	}

	maps.Copy(server, extras)

	s, err := structpb.NewStruct(map[string]any{"server": server})
	if err != nil {
		t.Fatalf("failed to build mcp input: %v", err)
	}

	return s
}

// --- MCPToRecord ---

func TestMCPToRecord_BasicFields(t *testing.T) {
	record, err := translator.MCPToRecord(minimalMCPInput(t, nil))
	if err != nil {
		t.Fatalf("MCPToRecord() error: %v", err)
	}

	fields := record.GetFields()

	if fields["name"].GetStringValue() != "io.github.example/my-server" {
		t.Errorf("unexpected name: %q", fields["name"].GetStringValue())
	}

	if fields["description"].GetStringValue() != "A test MCP server" {
		t.Error("description mismatch")
	}

	if fields["version"].GetStringValue() != "1.0.0" {
		t.Error("version mismatch")
	}

	if fields["schema_version"].GetStringValue() != translator.DefaultSchemaVersion {
		t.Errorf("expected default schema_version, got %q", fields["schema_version"].GetStringValue())
	}

	if fields["created_at"].GetStringValue() == "" {
		t.Error("expected non-empty created_at")
	}
}

func TestMCPToRecord_AuthorFromNamespace(t *testing.T) {
	record, err := translator.MCPToRecord(minimalMCPInput(t, nil))
	if err != nil {
		t.Fatalf("MCPToRecord() error: %v", err)
	}

	authors := record.GetFields()["authors"].GetListValue().GetValues()
	if len(authors) == 0 {
		t.Fatal("expected at least one author")
	}

	if authors[0].GetStringValue() != "example" {
		t.Errorf("expected author 'example' extracted from namespace, got %q", authors[0].GetStringValue())
	}
}

func TestMCPToRecord_ContainsMCPModule(t *testing.T) {
	record, err := translator.MCPToRecord(minimalMCPInput(t, nil))
	if err != nil {
		t.Fatalf("MCPToRecord() error: %v", err)
	}

	found := false

	for _, mod := range record.GetFields()["modules"].GetListValue().GetValues() {
		if mod.GetStructValue().GetFields()["name"].GetStringValue() == translator.MCPModuleName {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("expected module %q in record", translator.MCPModuleName)
	}
}

func TestMCPToRecord_RepositoryLocator(t *testing.T) {
	record, err := translator.MCPToRecord(minimalMCPInput(t, map[string]any{
		"repository": map[string]any{
			"type": "git",
			"url":  "https://github.com/example/my-server",
		},
	}))
	if err != nil {
		t.Fatalf("MCPToRecord() error: %v", err)
	}

	locators := record.GetFields()["locators"].GetListValue().GetValues()
	if len(locators) == 0 {
		t.Fatal("expected at least one locator when repository URL is provided")
	}

	urls := locators[0].GetStructValue().GetFields()["urls"].GetListValue().GetValues()
	if len(urls) == 0 || urls[0].GetStringValue() != "https://github.com/example/my-server" {
		t.Errorf("expected locator URL 'https://github.com/example/my-server'")
	}
}

func TestMCPToRecord_MissingServer(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{"other": "field"})
	if err != nil {
		t.Fatalf("failed to build struct: %v", err)
	}

	_, err = translator.MCPToRecord(s)
	if err == nil {
		t.Error("expected error when 'server' field is missing")
	}
}

func TestMCPToRecord_NoPackagesOrRemotes(t *testing.T) {
	s, err := structpb.NewStruct(map[string]any{
		"server": map[string]any{
			"name":        "empty-server",
			"description": "no packages",
			"version":     "1.0.0",
		},
	})
	if err != nil {
		t.Fatalf("failed to build struct: %v", err)
	}

	_, err = translator.MCPToRecord(s)
	if err == nil {
		t.Error("expected error when no packages or remotes are present")
	}
}

func TestMCPToRecord_PyPIPackage(t *testing.T) {
	record, err := translator.MCPToRecord(minimalMCPInput(t, map[string]any{
		"packages": []any{
			map[string]any{
				"registryType": "pypi",
				"identifier":   "my-mcp-server",
				"version":      "0.1.0",
				"transport":    map[string]any{"type": "stdio"},
			},
		},
	}))
	if err != nil {
		t.Fatalf("MCPToRecord() error: %v", err)
	}

	// Find the MCP module and check the connection command
	for _, mod := range record.GetFields()["modules"].GetListValue().GetValues() {
		ms := mod.GetStructValue()
		if ms.GetFields()["name"].GetStringValue() != translator.MCPModuleName {
			continue
		}

		connections := ms.GetFields()["data"].GetStructValue().
			GetFields()["connections"].GetListValue().GetValues()

		if len(connections) == 0 {
			t.Fatal("expected at least one connection")
		}

		cmd := connections[0].GetStructValue().GetFields()["command"].GetStringValue()
		if cmd != "python" {
			t.Errorf("expected command 'python' for pypi, got %q", cmd)
		}
	}
}

func TestMCPToRecord_SSERemote(t *testing.T) {
	record, err := translator.MCPToRecord(minimalMCPInput(t, map[string]any{
		"remotes": []any{
			map[string]any{
				"type": "sse",
				"url":  "https://api.example.com/mcp/sse",
			},
		},
	}))
	if err != nil {
		t.Fatalf("MCPToRecord() error: %v", err)
	}

	for _, mod := range record.GetFields()["modules"].GetListValue().GetValues() {
		ms := mod.GetStructValue()
		if ms.GetFields()["name"].GetStringValue() != translator.MCPModuleName {
			continue
		}

		connections := ms.GetFields()["data"].GetStructValue().
			GetFields()["connections"].GetListValue().GetValues()

		// First connection is from the default npm package, second from SSE remote
		for _, conn := range connections {
			cs := conn.GetStructValue()
			if cs.GetFields()["type"].GetStringValue() == "sse" {
				if cs.GetFields()["url"].GetStringValue() != "https://api.example.com/mcp/sse" {
					t.Errorf("unexpected SSE url: %q", cs.GetFields()["url"].GetStringValue())
				}

				return
			}
		}

		t.Error("expected SSE connection in record")
	}
}

// --- RecordToGHCopilot ---

func makeRecordWithMCPModule100(t *testing.T, serverName, command string) *structpb.Struct {
	t.Helper()

	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": translator.MCPModuleName,
				"data": map[string]any{
					"name": serverName,
					"connections": []any{
						map[string]any{
							"type":    "stdio",
							"command": command,
							"args":    []any{"-y", "@example/server"},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	return record
}

func TestRecordToGHCopilot_BasicStdio(t *testing.T) {
	record := makeRecordWithMCPModule100(t, "my-server", "npx")

	config, err := translator.RecordToGHCopilot(record)
	if err != nil {
		t.Fatalf("RecordToGHCopilot() error: %v", err)
	}

	if len(config.Servers) == 0 {
		t.Fatal("expected at least one server in GH Copilot config")
	}

	server, ok := config.Servers["my"]
	if !ok {
		t.Fatalf("expected server 'my' (normalized from 'my-server'), got: %v", config.Servers)
	}

	if server.Command != "npx" {
		t.Errorf("expected command 'npx', got %q", server.Command)
	}
}

func TestRecordToGHCopilot_MissingModule(t *testing.T) {
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules":        []any{},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	_, err = translator.RecordToGHCopilot(record)
	if err == nil {
		t.Error("expected error when MCP module is missing")
	}
}

func TestRecordToGHCopilot_EnvVarsCreateInputs(t *testing.T) {
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": translator.MCPModuleName,
				"data": map[string]any{
					"name": "secret-server",
					"connections": []any{
						map[string]any{
							"type":    "stdio",
							"command": "npx",
							"env_vars": []any{
								map[string]any{"name": "API_KEY"},
							},
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

	if len(config.Inputs) == 0 {
		t.Error("expected at least one input for env var without default")
	}

	found := false

	for _, inp := range config.Inputs {
		if inp.ID == "API_KEY" {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("expected input for API_KEY, got: %v", config.Inputs)
	}
}

func TestRecordToGHCopilot_InvalidModuleData(t *testing.T) {
	// Module data has neither 'name' (1.0.0 format) nor 'servers' (0.7.0/0.8.0)
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": translator.MCPModuleName,
				"data": map[string]any{
					"unexpected_field": "value",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to build record: %v", err)
	}

	_, err = translator.RecordToGHCopilot(record)
	if err == nil {
		t.Error("expected error for invalid MCP module data format")
	}
}
