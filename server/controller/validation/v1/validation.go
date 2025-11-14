// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/validation/v1/validationv1grpc"
	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/validation/v1"
	"github.com/agntcy/oasf-sdk/pkg/validator"
)

type validationCtrl struct {
	validator *validator.Validator
}

func New() (validationv1grpc.ValidationServiceServer, error) {
	v, err := validator.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create validation service: %w", err)
	}

	return &validationCtrl{
		validator: v,
	}, nil
}

func (v validationCtrl) ValidateRecord(ctx context.Context, req *validationv1.ValidateRecordRequest) (*validationv1.ValidateRecordResponse, error) {
	slog.Info("Received ValidateRecord request", "request", req)

	isValid, errors, err := v.validator.ValidateRecord(ctx, req.GetRecord(), validator.WithSchemaURL(req.GetSchemaUrl()))
	if err != nil {
		return nil, fmt.Errorf("failed to validate record: %w", err)
	}

	return &validationv1.ValidateRecordResponse{
		IsValid: isValid,
		Errors:  errors,
	}, nil
}

func (v validationCtrl) ValidateRecordStream(stream validationv1grpc.ValidationService_ValidateRecordStreamServer) error {
	slog.Info("Received ValidateRecordStream request")

	for {
		req, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("failed to receive record: %w", err)
		}

		isValid, errors, err := v.validator.ValidateRecord(stream.Context(), req.GetRecord(), validator.WithSchemaURL(req.GetSchemaUrl()))
		if err != nil {
			return fmt.Errorf("failed to validate record: %w", err)
		}

		response := &validationv1.ValidateRecordStreamResponse{
			IsValid: isValid,
			Errors:  errors,
		}

		if err := stream.Send(response); err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
	}
}
