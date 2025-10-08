// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"github.com/agntcy/oasf-sdk/pkg/validator"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	MCPModuleName = "runtime/mcp"
	A2AModuleName = "runtime/a2a"
)

// RecordToGHCopilot translates a record into a GHCopilotMCPConfig structure.
func RecordToGHCopilot(record *structpb.Struct) (*GHCopilotMCPConfig, error) {
	// Get MCP module
	found, mcpModule := getModuleDataFromRecord(record, MCPModuleName)
	if !found {
		return nil, errors.New("MCP module not found in record")
	}

	// Process MCP module
	serversVal, ok := mcpModule.GetFields()["servers"]
	if !ok {
		return nil, errors.New("invalid or missing 'servers' in MCP module data")
	}

	serversStruct := serversVal.GetStructValue()
	if serversStruct == nil {
		return nil, errors.New("'servers' is not a struct")
	}

	servers := make(map[string]MCPServer)
	inputs := []MCPInput{}

	for serverName, serverVal := range serversStruct.Fields {
		serverMap := serverVal.GetStructValue()
		if serverMap == nil {
			continue
		}

		command, ok := serverMap.Fields["command"]
		if !ok {
			return nil, fmt.Errorf("missing 'command' for server '%s'", serverName)
		}

		args := []string{}
		if argsVal, ok := serverMap.Fields["args"]; ok {
			for _, arg := range argsVal.GetListValue().Values {
				args = append(args, arg.GetStringValue())
			}
		}

		env := map[string]string{}
		if envVal, ok := serverMap.Fields["env"]; ok {
			envStruct := envVal.GetStructValue()
			if envStruct != nil {
				for key, val := range envStruct.Fields {
					env[key] = val.GetStringValue()

					if after, ok0 := strings.CutPrefix(val.GetStringValue(), "${input:"); ok0 {
						id := strings.TrimSuffix(after, "}")
						inputs = append(inputs, MCPInput{
							ID:          id,
							Type:        "promptString",
							Password:    true,
							Description: fmt.Sprintf("Secret value for %s", id),
						})
					}
				}
			}
		}

		servers[serverName] = MCPServer{
			Command: command.GetStringValue(),
			Args:    args,
			Env:     env,
		}
	}

	return &GHCopilotMCPConfig{
		Servers: servers,
		Inputs:  inputs,
	}, nil
}

// RecordToA2A translates a record into an A2ACard structure.
func RecordToA2A(record *structpb.Struct) (*A2ACard, error) {
	// Get A2A module
	found, a2aModule := getModuleDataFromRecord(record, A2AModuleName)
	if !found {
		return nil, errors.New("A2A module not found in record")
	}

	// Process A2A module
	jsonBytes, err := json.Marshal(a2aModule)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal A2A data to JSON: %w", err)
	}

	var card A2ACard
	if err := json.Unmarshal(jsonBytes, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal A2A data into A2ACard: %w", err)
	}

	return &card, nil
}

// A2AToRecord translates an A2A card data back into an OASF-compliant record format.
func A2AToRecord(a2aData *structpb.Struct) (*structpb.Struct, error) {
	// Extract the a2aCard from the input data
	a2aCardVal, ok := a2aData.GetFields()["a2aCard"]
	if !ok {
		return nil, errors.New("missing 'a2aCard' in input data")
	}

	a2aCardStruct := a2aCardVal.GetStructValue()
	if a2aCardStruct == nil {
		return nil, errors.New("'a2aCard' is not a struct")
	}

	// Convert A2A card struct to map for easier access
	cardMap := a2aCardStruct.AsMap()

	// Extract name and description from A2A card for record metadata
	cardName := "generated-agent"
	cardDescription := "Agent generated from A2A card"

	if name, ok := cardMap["name"]; ok {
		if nameStr, ok := name.(string); ok {
			cardName = nameStr
		}
	}
	if description, ok := cardMap["description"]; ok {
		if descStr, ok := description.(string); ok {
			cardDescription = descStr
		}
	}

	// Create A2A data structure conforming to OASF v0.7.0 A2A data schema
	a2aModuleData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"card_data": {
				Kind: &structpb.Value_StructValue{StructValue: a2aCardStruct},
			},
			"protocol_version": {
				Kind: &structpb.Value_StringValue{StringValue: "v1.0.0"},
			},
			"capabilities": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: "streaming"}},
						},
					},
				},
			},
			"input_modes": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: "text/plain"}},
							{Kind: &structpb.Value_StringValue{StringValue: "application/json"}},
						},
					},
				},
			},
			"output_modes": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: "text/html"}},
							{Kind: &structpb.Value_StringValue{StringValue: "application/json"}},
						},
					},
				},
			},
			"security_schemes": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: "none"}},
						},
					},
				},
			},
			"transports": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StringValue{StringValue: "http"}},
						},
					},
				},
			},
		},
	}

	// Create the A2A module with schema-compliant data
	a2aModule := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name": {
				Kind: &structpb.Value_StringValue{StringValue: A2AModuleName},
			},
			"version": {
				Kind: &structpb.Value_StringValue{StringValue: "v1.0.0"},
			},
			"data": {
				Kind: &structpb.Value_StructValue{StructValue: a2aModuleData},
			},
		},
	}

	// Create the modules list
	modulesList := &structpb.ListValue{
		Values: []*structpb.Value{
			{
				Kind: &structpb.Value_StructValue{StructValue: a2aModule},
			},
		},
	}

	// Create OASF-compliant record with all required fields
	record := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name": {
				Kind: &structpb.Value_StringValue{StringValue: cardName},
			},
			"schema_version": {
				Kind: &structpb.Value_StringValue{StringValue: "0.7.0"},
			},
			"version": {
				Kind: &structpb.Value_StringValue{StringValue: "v1.0.0"},
			},
			"description": {
				Kind: &structpb.Value_StringValue{StringValue: cardDescription},
			},
			"authors": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StringValue{StringValue: "Generated by OASF SDK"},
							},
						},
					},
				},
			},
			"created_at": {
				Kind: &structpb.Value_StringValue{StringValue: "2025-10-06T00:00:00Z"},
			},
			"skills": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{Values: []*structpb.Value{}}, // Empty skills array
				},
			},
			"locators": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{Values: []*structpb.Value{}}, // Empty locators array
				},
			},
			"modules": {
				Kind: &structpb.Value_ListValue{ListValue: modulesList},
			},
		},
	}

	// Validate OASF compliance using the validator
	v, err := validator.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}
	if _, _, err := v.ValidateRecord(record); err != nil {
		return nil, fmt.Errorf("record validation failed: %w", err)
	}
	return record, nil
}

func getModuleDataFromRecord(record *structpb.Struct, moduleName string) (bool, *structpb.Struct) {
	// Find module by name
	modules, ok := record.GetFields()["modules"]
	if !ok {
		return false, nil
	}

	for _, module := range modules.GetListValue().Values {
		if strings.HasSuffix(module.GetStructValue().GetFields()["name"].GetStringValue(), moduleName) {
			return true, module.GetStructValue().GetFields()["data"].GetStructValue()
		}
	}
	return false, nil
}
