// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	agentv1alpha1 "github.com/kagenti/operator/api/v1alpha1"
)

// Constants for repeated strings.
const (
	serverTypeLocal  = "local"
	serverTypeSSE    = "sse"
	serverTypeHTTP   = "http"
	packageTypeNPM   = "npm"
	packageTypePyPI  = "pypi"
	packageTypeOCI   = "oci"
	packageTypeNuGet = "nuget"
	packageTypeMCPB  = "mcpb"
	defaultVersion   = "v1.0.0"
)

// OASF schema domain and skill IDs.
const (
	domainAgentOrchestration = 1004  // agent_orchestration/agent_coordination
	domainMultiModal         = 703   // multi_modal/any_to_any
	skillAPIIntegration      = 10204 // technology/software_engineering/apis_integration
)

const (
	MCPModuleName = "integration/mcp"
	A2AModuleName = "integration/a2a"
	targetSchema  = "0.8.0"
)

// RecordToGHCopilot translates a record into a GHCopilotMCPConfig structure.
func RecordToGHCopilot(record *structpb.Struct) (*GHCopilotMCPConfig, error) {
	// Get MCP module - try 0.8.0 name first, then fall back to 0.7.0 for backward compatibility
	found, mcpModule := getModuleDataFromRecord(record, MCPModuleName) // "integration/mcp"
	if !found {
		found, mcpModule = getModuleDataFromRecord(record, "runtime/mcp") // 0.7.0 compatibility
	}

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

	for serverName, serverVal := range serversStruct.GetFields() {
		serverMap := serverVal.GetStructValue()
		if serverMap == nil {
			continue
		}

		command, ok := serverMap.GetFields()["command"]
		if !ok {
			return nil, fmt.Errorf("missing 'command' for server '%s'", serverName)
		}

		args := []string{}

		if argsVal, ok := serverMap.GetFields()["args"]; ok {
			for _, arg := range argsVal.GetListValue().GetValues() {
				args = append(args, arg.GetStringValue())
			}
		}

		env := map[string]string{}

		if envVal, ok := serverMap.GetFields()["env"]; ok {
			envStruct := envVal.GetStructValue()
			if envStruct != nil {
				for key, val := range envStruct.GetFields() {
					env[key] = val.GetStringValue()

					if after, ok0 := strings.CutPrefix(val.GetStringValue(), "${input:"); ok0 {
						id := strings.TrimSuffix(after, "}")
						inputs = append(inputs, MCPInput{
							ID:          id,
							Type:        "promptString",
							Password:    true,
							Description: "Secret value for " + id,
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

// RecordToA2A translates a record into an A2A card structure.
// Returns the A2A card data as a structpb.Struct, preserving all fields
// from the A2A protocol definition to prevent schema drift.
func RecordToA2A(record *structpb.Struct) (*structpb.Struct, error) {
	// Get A2A module - try 0.8.0 name first, then fall back to 0.7.0 for backward compatibility
	found, a2aModule := getModuleDataFromRecord(record, A2AModuleName) // "integration/a2a"
	if !found {
		found, a2aModule = getModuleDataFromRecord(record, "runtime/a2a") // 0.7.0 compatibility
	}

	if !found {
		return nil, errors.New("A2A module not found in record")
	}

	if cardDataVal, ok := a2aModule.GetFields()["card_data"]; ok {
		cardData := cardDataVal.GetStructValue()
		if cardData != nil {
			return cardData, nil
		}
	}

	// Fallback: return the module data directly (for records where card data is at the top level)
	return a2aModule, nil
}

// A2AToRecord translates an A2A card data back into an OASF-compliant record format.
func A2AToRecord(a2aData *structpb.Struct) (*structpb.Struct, error) { //nolint:cyclop,maintidx
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
	cardVersion := defaultVersion

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

	if version, ok := cardMap["version"]; ok {
		if versionStr, ok := version.(string); ok && versionStr != "" {
			cardVersion = versionStr
		}
	}

	// Extract authors from provider organization if available
	authors := []string{"Generated by OASF SDK"}

	if provider, ok := cardMap["provider"].(map[string]any); ok {
		if org, ok := provider["organization"].(string); ok && org != "" {
			authors = []string{org}
		}
	}

	// Use current timestamp for created_at (RFC3339 format)
	// Note: For consistent test results, this could be overridden in test fixtures
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Collect A2A URLs and other metadata in annotations
	annotations := extractA2AAnnotations(cardMap)

	// Extract protocol version from card (with fallback)
	// A2A proto uses "protocol_version" but JSON serialization may use "protocolVersion" (camelCase)
	protocolVersion := defaultVersion
	if pv, ok := cardMap["protocolVersion"].(string); ok && pv != "" {
		protocolVersion = pv
	} else if pv, ok := cardMap["protocol_version"].(string); ok && pv != "" {
		protocolVersion = pv
	}

	// Create A2A data structure conforming to OASF v0.8.0 A2A data schema
	A2AModuleData := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"card_data": {
				Kind: &structpb.Value_StructValue{StructValue: A2ACardStruct},
			},
			"protocol_version": {
				Kind: &structpb.Value_StringValue{StringValue: protocolVersion},
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
				Kind: &structpb.Value_StringValue{StringValue: cardVersion},
			},
			"description": {
				Kind: &structpb.Value_StringValue{StringValue: cardDescription},
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
				Kind: &structpb.Value_StringValue{StringValue: createdAt},
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
												Kind: &structpb.Value_NumberValue{NumberValue: domainAgentOrchestration},
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
			"domains": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{
								Kind: &structpb.Value_StructValue{
									StructValue: &structpb.Struct{
										Fields: map[string]*structpb.Value{
											"id": {
												Kind: &structpb.Value_NumberValue{NumberValue: skillAPIIntegration},
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

// MCPToRecord translates an MCP Registry server.json into an OASF-compliant record format.
func MCPToRecord(mcpData *structpb.Struct) (*structpb.Struct, error) { //nolint:gocognit,gocyclo,cyclop,maintidx
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

	if repo, ok := serverMap["repository"].(map[string]any); ok {
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

	// Use current timestamp for created_at (RFC3339 format)
	// Note: For consistent test results, this could be overridden in test fixtures
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Build MCP servers array from packages and/or remotes
	var mcpServers []*structpb.Value

	// Collect important metadata that doesn't fit in OASF structure
	annotations := make(map[string]string)

	// Process packages (local stdio servers)
	if packages, ok := serverMap["packages"].([]any); ok { //nolint:nestif
		for _, pkg := range packages {
			if pkgMap, ok := pkg.(map[string]any); ok {
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
				serverType := serverTypeLocal
				transportUrl := ""

				if transport, ok := pkgMap["transport"].(map[string]any); ok {
					if tType, ok := transport["type"].(string); ok {
						switch tType {
						case serverTypeSSE:
							serverType = serverTypeSSE
						case "streamable-http":
							serverType = serverTypeHTTP
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
					var command string
					if runtimeHint, ok := pkgMap["runtimeHint"].(string); ok {
						command = runtimeHint
					} else {
						// Generate command based on registryType
						switch registryType {
						case packageTypeNPM:
							command = "npx"
						case packageTypePyPI:
							command = "python"
						case packageTypeOCI:
							command = "docker"
						case packageTypeNuGet:
							command = "dotnet"
						case packageTypeMCPB:
							command = "mcpb"
						default:
							command = "echo"
						}
					}

					serverFields["command"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: command},
					}
				} else if transportUrl != "" {
					// For http/sse servers, URL from transport is required
					// If missing, the record will be invalid (url is required for non-local servers)
					serverFields["url"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: transportUrl},
					}
				}

				// Build args array from runtimeArguments and packageArguments (only for local servers)
				if serverType == "local" {
					var argsValues []*structpb.Value

					// Add registry-specific args based on command type (only if no runtimeHint)
					_, hasRuntimeHint := pkgMap["runtimeHint"]
					if !hasRuntimeHint && registryType != "" {
						switch registryType {
						case packageTypePyPI:
							// python -m <package>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "-m"},
							})
						case packageTypeOCI:
							// docker run <image>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "run"},
							})
						case packageTypeNuGet:
							// dotnet tool run <package>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "tool"},
							})
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "run"},
							})
						case packageTypeMCPB:
							// mcpb run <package>
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: "run"},
							})
						case packageTypeNPM:
							// npx <package> (no extra args needed)
						default:
							// echo "command generation for registryType <type> is not yet implemented"
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: fmt.Sprintf("command generation for registryType %s is not yet implemented", registryType)},
							})
						}
					}

					// Add runtime arguments
					if runtimeArgs, ok := pkgMap["runtimeArguments"].([]any); ok {
						for _, arg := range runtimeArgs {
							if argMap, ok := arg.(map[string]any); ok {
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
						case packageTypeNPM:
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
						case packageTypePyPI:
							// pypi: python -m package (version pinned at install time, not at runtime)
							argsValues = append(argsValues, &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: identifier},
							})
						case packageTypeOCI:
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
						case packageTypeNuGet:
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
						case packageTypeMCPB:
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
					if packageArgs, ok := pkgMap["packageArguments"].([]any); ok {
						for _, arg := range packageArgs {
							if argMap, ok := arg.(map[string]any); ok {
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
				if envVars, ok := pkgMap["environmentVariables"].([]any); ok {
					envVarsValues := make([]*structpb.Value, 0, len(envVars))
					for _, envVar := range envVars {
						if envMap, ok := envVar.(map[string]any); ok {
							envFields := map[string]*structpb.Value{}

							// Skip env vars with empty or missing name (invalid)
							name, hasName := envMap["name"].(string)
							if !hasName || name == "" {
								continue
							}

							envFields["name"] = &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: name},
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
							description := ""
							if desc, ok := envMap["description"].(string); ok && desc != "" {
								description = desc
							} else if variables, ok := envMap["variables"].(map[string]any); ok {
								// Check for description in variables.{variable_name}.description
								// Extract variable name from value (e.g., "{weather_choices}" -> "weather_choices")
								if value, ok := envMap["value"].(string); ok && len(value) > 2 && value[0] == '{' && value[len(value)-1] == '}' {
									varKey := value[1 : len(value)-1]
									if varDef, ok := variables[varKey].(map[string]any); ok {
										if desc, ok := varDef["description"].(string); ok && desc != "" {
											description = desc
										}
									}
								}
							}

							// OASF requires description - provide default if missing
							if description == "" {
								description = "Environment variable: " + name
							}

							envFields["description"] = &structpb.Value{
								Kind: &structpb.Value_StringValue{StringValue: description},
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
	if remotes, ok := serverMap["remotes"].([]any); ok { //nolint:nestif
		for _, remote := range remotes {
			if remoteMap, ok := remote.(map[string]any); ok {
				// Determine server type from remote type
				remoteType := "http"

				if rType, ok := remoteMap["type"].(string); ok {
					switch rType {
					case serverTypeSSE:
						remoteType = serverTypeSSE
					case "streamable-http":
						remoteType = serverTypeHTTP
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
				if headers, ok := remoteMap["headers"].([]any); ok && len(headers) > 0 {
					headersMap := &structpb.Struct{Fields: map[string]*structpb.Value{}}

					for _, header := range headers {
						if headerMap, ok := header.(map[string]any); ok {
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

					if len(headersMap.GetFields()) > 0 {
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
				Kind: &structpb.Value_StringValue{StringValue: createdAt},
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
												Kind: &structpb.Value_NumberValue{NumberValue: domainMultiModal},
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
												Kind: &structpb.Value_NumberValue{NumberValue: skillAPIIntegration},
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
	// Find module by exact name match
	modules, ok := record.GetFields()["modules"]
	if !ok {
		return false, nil
	}

	for _, module := range modules.GetListValue().GetValues() {
		moduleStruct := module.GetStructValue()
		if moduleStruct == nil {
			continue
		}

		nameField := moduleStruct.GetFields()["name"]
		if nameField == nil {
			continue
		}

		// Exact match required (e.g., "integration/a2a")
		if nameField.GetStringValue() == moduleName {
			return true, moduleStruct.GetFields()["data"].GetStructValue()
		}
	}

	return false, nil
}

// extractA2AAnnotations extracts A2A card URLs and metadata into annotations.
// These don't map cleanly to OASF locators, so we store them as annotations.
func extractA2AAnnotations(cardMap map[string]any) map[string]string { //nolint:gocognit,nestif,cyclop,gocyclo
	annotations := make(map[string]string)

	// Store deprecated URL field if present
	if url, ok := cardMap["url"]; ok {
		if urlStr, ok := url.(string); ok && urlStr != "" {
			annotations["a2a.url"] = urlStr
		}
	}

	// Store supported_interfaces URLs
	// Note: A2A proto uses "supported_interfaces" but JSON may use "supportedInterfaces"
	var interfaces []any
	if ifaces, ok := cardMap["supportedInterfaces"].([]any); ok {
		interfaces = ifaces
	} else if ifaces, ok := cardMap["supported_interfaces"].([]any); ok {
		interfaces = ifaces
	}

	if len(interfaces) > 0 { //nolint:nestif
		for i, iface := range interfaces {
			if ifaceMap, ok := iface.(map[string]any); ok {
				if url, ok := ifaceMap["url"].(string); ok && url != "" {
					annotations[fmt.Sprintf("a2a.interface.%d.url", i)] = url
				}

				// Check both camelCase and snake_case for protocol_binding
				var protocolBinding string
				if pb, ok := ifaceMap["protocolBinding"].(string); ok && pb != "" {
					protocolBinding = pb
				} else if pb, ok := ifaceMap["protocol_binding"].(string); ok && pb != "" {
					protocolBinding = pb
				}

				if protocolBinding != "" {
					annotations[fmt.Sprintf("a2a.interface.%d.protocol_binding", i)] = protocolBinding
				}
			}
		}
	}

	// Store provider information
	if provider, ok := cardMap["provider"].(map[string]any); ok { //nolint:nestif
		if providerURL, ok := provider["url"].(string); ok && providerURL != "" {
			annotations["a2a.provider.url"] = providerURL
		}

		if org, ok := provider["organization"].(string); ok && org != "" {
			annotations["a2a.provider.organization"] = org
		}

		// Store provider extensions - important for protocol capability discovery
		if extensions, ok := provider["extensions"].([]any); ok {
			for i, ext := range extensions {
				if extMap, ok := ext.(map[string]any); ok {
					if uri, ok := extMap["uri"].(string); ok && uri != "" {
						annotations[fmt.Sprintf("a2a.provider.extension.%d.uri", i)] = uri
					}

					if desc, ok := extMap["description"].(string); ok && desc != "" {
						annotations[fmt.Sprintf("a2a.provider.extension.%d.description", i)] = desc
					}

					if required, ok := extMap["required"].(bool); ok {
						annotations[fmt.Sprintf("a2a.provider.extension.%d.required", i)] = strconv.FormatBool(required)
					}
				}
			}
		}
	}

	// Store documentation URL
	// Note: A2A proto uses "documentation_url" but JSON may use "documentationUrl"
	var docURL string
	if url, ok := cardMap["documentationUrl"].(string); ok && url != "" {
		docURL = url
	} else if url, ok := cardMap["documentation_url"].(string); ok && url != "" {
		docURL = url
	}

	if docURL != "" {
		annotations["a2a.documentation_url"] = docURL
	}

	// Store icon URL if present
	// Note: A2A proto uses "icon_url" but JSON may use "iconUrl"
	var iconURL string
	if url, ok := cardMap["iconUrl"].(string); ok && url != "" {
		iconURL = url
	} else if url, ok := cardMap["icon_url"].(string); ok && url != "" {
		iconURL = url
	}

	if iconURL != "" {
		annotations["a2a.icon_url"] = iconURL
	}

	// Store authenticated extended card support flag
	// Note: A2A proto uses "supports_authenticated_extended_card" but JSON may use "supportsAuthenticatedExtendedCard"
	var supportsAuth bool

	var hasAuthFlag bool

	if auth, ok := cardMap["supportsAuthenticatedExtendedCard"].(bool); ok {
		supportsAuth = auth
		hasAuthFlag = true
	} else if auth, ok := cardMap["supports_authenticated_extended_card"].(bool); ok {
		supportsAuth = auth
		hasAuthFlag = true
	}

	if hasAuthFlag {
		annotations["a2a.supports_authenticated_extended_card"] = strconv.FormatBool(supportsAuth)
	}

	return annotations
}

// RecordToKagentiAgentSpec translates an OASF record into a Kubernetes Agent CRD structure for kagenti-operator.
// The record must have a locator of type "docker_image" for this to work.
// The exposed port can be specified via locator annotations (e.g., "kagenti.io/port" or "port")
// or will default to 8080 if not specified.
func RecordToKagentiAgentSpec(record *structpb.Struct) (*agentv1alpha1.Agent, error) {
	// Extract basic record metadata
	recordName := "agent"
	if nameVal, ok := record.GetFields()["name"]; ok {
		recordName = nameVal.GetStringValue()
	}

	description := ""
	if descVal, ok := record.GetFields()["description"]; ok {
		description = descVal.GetStringValue()
	}

	version := defaultVersion
	if versionVal, ok := record.GetFields()["version"]; ok {
		version = versionVal.GetStringValue()
	}

	// Extract env_vars from record annotations (for agent container)
	// Since env_vars is not a top-level field in OASF 0.8.0 schema, we use annotations
	// Format: kagenti.io/env-var.<name> = <value>
	var recordEnvVars []corev1.EnvVar
	if annotationsVal, ok := record.GetFields()["annotations"]; ok {
		annotationsStruct := annotationsVal.GetStructValue()
		if annotationsStruct != nil {
			envVarPrefix := "kagenti.io/env-var."
			for key, val := range annotationsStruct.GetFields() {
				if strings.HasPrefix(key, envVarPrefix) {
					envName := strings.TrimPrefix(key, envVarPrefix)
					envValue := val.GetStringValue()
					if envName != "" {
						recordEnvVars = append(recordEnvVars, corev1.EnvVar{
							Name:  envName,
							Value: envValue,
						})
					}
				}
			}
		}
	}

	// Extract docker-image locator
	locatorsVal, ok := record.GetFields()["locators"]
	if !ok {
		return nil, errors.New("record does not contain 'locators' field")
	}

	locatorsList := locatorsVal.GetListValue()
	if locatorsList == nil {
		return nil, errors.New("'locators' is not a list")
	}

	var dockerImageLocator *structpb.Struct
	for _, locatorVal := range locatorsList.GetValues() {
		locatorStruct := locatorVal.GetStructValue()
		if locatorStruct == nil {
			continue
		}

		typeVal, ok := locatorStruct.GetFields()["type"]
		if !ok {
			continue
		}

		if typeVal.GetStringValue() == "docker_image" {
			dockerImageLocator = locatorStruct
			break
		}
	}

	if dockerImageLocator == nil {
		return nil, errors.New("no locator of type 'docker_image' found in record")
	}

	// Extract image URL from docker_image locator
	urlVal, ok := dockerImageLocator.GetFields()["url"]
	if !ok {
		return nil, errors.New("docker_image locator missing 'url' field")
	}

	imageName := urlVal.GetStringValue()
	if imageName == "" {
		return nil, errors.New("docker_image locator 'url' is empty")
	}

	// Extract port from locator annotations or use default
	port := int32(8080) // default port
	locatorAnnotations := dockerImageLocator.GetFields()["annotations"]
	if locatorAnnotations != nil {
		locatorAnnotationsStruct := locatorAnnotations.GetStructValue()
		if locatorAnnotationsStruct != nil {
			// Try kagenti.io/port first
			if portVal, ok := locatorAnnotationsStruct.GetFields()["kagenti.io/port"]; ok {
				if portStr := portVal.GetStringValue(); portStr != "" {
					if parsedPort, err := strconv.ParseInt(portStr, 10, 32); err == nil {
						port = int32(parsedPort)
					}
				}
			} else if portVal, ok := locatorAnnotationsStruct.GetFields()["port"]; ok {
				// Fallback to "port" annotation
				if portStr := portVal.GetStringValue(); portStr != "" {
					if parsedPort, err := strconv.ParseInt(portStr, 10, 32); err == nil {
						port = int32(parsedPort)
					}
				}
			}
		}
	}

	// Try to extract port from A2A module transport URL if port not found in annotations
	if port == 8080 {
		_, a2aModule := getModuleDataFromRecord(record, A2AModuleName)
		if a2aModule == nil {
			_, a2aModule = getModuleDataFromRecord(record, "runtime/a2a")
		}
		if a2aModule != nil {
			// Check for transport URL in A2A module
			// A2A modules may have transport information with URLs like "http://host:port"
			transportsVal, ok := a2aModule.GetFields()["transports"]
			if ok {
				transportsList := transportsVal.GetListValue()
				if transportsList != nil {
					for _, transportVal := range transportsList.GetValues() {
						transportStr := transportVal.GetStringValue()
						// Try to parse URL to extract port
						// Format: "http://host:port" or "https://host:port"
						if strings.HasPrefix(transportStr, "http://") || strings.HasPrefix(transportStr, "https://") {
							// Extract port from URL if present
							parts := strings.Split(transportStr, ":")
							if len(parts) >= 3 {
								// Last part might be port or path
								portPart := parts[len(parts)-1]
								// Remove any path after port
								portPart = strings.Split(portPart, "/")[0]
								if parsedPort, err := strconv.ParseInt(portPart, 10, 32); err == nil {
									port = int32(parsedPort)
									break
								}
							}
						}
					}
				}
			}
		}
	}

	// Replicas is a runtime configuration, not static OASF data
	// Leave it empty so it can be set when deployed
	var replicas *int32 // nil means use default from CRD (which is 1)

	// Namespace is a runtime configuration, not static OASF data
	// Leave it blank so it can be set when deployed
	namespace := ""

	// Build labels
	labels := make(map[string]string)
	labels["app.kubernetes.io/name"] = recordName
	labels["kagenti.io/type"] = "agent"
	if version != "" {
		labels["app.kubernetes.io/version"] = version
	}

	// Extract protocol from A2A module if present
	_, a2aModule := getModuleDataFromRecord(record, A2AModuleName)
	if a2aModule == nil {
		_, a2aModule = getModuleDataFromRecord(record, "runtime/a2a")
	}
	if a2aModule != nil {
		labels["kagenti.io/agent-protocol"] = "a2a"
	}

	// Build annotations from record annotations
	// Exclude kagenti.io/env-var.* annotations as they're only used to extract env vars
	recordAnnotations := make(map[string]string)
	if recordAnnotsVal, ok := record.GetFields()["annotations"]; ok {
		recordAnnotsStruct := recordAnnotsVal.GetStructValue()
		if recordAnnotsStruct != nil {
			envVarPrefix := "kagenti.io/env-var."
			for k, v := range recordAnnotsStruct.GetFields() {
				// Skip env-var annotations - they're only used to extract env vars, not for CRD metadata
				if !strings.HasPrefix(k, envVarPrefix) {
					recordAnnotations[k] = v.GetStringValue()
				}
			}
		}
	}

	// Check for MCP module to determine if we need a tool sidecar container
	var toolImage string
	var toolPort int32
	var mcpServerEnvVars []corev1.EnvVar
	hasMCPModule := false
	found, mcpModule := getModuleDataFromRecord(record, MCPModuleName)
	if found && mcpModule != nil {
		hasMCPModule = true
		// Extract tool container image and port from MCP module annotations
		// Check module-level annotations first
		modulesVal, ok := record.GetFields()["modules"]
		if ok {
			modulesList := modulesVal.GetListValue()
			if modulesList != nil {
				for _, moduleVal := range modulesList.GetValues() {
					moduleStruct := moduleVal.GetStructValue()
					if moduleStruct == nil {
						continue
					}
					nameField := moduleStruct.GetFields()["name"]
					if nameField != nil && nameField.GetStringValue() == MCPModuleName {
						// Found the MCP module, check its annotations
						moduleAnnotsVal, ok := moduleStruct.GetFields()["annotations"]
						if ok {
							moduleAnnotsStruct := moduleAnnotsVal.GetStructValue()
							if moduleAnnotsStruct != nil {
								// Extract tool image
								if toolImageVal, ok := moduleAnnotsStruct.GetFields()["kagenti.io/tool-image"]; ok {
									toolImage = toolImageVal.GetStringValue()
								} else if toolImageVal, ok := moduleAnnotsStruct.GetFields()["tool-image"]; ok {
									toolImage = toolImageVal.GetStringValue()
								}
								// Extract tool port
								if toolPortVal, ok := moduleAnnotsStruct.GetFields()["kagenti.io/tool-port"]; ok {
									if toolPortStr := toolPortVal.GetStringValue(); toolPortStr != "" {
										if parsedPort, err := strconv.ParseInt(toolPortStr, 10, 32); err == nil {
											toolPort = int32(parsedPort)
										}
									}
								} else if toolPortVal, ok := moduleAnnotsStruct.GetFields()["tool-port"]; ok {
									if toolPortStr := toolPortVal.GetStringValue(); toolPortStr != "" {
										if parsedPort, err := strconv.ParseInt(toolPortStr, 10, 32); err == nil {
											toolPort = int32(parsedPort)
										}
									}
								}
							}
						}

						// Extract env_vars from MCP servers
						// MCP module data contains servers array
						serversVal, ok := mcpModule.GetFields()["servers"]
						if ok {
							serversList := serversVal.GetListValue()
							if serversList != nil {
								for _, serverVal := range serversList.GetValues() {
									serverStruct := serverVal.GetStructValue()
									if serverStruct == nil {
										continue
									}
									// Extract env_vars from this server
									envVarsVal, ok := serverStruct.GetFields()["env_vars"]
									if ok {
										envVarsList := envVarsVal.GetListValue()
										if envVarsList != nil {
											for _, envVarVal := range envVarsList.GetValues() {
												envVarStruct := envVarVal.GetStructValue()
												if envVarStruct == nil {
													continue
												}
												// Extract name and default_value from env_var
												var envName, envValue string
												if nameVal, ok := envVarStruct.GetFields()["name"]; ok {
													envName = nameVal.GetStringValue()
												}
												if defaultValueVal, ok := envVarStruct.GetFields()["default_value"]; ok {
													envValue = defaultValueVal.GetStringValue()
												}
												if envName != "" {
													// Add all env_vars from MCP servers to tool container
													mcpServerEnvVars = append(mcpServerEnvVars, corev1.EnvVar{
														Name:  envName,
														Value: envValue,
													})
												}
											}
										}
									}
								}
							}
						}
						break
					}
				}
			}
		}
	}

	// Build containers list
	containers := []corev1.Container{
		{
			Name:  "agent",
			Image: imageName,
			Ports: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: port,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
	}

	// Add volume mounts and env vars to agent container if MCP module is present
	if hasMCPModule && toolImage != "" && toolPort > 0 {
		containers[0].VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "tmp",
				MountPath: "/tmp",
			},
		}
		// Set MCP_URL environment variable for agent container
		// Use value from record env_vars if present, otherwise construct default
		mcpURL := ""
		// Check if MCP_URL is in record env_vars
		for _, envVar := range recordEnvVars {
			if envVar.Name == "MCP_URL" {
				mcpURL = envVar.Value
				break
			}
		}
		// Default if not found in record env_vars
		if mcpURL == "" {
			// Default format: http://<service-name>:<port>/mcp
			// The service name is the Agent CRD name (recordName)
			mcpURL = fmt.Sprintf("http://%s:%d/mcp", recordName, toolPort)
		}
		// Add all record env_vars to agent container, ensuring MCP_URL is set
		agentEnvVars := make([]corev1.EnvVar, 0, len(recordEnvVars)+1)
		agentEnvVars = append(agentEnvVars, recordEnvVars...)
		// Add MCP_URL if not already present
		hasMCPURL := false
		for _, envVar := range agentEnvVars {
			if envVar.Name == "MCP_URL" {
				hasMCPURL = true
				break
			}
		}
		if !hasMCPURL {
			agentEnvVars = append(agentEnvVars, corev1.EnvVar{
				Name:  "MCP_URL",
				Value: mcpURL,
			})
		}
		containers[0].Env = agentEnvVars
	} else if len(recordEnvVars) > 0 {
		// Add record env_vars even if no MCP module
		containers[0].Env = recordEnvVars
	}

	// Add tool container if MCP module with tool image is present
	var volumes []corev1.Volume
	if hasMCPModule && toolImage != "" {
		// Add tool container with env vars from MCP server
		toolContainer := corev1.Container{
			Name:  "tool",
			Image: toolImage,
			Ports: []corev1.ContainerPort{
				{
					Name:          "http",
					ContainerPort: toolPort,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Env: mcpServerEnvVars, // Add env vars from MCP server objects
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "tmp",
					MountPath: "/tmp",
				},
			},
		}
		containers = append(containers, toolContainer)

		// Add shared volume
		volumes = []corev1.Volume{
			{
				Name: "tmp",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		}
	}

	// Build service ports
	// targetPort can be omitted - Kubernetes will default to the same value as port
	servicePorts := []corev1.ServicePort{
		{
			Name:     "agent",
			Port:     port,
			Protocol: corev1.ProtocolTCP,
		},
	}

	// Add tool service port if tool container is present
	if hasMCPModule && toolImage != "" && toolPort > 0 {
		servicePorts = append(servicePorts, corev1.ServicePort{
			Name:     "tool",
			Port:     toolPort,
			Protocol: corev1.ProtocolTCP,
		})
	}

	// Create Agent structure using kagenti-operator types
	agent := &agentv1alpha1.Agent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "agent.kagenti.dev/v1alpha1",
			Kind:       "Agent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        recordName,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: recordAnnotations,
		},
		Spec: agentv1alpha1.AgentSpec{
			Description: description,
			Replicas:    replicas, // nil means use CRD default
			ImageSource: agentv1alpha1.ImageSource{
				Image: &imageName,
			},
			ServicePorts: servicePorts,
			MetadataSpec: agentv1alpha1.MetadataSpec{
				Labels:      labels,
				Annotations: recordAnnotations,
			},
			// PodTemplateSpec is required but can be minimal - will be populated by the operator
			// Note: The CRD doesn't support metadata.name or metadata.labels in PodTemplateSpec
			PodTemplateSpec: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: containers,
					Volumes:    volumes,
				},
			},
		},
	}

	return agent, nil
}
