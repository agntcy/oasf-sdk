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
	MCPModuleName = "integration/mcp"
	A2AModuleName = "integration/a2a"
	targetSchema  = "0.8.0"
)

// RecordToGHCopilot translates a record into a GHCopilotMCPConfig structure.
func RecordToGHCopilot(record *structpb.Struct) (*GHCopilotMCPConfig, error) {
	// Get MCP module - use suffix matching
	found, mcpModule := getModuleDataFromRecord(record, "mcp")
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
	// Get A2A module - use suffix matching
	found, a2aModule := getModuleDataFromRecord(record, "a2a")
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

	A2ACardStruct := a2aCardVal.GetStructValue()
	if A2ACardStruct == nil {
		return nil, errors.New("'a2aCard' is not a struct")
	}

	// Convert A2A card struct to map for easier access
	cardMap := A2ACardStruct.AsMap()

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

	// Create A2A data structure conforming to OASF v0.8.0 A2A data schema
	A2AModuleData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"card_data": {
				Kind: &structpb.Value_StructValue{StructValue: A2ACardStruct},
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
	A2AModule := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name": {
				Kind: &structpb.Value_StringValue{StringValue: A2AModuleName},
			},
			"data": {
				Kind: &structpb.Value_StructValue{StructValue: A2AModuleData},
			},
		},
	}

	// Create the modules list
	modulesList := &structpb.ListValue{
		Values: []*structpb.Value{
			{
				Kind: &structpb.Value_StructValue{StructValue: A2AModule},
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
				Kind: &structpb.Value_StringValue{StringValue: targetSchema},
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
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"id": {
												Kind: &structpb.Value_NumberValue{NumberValue: 0},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "base_skill"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"locators": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"type": {
												Kind: &structpb.Value_StringValue{StringValue: "source_code"},
											},
											"url": {
												Kind: &structpb.Value_StringValue{StringValue: "https://example.com/mcp-server.git"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"domains": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"id": {
												Kind: &structpb.Value_NumberValue{NumberValue: 0},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "base_domain"},
											},
										},
									},
								},
							},
						},
					},
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

