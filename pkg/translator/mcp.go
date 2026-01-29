// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
)

// processMCPServer processes a single MCP server struct (0.7.0/0.8.0 format) and extracts its configuration.
func processMCPServer(serverMap *structpb.Struct, serverName string, inputs *[]MCPInput) (MCPServer, error) {
	command, ok := serverMap.GetFields()["command"]
	if !ok {
		return MCPServer{}, fmt.Errorf("missing 'command' for server '%s'", serverName)
	}

	args := []string{}
	env := map[string]string{}

	if argsVal, ok := serverMap.GetFields()["args"]; ok {
		argsList := argsVal.GetListValue().GetValues()

		rawArgs := make([]string, 0, len(argsList))
		for _, argVal := range argsList {
			rawArgs = append(rawArgs, argVal.GetStringValue())
		}

		// Process args to extract environment variables (e.g., "-e VAR_NAME")
		for i := 0; i < len(rawArgs); i++ {
			arg := rawArgs[i]
			if arg == "-e" && i+1 < len(rawArgs) {
				// Found environment variable flag
				envVarName := rawArgs[i+1]
				// Add env var to env map with input reference
				env[envVarName] = "${input:" + envVarName + "}"

				// Add input for this environment variable
				*inputs = append(*inputs, MCPInput{
					ID:          envVarName,
					Type:        "promptString",
					Password:    true,
					Description: "Secret value for " + envVarName,
				})

				// Keep both "-e" and the env var name in args (for docker compatibility)
				args = append(args, arg, envVarName)
				i++ // Skip the env var name in next iteration
			} else {
				args = append(args, arg)
			}
		}
	}

	// Also check for explicit env field (if present)
	if envVal, ok := serverMap.GetFields()["env"]; ok {
		envStruct := envVal.GetStructValue()
		if envStruct != nil {
			for key, val := range envStruct.GetFields() {
				value := val.GetStringValue()
				env[key] = value

				// Only create input if it's an input reference (not a literal value)
				if after, ok0 := strings.CutPrefix(value, "${input:"); ok0 {
					id := strings.TrimSuffix(after, "}")
					addInputIfNotExists(inputs, id)
				}
			}
		}
	}

	return MCPServer{
		Command: command.GetStringValue(),
		Args:    args,
		Env:     env,
	}, nil
}

// addInputIfNotExists adds an input to the inputs slice if it doesn't already exist.
func addInputIfNotExists(inputs *[]MCPInput, id string) {
	for _, input := range *inputs {
		if input.ID == id {
			return
		}
	}

	*inputs = append(*inputs, MCPInput{
		ID:          id,
		Type:        "promptString",
		Password:    true,
		Description: "Secret value for " + id,
	})
}

// processEnvVar processes a single env_var object and adds it to env map and inputs if needed.
func processEnvVar(envVarMap *structpb.Struct, env map[string]string, inputs *[]MCPInput) {
	nameVal, ok := envVarMap.GetFields()["name"]
	if !ok {
		return
	}

	envName := nameVal.GetStringValue()

	// Get default_value if present
	valueVal, hasDefaultValue := envVarMap.GetFields()["default_value"]

	value := ""
	if hasDefaultValue {
		value = valueVal.GetStringValue()
	}

	// If default_value exists and is not empty, use it directly
	if hasDefaultValue && value != "" {
		env[envName] = value

		// Only create input if it's an input reference
		if after, ok0 := strings.CutPrefix(value, "${input:"); ok0 {
			id := strings.TrimSuffix(after, "}")
			addInputIfNotExists(inputs, id)
		}
	} else {
		// No default value, create input reference for required env
		env[envName] = "${input:" + envName + "}"
		addInputIfNotExists(inputs, envName)
	}
}

