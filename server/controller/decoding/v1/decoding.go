// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"
	"log/slog"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/decoding/v1/decodingv1grpc"
	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/decoding/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
)

type decodingCtrl struct{}

func New() decodingv1grpc.DecodingServiceServer {
	return &decodingCtrl{}
}

func (t *decodingCtrl) DecodeRecord(_ context.Context, req *decodingv1.DecodeRecordRequest) (*decodingv1.DecodeRecordResponse, error) {
	slog.Info("Received DecodeRecord request", "request", req)

	res, err := decoder.DecodeRecord(req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to decode record: %w", err)
	}

	return res, nil
}