// MCPToRecord translates an MCP Registry server.json into an OASF-compliant record format.
func MCPToRecord(mcpData *structpb.Struct) (*structpb.Struct, error) {
	// Extract the server from the input data
	mcpServerVal, ok := mcpData.GetFields()["server"]
	if !ok {
		return nil, errors.New("missing 'server' in input data")
	}

	mcpServerStruct := mcpServerVal.GetStructValue()
	if mcpServerStruct == nil {
		return nil, errors.New("'server' is not a struct")
	}

	// Convert MCP server.json struct to map for easier access
	serverMap := mcpServerStruct.AsMap()

	// Extract metadata from server.json for record metadata
	serverName := "generated-mcp-agent"
	serverTitle := ""
	serverDescription := "Agent generated from MCP server.json"
	serverVersion := "v1.0.0"
	websiteUrl := ""
	repoUrl := "https://example.com/mcp-server.git"
	authors := []string{"Generated by OASF SDK"}

	if name, ok := serverMap["name"].(string); ok {
		serverName = name
	}
	if title, ok := serverMap["title"].(string); ok {
		serverTitle = title
	}
	if description, ok := serverMap["description"].(string); ok {
		serverDescription = description
	}
	if version, ok := serverMap["version"].(string); ok {
		serverVersion = version
	}
	if url, ok := serverMap["websiteUrl"].(string); ok {
		websiteUrl = url
	}
	if repo, ok := serverMap["repository"].(map[string]interface{}); ok {
		if url, ok := repo["url"].(string); ok {
			repoUrl = url
		}
	}

	// Extract namespace/vendor from name (e.g., "io.github.vendor/server" -> "vendor")
	nameParts := strings.Split(serverName, "/")
	if len(nameParts) > 1 {
		namespaceParts := strings.Split(nameParts[0], ".")
		if len(namespaceParts) > 0 {
			vendor := namespaceParts[len(namespaceParts)-1]
			authors = []string{vendor}
		}
	}

	// Build MCP servers array from packages
	var mcpServers []*structpb.Value
	if packages, ok := serverMap["packages"].([]interface{}); ok {
		for _, pkg := range packages {
			if pkgMap, ok := pkg.(map[string]interface{}); ok {
				serverFields := map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{StringValue: serverTitle},
					},
					"type": {
						Kind: &structpb.Value_StringValue{StringValue: "local"},
					},
					"capabilities": {
						Kind: &structpb.Value_ListValue{
							ListValue: &structpb.ListValue{Values: []*structpb.Value{}},
						},
					},
				}

				// Add description
				if serverDescription != "" {
					serverFields["description"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: serverDescription},
					}
				}

				// Build command from runtimeHint and identifier
				if runtimeHint, ok := pkgMap["runtimeHint"].(string); ok {
					serverFields["command"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: runtimeHint},
					}
				}

				// Build args array from runtimeArguments and packageArguments
				var argsValues []*structpb.Value

				// Add runtime arguments
				if runtimeArgs, ok := pkgMap["runtimeArguments"].([]interface{}); ok {
					for _, arg := range runtimeArgs {
						if argMap, ok := arg.(map[string]interface{}); ok {
							if argType, ok := argMap["type"].(string); ok && argType == "named" {
								if name, ok := argMap["name"].(string); ok {
									argsValues = append(argsValues, &structpb.Value{
										Kind: &structpb.Value_StringValue{StringValue: name},
									})
								}
							}
						}
					}
				}

				// Add package identifier
				if identifier, ok := pkgMap["identifier"].(string); ok {
					argsValues = append(argsValues, &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: identifier},
					})
				}

				// Add package arguments
				if packageArgs, ok := pkgMap["packageArguments"].([]interface{}); ok {
					for _, arg := range packageArgs {
						if argMap, ok := arg.(map[string]interface{}); ok {
							if value, ok := argMap["value"].(string); ok {
								argsValues = append(argsValues, &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: value},
								})
							}
						}
					}
				}

				if len(argsValues) > 0 {
					serverFields["args"] = &structpb.Value{
						Kind: &structpb.Value_ListValue{
							ListValue: &structpb.ListValue{Values: argsValues},
						},
					}
				}

				// Add environment variables
				if envVars, ok := pkgMap["environmentVariables"].([]interface{}); ok {
					envVarsValues := make([]*structpb.Value, 0, len(envVars))
					for _, envVar := range envVars {
						if envMap, ok := envVar.(map[string]interface{}); ok {
							envFields := map[string]*structpb.Value{}
							if name, ok := envMap["name"].(string); ok {
								envFields["name"] = &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: name},
								}
							}
							if value, ok := envMap["value"].(string); ok {
								envFields["default_value"] = &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: value},
								}
							}
							if value, ok := envMap["description"].(string); ok {
								envFields["description"] = &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: value},
								}
							}
							if len(envFields) > 0 {
								envVarsValues = append(envVarsValues, &structpb.Value{
									Kind: &structpb.Value_StructValue{
										StructValue: &structpb.Struct{Fields: envFields},
									},
								})
							}
						}
					}
					if len(envVarsValues) > 0 {
						serverFields["env_vars"] = &structpb.Value{
							Kind: &structpb.Value_ListValue{
								ListValue: &structpb.ListValue{Values: envVarsValues},
							},
						}
					}
				}

				mcpServers = append(mcpServers, &structpb.Value{
					Kind: &structpb.Value_StructValue{
						StructValue: &structpb.Struct{Fields: serverFields},
					},
				})
			}
		}
	}

	// Create MCP data structure conforming to OASF v0.8.0 MCP data schema
	mcpModuleData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"servers": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{Values: mcpServers},
				},
			},
		},
	}

	// Create the MCP module with schema-compliant data
	mcpModule := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name": {
				Kind: &structpb.Value_StringValue{StringValue: MCPModuleName},
			},
			"data": {
				Kind: &structpb.Value_StructValue{StructValue: mcpModuleData},
			},
		},
	}

	// Create the modules list
	modulesList := &structpb.ListValue{
		Values: []*structpb.Value{
			{
				Kind: &structpb.Value_StructValue{StructValue: mcpModule},
			},
		},
	}

	// Use websiteUrl or repoUrl for locators
	locatorUrl := repoUrl
	if websiteUrl != "" {
		locatorUrl = websiteUrl
	}

	// Create OASF-compliant record with all required fields
	record := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name": {
				Kind: &structpb.Value_StringValue{StringValue: serverName},
			},
			"schema_version": {
				Kind: &structpb.Value_StringValue{StringValue: targetSchema},
			},
			"version": {
				Kind: &structpb.Value_StringValue{StringValue: serverVersion},
			},
			"description": {
				Kind: &structpb.Value_StringValue{StringValue: serverDescription},
			},
			"authors": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StringValue{StringValue: authors[0]},
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
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"id": {
												Kind: &structpb.Value_NumberValue{NumberValue: 0},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "base_skill"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"locators": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"type": {
												Kind: &structpb.Value_StringValue{StringValue: "source_code"},
											},
											"url": {
												Kind: &structpb.Value_StringValue{StringValue: locatorUrl},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			"domains": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"id": {
												Kind: &structpb.Value_NumberValue{NumberValue: 0},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "base_domain"},
											},
										},
									},
								},
							},
						},
					},
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