// processMCPConnection processes a single MCP connection (1.0.0 format) and extracts its configuration.
func processMCPConnection(connectionMap *structpb.Struct, inputs *[]MCPInput) (MCPServer, error) {
	commandVal, hasCommand := connectionMap.GetFields()["command"]
	if !hasCommand {
		return MCPServer{}, errors.New("missing 'command' in connection")
	}

	args := []string{}

	if argsVal, ok := connectionMap.GetFields()["args"]; ok {
		for _, arg := range argsVal.GetListValue().GetValues() {
			args = append(args, arg.GetStringValue())
		}
	}

	env := map[string]string{}

	// In 1.0.0, env_vars is an array of env_var objects, not a map
	if envVarsVal, ok := connectionMap.GetFields()["env_vars"]; ok {
		envVarsList := envVarsVal.GetListValue()
		if envVarsList != nil {
			for _, envVarVal := range envVarsList.GetValues() {
				envVarMap := envVarVal.GetStructValue()
				if envVarMap == nil {
					continue
				}

				processEnvVar(envVarMap, env, inputs)
			}
		}
	}

	return MCPServer{
		Command: commandVal.GetStringValue(),
		Args:    args,
		Env:     env,
	}, nil
}

// processMCPModule100 processes MCP module data in 1.0.0 format (mcp_data with connections).
func processMCPModule100(mcpModule *structpb.Struct, servers map[string]MCPServer, inputs *[]MCPInput) error {
	nameVal, ok := mcpModule.GetFields()["name"]
	if !ok {
		return errors.New("missing 'name' in MCP module data (1.0.0 format)")
	}

	originalName := nameVal.GetStringValue()
	// Normalize server name for GH Copilot config
	serverName := normalizeServerName(originalName)

	connectionsVal, ok := mcpModule.GetFields()["connections"]
	if !ok {
		return errors.New("invalid or missing 'connections' in MCP module data (1.0.0 format)")
	}

	connectionsList := connectionsVal.GetListValue()
	if connectionsList == nil {
		return errors.New("'connections' must be an array")
	}

	// Process each connection - for GH Copilot, we only support stdio connections
	for _, connectionVal := range connectionsList.GetValues() {
		connectionMap := connectionVal.GetStructValue()
		if connectionMap == nil {
			continue
		}

		// Check connection type - GH Copilot only supports stdio
		typeVal, ok := connectionMap.GetFields()["type"]
		if !ok {
			continue
		}

		connectionType := typeVal.GetStringValue()
		if connectionType != "stdio" {
			// Skip non-stdio connections for GH Copilot
			continue
		}

		server, err := processMCPConnection(connectionMap, inputs)
		if err != nil {
			return fmt.Errorf("failed to process connection: %w", err)
		}

		servers[serverName] = server
		// GH Copilot config only supports one connection per server name
		break
	}

	return nil
}

// normalizeServerName normalizes server names for GH Copilot config (e.g., "github-mcp-server" -> "github").
func normalizeServerName(name string) string {
	// Remove common suffixes
	name = strings.TrimSuffix(name, "-mcp-server")
	name = strings.TrimSuffix(name, "-server")
	name = strings.TrimSuffix(name, "-mcp")

	return name
}

// processMCPModule070080 processes MCP module data in 0.7.0/0.8.0 format (servers array).
func processMCPModule070080(mcpModule *structpb.Struct, servers map[string]MCPServer, inputs *[]MCPInput) error {
	serversVal, ok := mcpModule.GetFields()["servers"]
	if !ok {
		return errors.New("invalid or missing 'servers' in MCP module data")
	}

	serversList := serversVal.GetListValue()
	if serversList == nil {
		return errors.New("'servers' must be an array")
	}

	for _, serverVal := range serversList.GetValues() {
		serverMap := serverVal.GetStructValue()
		if serverMap == nil {
			continue
		}

		// Extract server name
		nameVal, ok := serverMap.GetFields()["name"]
		if !ok {
			continue
		}

		originalName := nameVal.GetStringValue()
		// Normalize server name for GH Copilot config
		serverName := normalizeServerName(originalName)

		server, err := processMCPServer(serverMap, originalName, inputs)
		if err != nil {
			return err
		}

		servers[serverName] = server
	}

	return nil
}

