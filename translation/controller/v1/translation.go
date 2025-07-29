// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"errors"
	"log/slog"

	translationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/translation/v1/translationv1grpc"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/translation/v1"
)

type translationCtrl struct {
	translationv1grpc.UnimplementedTranslationServiceServer
}

func NewRoutingController() translationv1grpc.TranslationServiceServer {
	return &translationCtrl{
		UnimplementedTranslationServiceServer: translationv1grpc.UnimplementedTranslationServiceServer{},
	}
}

func (t translationCtrl) GenerateArtifactFromRecord(ctx context.Context, req *translationv1.GenerateArtifactFromRecordRequest) (*translationv1.GenerateArtifactFromRecordResponse, error) {
	slog.Info("Received Publish request", "request", req)

	return nil, errors.New("not implemented")
}
