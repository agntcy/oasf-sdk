// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"
	"log/slog"

	translationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/translation/v1/translationv1grpc"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/translation/v1"
	"github.com/agntcy/oasf-sdk/core/converter"
	"github.com/agntcy/oasf-sdk/translation/translator"
)

type translationCtrl struct {
	translator *translator.Translator
}

func NewRoutingController() translationv1grpc.TranslationServiceServer {
	return &translationCtrl{
		translator: translator.New(),
	}
}

func (t translationCtrl) RecordToGHCopilot(_ context.Context, req *translationv1.RecordToGHCopilotRequest) (*translationv1.RecordToGHCopilotResponse, error) {
	slog.Info("Received Publish request", "request", req)

	result, err := t.translator.RecordToGHCopilot(req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to generate GHCopilot config from record: %w", err)
	}

	data, err := converter.StructToProto(map[string]any{"mcpConfig": result})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to proto struct: %w", err)
	}

	return &translationv1.RecordToGHCopilotResponse{Data: data}, nil
}

func (t translationCtrl) RecordToA2A(_ context.Context, req *translationv1.RecordToA2ARequest) (*translationv1.RecordToA2AResponse, error) {
	slog.Info("Received RecordToA2A request", "request", req)

	result, err := t.translator.RecordToA2A(req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to generate A2A card from record: %w", err)
	}

	data, err := converter.StructToProto(map[string]any{"a2aCard": result})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to proto struct: %w", err)
	}

	return &translationv1.RecordToA2AResponse{Data: data}, nil
}

// GHCopilotToRecord implements translationv1grpc.TranslationServiceServer.
func (t *translationCtrl) GHCopilotToRecord(context.Context, *translationv1.GHCopilotToRecordRequest) (*translationv1.GHCopilotToRecordResponse, error) {
	panic("unimplemented")
}

// A2AToRecord implements translationv1grpc.TranslationServiceServer.
func (t *translationCtrl) A2AToRecord(context.Context, *translationv1.A2AToRecordRequest) (*translationv1.A2AToRecordResponse, error) {
	panic("unimplemented")
}