// RecordToGHCopilot translates a record into a GHCopilotMCPConfig structure.
// Supports OASF versions 0.7.0, 0.8.0, and 1.0.0.
func RecordToGHCopilot(record *structpb.Struct) (*GHCopilotMCPConfig, error) { //nolint:gocognit
	// Get MCP module - try 0.8.0/1.0.0 name first, then fall back to 0.7.0 for backward compatibility
	found, mcpModule := getModuleDataFromRecord(record, MCPModuleName) // "integration/mcp" (0.8.0, 1.0.0)
	if !found {
		found, mcpModule = getModuleDataFromRecord(record, "runtime/mcp") // 0.7.0 compatibility
	}

	if !found {
		return nil, errors.New("MCP module not found in record")
	}

	servers := make(map[string]MCPServer)
	inputs := []MCPInput{}

	// Check for 1.0.0 format: mcp_data with connections array
	if _, ok := mcpModule.GetFields()["name"]; ok { //nolint:nestif
		// 1.0.0 format: mcp_data object with name and connections
		err := processMCPModule100(mcpModule, servers, &inputs)
		if err != nil {
			return nil, err
		}
	} else if _, ok := mcpModule.GetFields()["servers"]; ok {
		// 0.7.0/0.8.0 format: servers array
		err := processMCPModule070080(mcpModule, servers, &inputs)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("invalid MCP module data: missing 'servers' (0.7.0/0.8.0) or 'connections' (1.0.0)")
	}

	return &GHCopilotMCPConfig{
		Servers: servers,
		Inputs:  inputs,
	}, nil
}

// processHeaders converts MCP headers array to structpb.Struct.
func processHeaders(headers []any) *structpb.Struct {
	headersMap := &structpb.Struct{Fields: map[string]*structpb.Value{}}

	for _, header := range headers {
		if headerMap, ok := header.(map[string]any); ok {
			if name, ok := headerMap["name"].(string); ok {
				headerValue := ""
				if val, ok := headerMap["value"].(string); ok {
					headerValue = val
				} else if _, ok := headerMap["description"].(string); ok {
					headerValue = "{" + strings.ToLower(strings.ReplaceAll(name, " ", "_")) + "}"
				}

				headersMap.Fields[name] = &structpb.Value{
					Kind: &structpb.Value_StringValue{StringValue: headerValue},
				}
			}
		}
	}

	if len(headersMap.GetFields()) > 0 {
		return headersMap
	}

	return nil
}

