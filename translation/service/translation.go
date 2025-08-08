// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/translation/v1"
	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
	structpb "google.golang.org/protobuf/types/known/structpb"
)

type TranslationService struct{}

type VSCodeCopilotMCPConfig struct {
	Servers map[string]Server `json:"servers"`
	Inputs  []Input           `json:"inputs"`
}

type Server struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type Input struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Password    bool   `json:"password"`
	Description string `json:"description"`
}

type Skill struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type A2ACard struct {
	Name               string          `json:"name"`
	Description        string          `json:"description"`
	URL                string          `json:"url"`
	Capabilities       map[string]bool `json:"capabilities"`
	DefaultInputModes  []string        `json:"defaultInputModes"`
	DefaultOutputModes []string        `json:"defaultOutputModes"`
	Skills             []Skill         `json:"skills"`
}

func NewTranslationService() *TranslationService {
	return &TranslationService{}
}

func (t TranslationService) RecordToVSCodeCopilot(req *translationv1.RecordToVSCodeCopilotRequest) (*structpb.Struct, error) {
	vsCodeCopilotMCPConfig, err := buildVSCodeCopilotMCPConfig(req.Record)
	if err != nil {
		return nil, fmt.Errorf("failed to build VSCode MCP config: %w", err)
	}

	return toStruct(map[string]any{
		"mcpConfig": *vsCodeCopilotMCPConfig,
	})
}

func (t TranslationService) RecordToA2A(req *translationv1.RecordToA2ARequest) (*structpb.Struct, error) {
	a2aCard, err := buildA2ACard(req.Record)
	if err != nil {
		return nil, fmt.Errorf("failed to build A2A card: %w", err)
	}

	return toStruct(map[string]any{
		"a2aCard": *a2aCard,
	})
}

func buildVSCodeCopilotMCPConfig(record *objectsv3.Record) (*VSCodeCopilotMCPConfig, error) {
	var mcpExt *objectsv3.Extension
	for _, ext := range record.Extensions {
		if ext.Name == "schema.oasf.agntcy.org/features/runtime/mcp" {
			mcpExt = ext
			break
		}
	}

	if mcpExt == nil {
		return nil, errors.New("MCP extension not found in record")
	}

	serversVal, ok := mcpExt.Data.Fields["servers"]
	if !ok {
		return nil, errors.New("invalid or missing 'servers' in MCP extension data")
	}

	serversStruct := serversVal.GetStructValue()
	if serversStruct == nil {
		return nil, errors.New("'servers' is not a struct")
	}

	servers := make(map[string]Server)
	inputs := []Input{}

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
						inputs = append(inputs, Input{
							ID:          id,
							Type:        "promptString",
							Password:    true,
							Description: fmt.Sprintf("Secret value for %s", id),
						})
					}
				}
			}
		}

		servers[serverName] = Server{
			Command: command.GetStringValue(),
			Args:    args,
			Env:     env,
		}
	}

	return &VSCodeCopilotMCPConfig{
		Servers: servers,
		Inputs:  inputs,
	}, nil
}

func buildA2ACard(record *objectsv3.Record) (*A2ACard, error) {
	var a2aExt *objectsv3.Extension
	for _, ext := range record.Extensions {
		if ext.Name == "schema.oasf.agntcy.org/features/runtime/a2a" {
			a2aExt = ext
			break
		}
	}

	if a2aExt == nil {
		return nil, errors.New("A2A extension not found in record")
	}

	jsonBytes, err := json.Marshal(a2aExt.Data.AsMap())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal A2A data to JSON: %w", err)
	}

	var card A2ACard
	if err := json.Unmarshal(jsonBytes, &card); err != nil {
		return nil, fmt.Errorf("failed to unmarshal A2A data into A2ACard: %w", err)
	}

	return &card, nil
}

func toStruct(v any) (*structpb.Struct, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, err
	}
	return structpb.NewStruct(data)
}
