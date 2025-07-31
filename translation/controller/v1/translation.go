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

func (t translationCtrl) GenerateArtifactFromRecord(ctx context.Context, req *translationv1.GenerateArtifactFromRecordRequest) (*translationv1.GenerateArtifactFromRecordResponse, error) {
	slog.Info("Received Publish request", "request", req)

	data, err := t.translationService.GenerateArtifactFromRecord(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate artifact from record: %w", err)
	}

	return &translationv1.GenerateArtifactFromRecordResponse{Data: data}, nil
}