// buildStdioConnection builds a stdio connection from package data.
func buildStdioConnection(pkgMap map[string]any) map[string]*structpb.Value { //nolint:gocognit,nestif,gocyclo,cyclop,maintidx
	connectionFields := map[string]*structpb.Value{}

	registryType := ""
	identifier := ""

	if rt, ok := pkgMap["registryType"].(string); ok {
		registryType = rt
	}

	if id, ok := pkgMap["identifier"].(string); ok {
		identifier = id
	}

	// Build command
	var command string
	if runtimeHint, ok := pkgMap["runtimeHint"].(string); ok {
		command = runtimeHint
	} else {
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

	connectionFields["command"] = &structpb.Value{
		Kind: &structpb.Value_StringValue{StringValue: command},
	}

	// Build args array
	var argsValues []*structpb.Value

	_, hasRuntimeHint := pkgMap["runtimeHint"]

	if !hasRuntimeHint && registryType != "" {
		switch registryType {
		case packageTypePyPI:
			argsValues = append(argsValues, &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: "-m"},
			})
		case packageTypeOCI:
			argsValues = append(argsValues, &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: "run"},
			})
		case packageTypeNuGet:
			argsValues = append(argsValues, &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: "tool"},
			})
			argsValues = append(argsValues, &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: "run"},
			})
		case packageTypeMCPB:
			argsValues = append(argsValues, &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: "run"},
			})
		}
	}

	// Add runtime arguments
	if runtimeArgs, ok := pkgMap["runtimeArguments"].([]any); ok { //nolint:nestif
		for _, arg := range runtimeArgs {
			if argMap, ok := arg.(map[string]any); ok {
				if argType, ok := argMap["type"].(string); ok && argType == "named" {
					if name, ok := argMap["name"].(string); ok {
						argsValues = append(argsValues, &structpb.Value{
							Kind: &structpb.Value_StringValue{StringValue: name},
						})
					}
				} else if argType == "positional" {
					if value, ok := argMap["value"].(string); ok {
						argsValues = append(argsValues, &structpb.Value{
							Kind: &structpb.Value_StringValue{StringValue: value},
						})
					}
				}
			}
		}
	}

	// Add package identifier
	if identifier != "" && registryType != "" { //nolint:nestif
		pkgVersion := ""
		if v, ok := pkgMap["version"].(string); ok {
			pkgVersion = v
		}

		switch registryType {
		case packageTypeNPM:
			if pkgVersion != "" && !hasRuntimeHint {
				argsValues = append(argsValues, &structpb.Value{
					Kind: &structpb.Value_StringValue{StringValue: fmt.Sprintf("%s@%s", identifier, pkgVersion)},
				})
			} else {
				argsValues = append(argsValues, &structpb.Value{
					Kind: &structpb.Value_StringValue{StringValue: identifier},
				})
			}
		case packageTypePyPI, packageTypeMCPB:
			argsValues = append(argsValues, &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: identifier},
			})
		case packageTypeOCI:
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
		connectionFields["args"] = &structpb.Value{
			Kind: &structpb.Value_ListValue{
				ListValue: &structpb.ListValue{Values: argsValues},
			},
		}
	}

	// Add environment variables
	if envVars, ok := pkgMap["environmentVariables"].([]any); ok { //nolint:nestif
		envVarsValues := make([]*structpb.Value, 0, len(envVars))
		for _, envVar := range envVars {
			if envMap, ok := envVar.(map[string]any); ok {
				name, hasName := envMap["name"].(string)
				if !hasName || name == "" {
					continue
				}

				envFields := map[string]*structpb.Value{
					"name": {
						Kind: &structpb.Value_StringValue{StringValue: name},
					},
				}

				description := "Environment variable: " + name
				if desc, ok := envMap["description"].(string); ok && desc != "" {
					description = desc
				}

				envFields["description"] = &structpb.Value{
					Kind: &structpb.Value_StringValue{StringValue: description},
				}

				if value, ok := envMap["value"].(string); ok {
					envFields["default_value"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: value},
					}
				} else if defaultVal, ok := envMap["default"].(string); ok {
					envFields["default_value"] = &structpb.Value{
						Kind: &structpb.Value_StringValue{StringValue: defaultVal},
					}
				}

				envVarsValues = append(envVarsValues, &structpb.Value{
					Kind: &structpb.Value_StructValue{
						StructValue: &structpb.Struct{Fields: envFields},
					},
				})
			}
		}

		if len(envVarsValues) > 0 {
			connectionFields["env_vars"] = &structpb.Value{
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{Values: envVarsValues},
				},
			}
		}
	}

	return connectionFields
}

// convertPackageToConnection converts an MCP package to an mcp_server_connection.
func convertPackageToConnection(pkgMap map[string]any) *structpb.Struct {
	connectionFields := map[string]*structpb.Value{}

	// Determine connection type from transport
	connectionType := connectionTypeStdio
	transportUrl := ""

	if transport, ok := pkgMap["transport"].(map[string]any); ok {
		if tType, ok := transport["type"].(string); ok {
			switch tType {
			case connectionTypeSSE:
				connectionType = connectionTypeSSE
			case connectionTypeHTTP:
				connectionType = connectionTypeHTTP
			}
		}

		if url, ok := transport["url"].(string); ok {
			transportUrl = url
		}
	}

	connectionFields["type"] = &structpb.Value{
		Kind: &structpb.Value_StringValue{StringValue: connectionType},
	}

	// For stdio connections, build command and args
	if connectionType == connectionTypeStdio { //nolint:nestif
		stdioFields := buildStdioConnection(pkgMap)

		// Copy fields from stdio connection
		maps.Copy(connectionFields, stdioFields)
	} else {
		// For HTTP/SSE connections, add URL
		if transportUrl != "" {
			connectionFields["url"] = &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: transportUrl},
			}
		}

		// Add headers if present
		if transport, ok := pkgMap["transport"].(map[string]any); ok {
			if headers, ok := transport["headers"].([]any); ok && len(headers) > 0 {
				headersMap := processHeaders(headers)
				if headersMap != nil {
					connectionFields["headers"] = &structpb.Value{
						Kind: &structpb.Value_StructValue{StructValue: headersMap},
					}
				}
			}
		}
	}

	return &structpb.Struct{Fields: connectionFields}
}

