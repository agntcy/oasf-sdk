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

type VSCodeMCPConfig struct {
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

func NewTranslationService() *TranslationService {
	return &TranslationService{}
}

func (t TranslationService) GenerateArtifactFromRecord(req *translationv1.GenerateArtifactFromRecordRequest) (*structpb.Struct, error) {
	switch req.TranslationType {
	case translationv1.TranslationType_TRANSLATION_TYPE_UNSPECIFIED:
		return nil, errors.New("translation type is unspecified")
	case translationv1.TranslationType_TRANSLATION_TYPE_VSCODE:
		return generateVSCodeArtifactFromRecord(req.Record)
	default:
		return nil, errors.New("unsupported translation type")
	}
}

func generateVSCodeArtifactFromRecord(record *objectsv3.Record) (*structpb.Struct, error) {
	vscodeMCPConfig, err := buildVSCodeMCPConfig(record)
	if err != nil {
		return nil, fmt.Errorf("failed to build VSCode MCP config: %w", err)
	}

	return toStruct(map[string]any{
		"mcpConfig": *vscodeMCPConfig,
	})
}

func buildVSCodeMCPConfig(record *objectsv3.Record) (*VSCodeMCPConfig, error) {
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

		command := serverMap.Fields["command"].GetStringValue()

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
			Command: command,
			Args:    args,
			Env:     env,
		}
	}

	return &VSCodeMCPConfig{
		Servers: servers,
		Inputs:  inputs,
	}, nil
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
