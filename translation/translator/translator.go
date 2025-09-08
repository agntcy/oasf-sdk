// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	corev1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/core/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha1"
)

const (
	MCPModuleName = "runtime/mcp"
	A2AModuleName = "runtime/a2a"
)

type Translator struct{}

func New() *Translator {
	return &Translator{}
}

// Validate checks if the provided record adheres to expected structure and content.
func (_ *Translator) Validate(req *corev1.DecodedRecord) error {
	// We can only support v1alpha1 records for now.
	if req.GetV1Alpha1() == nil {
		return errors.New("record is not of version v1alpha1")
	}

	return nil
}

// RecordToGHCopilot translates a DecodedRecord into a GHCopilotMCPConfig structure.
func (t *Translator) RecordToGHCopilot(req *corev1.DecodedRecord) (*GHCopilotMCPConfig, error) {
	// Validate
	if err := t.Validate(req); err != nil {
		return nil, fmt.Errorf("record validation failed: %w", err)
	}

	// Extract versioned record
	record := req.GetV1Alpha1()

	// Find MCP module
	var mcpModule *typesv1alpha1.Module
	for _, module := range record.Modules {
		if strings.HasSuffix(module.Name, MCPModuleName) {
			mcpModule = module
			break
		}
	}

	if mcpModule == nil {
		return nil, errors.New("MCP module not found in record")
	}

	// Process MCP module
	serversVal, ok := mcpModule.Data.Fields["servers"]
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

// RecordToA2A translates a DecodedRecord into an A2ACard structure.
func (t *Translator) RecordToA2A(req *corev1.DecodedRecord) (*A2ACard, error) {
	// Validate
	if err := t.Validate(req); err != nil {
		return nil, fmt.Errorf("record validation failed: %w", err)
	}

	// Extract versioned record
	record := req.GetV1Alpha1()

	// Find A2A module
	var a2aModule *typesv1alpha1.Module
	for _, module := range record.Modules {
		if strings.HasSuffix(module.Name, A2AModuleName) {
			a2aModule = module
			break
		}
	}

	if a2aModule == nil {
		return nil, errors.New("A2A module not found in record")
	}

	// Process A2A module
	jsonBytes, err := json.Marshal(a2aModule.Data.AsMap())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal A2A data to JSON: %w", err)
	}

	var card A2ACard
	if err := json.Unmarshal(jsonBytes, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal A2A data into A2ACard: %w", err)
	}

	return &card, nil
}
