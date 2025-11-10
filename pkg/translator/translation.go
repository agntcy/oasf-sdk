// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
												Kind: &structpb.Value_NumberValue{NumberValue: 1004},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "agent_orchestration/agent_coordination"},
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
												Kind: &structpb.Value_NumberValue{NumberValue: 10204},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "technology/software_engineering/apis_integration"},
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
	serverDescription := "Agent generated from MCP server JSON"
	serverVersion := "v1.0.0"
	repoUrl := "https://example.com/mcp-server.git"
	authors := []string{"Generated by OASF SDK"}

	if name, ok := serverMap["name"].(string); ok {
		serverName = name
	}
	if description, ok := serverMap["description"].(string); ok {
		serverDescription = description
	}
	if version, ok := serverMap["version"].(string); ok {
		serverVersion = version
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

	// Build MCP servers array from packages and/or remotes
	var mcpServers []*structpb.Value

	// Collect important metadata that doesn't fit in OASF structure
	annotations := make(map[string]string)

	// Process packages (local stdio servers)
	if packages, ok := serverMap["packages"].([]interface{}); ok {
		for _, pkg := range packages {
			if pkgMap, ok := pkg.(map[string]interface{}); ok {
				// Extract metadata early for use throughout
				registryType := ""
				identifier := ""

				if rt, ok := pkgMap["registryType"].(string); ok {
					registryType = rt
				}
				if id, ok := pkgMap["identifier"].(string); ok {
					identifier = id
				}

				// Add important package metadata to annotations
				if identifier != "" {
					// fileSha256 - important for security/integrity verification
					if fileSha256, ok := pkgMap["fileSha256"].(string); ok && fileSha256 != "" {
						annotations[fmt.Sprintf("mcp.package.%s.fileSha256", identifier)] = fileSha256
					}

					// registryType - useful for reconstructing package source
					if registryType != "" {
						annotations[fmt.Sprintf("mcp.package.%s.registryType", identifier)] = registryType
					}

					// registryBaseUrl - needed to reconstruct full package URL
					if registryBaseUrl, ok := pkgMap["registryBaseUrl"].(string); ok && registryBaseUrl != "" {
						annotations[fmt.Sprintf("mcp.package.%s.registryBaseUrl", identifier)] = registryBaseUrl
					}

					// package version - useful for tracking
					if pkgVersion, ok := pkgMap["version"].(string); ok && pkgVersion != "" {
						annotations[fmt.Sprintf("mcp.package.%s.version", identifier)] = pkgVersion
					}
				}

				// Determine server type from transport and extract URL if present
				serverType := "local"
				transportUrl := ""
				if transport, ok := pkgMap["transport"].(map[string]interface{}); ok {
					if tType, ok := transport["type"].(string); ok {
						if tType == "sse" {
							serverType = "sse"
						} else if tType == "streamable-http" {
							serverType = "http"
						}
						// stdio is local (default)
					}
					// Extract URL from transport if it's an HTTP/SSE transport
					if url, ok := transport["url"].(string); ok {
						transportUrl = url
					}
				}

				serverFields := map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{StringValue: serverName},
					},
					"type": {
						Kind: &structpb.Value_StringValue{StringValue: serverType},
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

				// For local servers, build command from package metadata (required!)
				// For http/sse servers, add url from websiteUrl or construct from registry
				if serverType == "local" {
					// Build command: use runtimeHint if available, otherwise generate from registryType
					command := ""
					if runtimeHint, ok := pkgMap["runtimeHint"].(string); ok {
						command = runtimeHint
					} else {
						// Generate command based on registryType
						switch registryType {
						case "npm":
							command = "npx"
						case "pypi":
							command = "python"
						case "oci":
							command = "docker"
						case "nuget":
							command = "dotnet"
						case "mcpb":
							command = "mcpb"
						default:
							command = "echo"
						}
					}

					serverFields["command"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: command},
					}
				} else {
					// For http/sse servers, URL from transport is required
					// If missing, the record will be invalid (url is required for non-local servers)
					if transportUrl != "" {
						serverFields["url"] = &structpb.Value{
							Kind: &structpb.Value_StringValue{StringValue: transportUrl},
						}
					}
				}

				// Build args array from runtimeArguments and packageArguments (only for local servers)
				if serverType == "local" {
					var argsValues []*structpb.Value

					// Add registry-specific args based on command type (only if no runtimeHint)
					_, hasRuntimeHint := pkgMap["runtimeHint"]
					if !hasRuntimeHint && registryType != "" {
						switch registryType {
						case "pypi":
							// python -m <package>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "-m"},
							})
						case "oci":
							// docker run <image>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "run"},
							})
						case "nuget":
							// dotnet tool run <package>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "tool"},
							})
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "run"},
							})
						case "mcpb":
							// mcpb run <package>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "run"},
							})
						case "npm":
							// npx <package> (no extra args needed)
						default:
							// echo "command generation for registryType <type> is not yet implemented"
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: fmt.Sprintf("command generation for registryType %s is not yet implemented", registryType)},
							})
						}
					}

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

					// Add package identifier with version where appropriate
					if identifier != "" && registryType != "" {
						pkgVersion := ""
						if v, ok := pkgMap["version"].(string); ok {
							pkgVersion = v
						}

						// For some registry types, append version to the identifier
						switch registryType {
						case "npm":
							// npm: npx package@version
							if pkgVersion != "" && !hasRuntimeHint {
								argsValues = append(argsValues, &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: fmt.Sprintf("%s@%s", identifier, pkgVersion)},
								})
							} else {
								argsValues = append(argsValues, &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: identifier},
								})
							}
						case "pypi":
							// pypi: python -m package (version pinned at install time, not at runtime)
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: identifier},
							})
						case "oci":
							// oci: docker run image:version
							if pkgVersion != "" && !hasRuntimeHint {
								argsValues = append(argsValues, &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: fmt.Sprintf("%s:%s", identifier, pkgVersion)},
								})
							} else {
								argsValues = append(argsValues, &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: identifier},
								})
							}
						case "nuget":
							// nuget: dotnet tool run package --version x.x.x
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: identifier},
							})
							if pkgVersion != "" && !hasRuntimeHint {
								argsValues = append(argsValues, &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: "--version"},
								})
								argsValues = append(argsValues, &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: pkgVersion},
								})
							}
						case "mcpb":
							// mcpb: mcpb run package (version typically in URL)
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: identifier},
							})
						default:
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: identifier},
							})
						}
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
							// Try "value" first, then "default"
							if value, ok := envMap["value"].(string); ok {
								envFields["default_value"] = &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: value},
								}
							} else if defaultVal, ok := envMap["default"].(string); ok {
								envFields["default_value"] = &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: defaultVal},
								}
							}
							// Try direct description first
							if description, ok := envMap["description"].(string); ok {
								envFields["description"] = &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: description},
								}
							} else if variables, ok := envMap["variables"].(map[string]interface{}); ok {
								// Check for description in variables.{variable_name}.description
								// Extract variable name from value (e.g., "{weather_choices}" -> "weather_choices")
								if value, ok := envMap["value"].(string); ok && len(value) > 2 && value[0] == '{' && value[len(value)-1] == '}' {
									varKey := value[1 : len(value)-1]
									if varDef, ok := variables[varKey].(map[string]interface{}); ok {
										if desc, ok := varDef["description"].(string); ok {
											envFields["description"] = &structpb.Value{
												Kind: &structpb.Value_StringValue{StringValue: desc},
											}
										}
									}
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

	// Process remotes (HTTP/SSE servers)
	if remotes, ok := serverMap["remotes"].([]interface{}); ok {
		for _, remote := range remotes {
			if remoteMap, ok := remote.(map[string]interface{}); ok {
				// Determine server type from remote type
				remoteType := "http"
				if rType, ok := remoteMap["type"].(string); ok {
					if rType == "sse" {
						remoteType = "sse"
					} else if rType == "streamable-http" {
						remoteType = "http"
					}
				}

				serverFields := map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{StringValue: serverName},
					},
					"type": {
						Kind: &structpb.Value_StringValue{StringValue: remoteType},
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

				// Add URL (required for http/sse)
				if url, ok := remoteMap["url"].(string); ok {
					serverFields["url"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: url},
					}
				}

				// Add headers if present
				if headers, ok := remoteMap["headers"].([]interface{}); ok && len(headers) > 0 {
					headersMap := &structpb.Struct{Fields: map[string]*structpb.Value{}}
					for _, header := range headers {
						if headerMap, ok := header.(map[string]interface{}); ok {
							if name, ok := headerMap["name"].(string); ok {
								headerValue := ""
								if val, ok := headerMap["value"].(string); ok {
									headerValue = val
								} else if _, ok := headerMap["description"].(string); ok {
									// Use description as a placeholder if no value
									headerValue = "{" + strings.ToLower(strings.ReplaceAll(name, " ", "_")) + "}"
								}
								headersMap.Fields[name] = &structpb.Value{
									Kind: &structpb.Value_StringValue{StringValue: headerValue},
								}
							}
						}
					}
					if len(headersMap.Fields) > 0 {
						serverFields["headers"] = &structpb.Value{
							Kind: &structpb.Value_StructValue{StructValue: headersMap},
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
					Values: func() []*structpb.Value {
						authorValues := make([]*structpb.Value, 0, len(authors))
						for _, author := range authors {
							authorValues = append(authorValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: author},
							})
						}
						return authorValues
					}(),
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
												Kind: &structpb.Value_NumberValue{NumberValue: 703},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "multi_modal/any_to_any"},
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
												Kind: &structpb.Value_StringValue{StringValue: repoUrl},
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
												Kind: &structpb.Value_NumberValue{NumberValue: 10204},
											},
											"name": {
												Kind: &structpb.Value_StringValue{StringValue: "technology/software_engineering/apis_integration"},
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

	// Add annotations if any exist
	if len(annotations) > 0 {
		annotationFields := make(map[string]*structpb.Value)
		for k, v := range annotations {
			annotationFields[k] = &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: v},
			}
		}
		record.Fields["annotations"] = &structpb.Value{
			Kind: &structpb.Value_StructValue{
				StructValue: &structpb.Struct{Fields: annotationFields},
			},
		}
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
