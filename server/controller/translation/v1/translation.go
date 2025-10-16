// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"
	"log/slog"
	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/translation/v1/translationv1grpc"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/translation/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/agntcy/oasf-sdk/pkg/translator"
)

type translationCtrl struct{}

func New() translationv1grpc.TranslationServiceServer {
	return &translationCtrl{}
}

func (t *translationCtrl) RecordToGHCopilot(_ context.Context, req *translationv1.RecordToGHCopilotRequest) (*translationv1.RecordToGHCopilotResponse, error) {
	slog.Info("Received Publish request", "request", req)

	result, err := translator.RecordToGHCopilot(req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to generate GHCopilot config from record: %w", err)
	}

	data, err := decoder.StructToProto(map[string]any{"mcpConfig": result})
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to proto struct: %w", err)
	}

	return &translationv1.RecordToGHCopilotResponse{Data: data}, nil
}

func (t *translationCtrl) RecordToA2A(_ context.Context, req *translationv1.RecordToA2ARequest) (*translationv1.RecordToA2AResponse, error) {
	slog.Info("Received RecordToA2A request", "request", req)

	result, err := translator.RecordToA2A(req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to generate A2A card from record: %w", err)
	}

	data, err := decoder.StructToProto(map[string]any{"a2aCard": result})
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
func (t *translationCtrl) A2AToRecord(_ context.Context, req *translationv1.A2AToRecordRequest) (*translationv1.A2AToRecordResponse, error) {
	slog.Info("Received A2AToRecord request", "request", req)

	result, err := translator.A2AToRecord(req.GetData())
	if err != nil {
		return nil, fmt.Errorf("failed to generate record from A2A data: %w", err)
	}

	return &translationv1.A2AToRecordResponse{Record: result}, nil
}