// convertRemoteToConnection converts an MCP remote to an mcp_server_connection.
func convertRemoteToConnection(remoteMap map[string]any) *structpb.Struct {
	connectionFields := map[string]*structpb.Value{}

	// Determine connection type
	connectionType := connectionTypeHTTP

	if rType, ok := remoteMap["type"].(string); ok {
		switch rType {
		case connectionTypeSSE:
			connectionType = connectionTypeSSE
		case connectionTypeHTTP:
			connectionType = connectionTypeHTTP
		}
	}

	connectionFields["type"] = &structpb.Value{
		Kind: &structpb.Value_StringValue{StringValue: connectionType},
	}

	// Add URL (required)
	if url, ok := remoteMap["url"].(string); ok {
		connectionFields["url"] = &structpb.Value{
			Kind: &structpb.Value_StringValue{StringValue: url},
		}
	}

	// Add headers if present
	if headers, ok := remoteMap["headers"].([]any); ok && len(headers) > 0 {
		headersMap := processHeaders(headers)
		if headersMap != nil {
			connectionFields["headers"] = &structpb.Value{
				Kind: &structpb.Value_StructValue{StructValue: headersMap},
			}
		}
	}

	return &structpb.Struct{Fields: connectionFields}
}

// MCPToRecord translates an MCP Registry server.json into an OASF-compliant record format (1.0.0-rc.1).
func MCPToRecord(mcpData *structpb.Struct) (*structpb.Struct, error) { //nolint:gocognit,cyclop,maintidx
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
	serverVersion := "v1.0.0-rc.1"
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
	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Build connections array from packages and/or remotes
	var connections []*structpb.Value

	// Process packages (convert to connections)
	if packages, ok := serverMap["packages"].([]any); ok {
		for _, pkg := range packages {
			if pkgMap, ok := pkg.(map[string]any); ok {
				connection := convertPackageToConnection(pkgMap)

				connections = append(connections, &structpb.Value{
					Kind: &structpb.Value_StructValue{StructValue: connection},
				})
			}
		}
	}

	// Process remotes (convert to connections)
	if remotes, ok := serverMap["remotes"].([]any); ok {
		for _, remote := range remotes {
			if remoteMap, ok := remote.(map[string]any); ok {
				connection := convertRemoteToConnection(remoteMap)

				connections = append(connections, &structpb.Value{
					Kind: &structpb.Value_StructValue{StructValue: connection},
				})
			}
		}
	}

	if len(connections) == 0 {
		return nil, errors.New("no packages or remotes found in MCP server data")
	}

	// Create a copy of mcpServerStruct without $schema field
	mcpDataWithoutSchema := &structpb.Struct{
		Fields: make(map[string]*structpb.Value),
	}

	for k, v := range mcpServerStruct.GetFields() {
		if k != "$schema" {
			mcpDataWithoutSchema.Fields[k] = v
		}
	}

	// Create mcp_data structure with the entire server.json stored in mcp_data field (without $schema)
	mcpDataFields := map[string]*structpb.Value{
		"name": {
			Kind: &structpb.Value_StringValue{StringValue: serverName},
		},
		"connections": {
			Kind: &structpb.Value_ListValue{
				ListValue: &structpb.ListValue{Values: connections},
			},
		},
		"mcp_data": {
			Kind: &structpb.Value_StructValue{StructValue: mcpDataWithoutSchema},
		},
	}

	if serverDescription != "" {
		mcpDataFields["description"] = &structpb.Value{
			Kind: &structpb.Value_StringValue{StringValue: serverDescription},
		}
	}

	mcpModuleData := &structpb.Struct{Fields: mcpDataFields}

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
						Values: []*structpb.Value{},
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
											"urls": {
												Kind: &structpb.Value_ListValue{
													ListValue: &structpb.ListValue{
														Values: []*structpb.Value{
															{
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
						},
					},
				},
			},
			"domains": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{},
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
