// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"
	"log/slog"

	translationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/translation/v1/translationv1grpc"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/translation/v1"
	"github.com/agntcy/oasf-sdk/translation/service"
)

type translationCtrl struct {
	translationv1grpc.UnimplementedTranslationServiceServer
	translationService *service.TranslationService
}

func NewRoutingController() translationv1grpc.TranslationServiceServer {
	return &translationCtrl{
		UnimplementedTranslationServiceServer: translationv1grpc.UnimplementedTranslationServiceServer{},
		translationService:                    service.NewTranslationService(),
	}
}

func (t translationCtrl) RecordToVSCodeCopilot(_ context.Context, req *translationv1.RecordToVSCodeCopilotRequest) (*translationv1.RecordToVSCodeCopilotResponse, error) {
	slog.Info("Received Publish request", "request", req)

	data, err := t.translationService.RecordToVSCodeCopilot(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate VSCodeCopilot config from record: %w", err)
	}

	return &translationv1.RecordToVSCodeCopilotResponse{Data: data}, nil
}

func (t translationCtrl) RecordToA2A(_ context.Context, req *translationv1.RecordToA2ARequest) (*translationv1.RecordToA2AResponse, error) {
	slog.Info("Received RecordToA2A request", "request", req)

	data, err := t.translationService.RecordToA2A(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate A2A card from record: %w", err)
	}

	return &translationv1.RecordToA2AResponse{Data: data}, nil
}
