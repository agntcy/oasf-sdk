// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	corev1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/core/v1"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	MCPModuleName = "runtime/mcp"
	A2AModuleName = "runtime/a2a"
)

type Translator struct{}

func New() *Translator {
	return &Translator{}
}

// RecordToGHCopilot translates a record into a GHCopilotMCPConfig structure.
func (t *Translator) RecordToGHCopilot(req *corev1.EncodedRecord) (*GHCopilotMCPConfig, error) {
	// Get MCP module
	found, mcpModule := getModuleFromRecord(req, MCPModuleName)
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
func (t *Translator) RecordToA2A(req *corev1.EncodedRecord) (*A2ACard, error) {
	// Get A2A module
	found, a2aModule := getModuleFromRecord(req, A2AModuleName)
	if !found {
		return nil, errors.New("A2A module not found in record")
	}

	// Process A2A module
	jsonBytes, err := json.Marshal(a2aModule.AsMap())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal A2A data to JSON: %w", err)
	}

	var card A2ACard
	if err := json.Unmarshal(jsonBytes, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal A2A data into A2ACard: %w", err)
	}

	return &card, nil
}

func getModuleFromRecord(record *corev1.EncodedRecord, moduleName string) (bool, *structpb.Struct) {
	// Find module by name
	for _, module := range record.GetRecord().GetFields()["modules"].GetListValue().Values {
		if strings.HasSuffix(module.GetStructValue().GetFields()["name"].GetStringValue(), moduleName) {
			return true, module.GetStructValue()
		}
	}
	return false, nil
}
