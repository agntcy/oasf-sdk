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

type validationCtrl struct{}

func New() (validationv1grpc.ValidationServiceServer, error) {
	return &validationCtrl{}, nil
}

func (v validationCtrl) ValidateRecord(ctx context.Context, req *validationv1.ValidateRecordRequest) (*validationv1.ValidateRecordResponse, error) {
	slog.Info("Received ValidateRecord request", "request", req)

	validatorInstance, err := validator.New(req.GetSchemaUrl())
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	isValid, errors, warnings, err := validatorInstance.ValidateRecord(ctx, req.GetRecord())
	if err != nil {
		return nil, fmt.Errorf("failed to validate record: %w", err)
	}

	return &validationv1.ValidateRecordResponse{
		IsValid:  isValid,
		Errors:   errors,
		Warnings: warnings,
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

		validatorInstance, err := validator.New(req.GetSchemaUrl())
		if err != nil {
			return fmt.Errorf("failed to create validator: %w", err)
		}

		isValid, errors, warnings, err := validatorInstance.ValidateRecord(stream.Context(), req.GetRecord())
		if err != nil {
			return fmt.Errorf("failed to validate record: %w", err)
		}

		response := &validationv1.ValidateRecordStreamResponse{
			IsValid:  isValid,
			Errors:   errors,
			Warnings: warnings,
		}

		if err := stream.Send(response); err != nil {
			return fmt.Errorf("failed to send response: %w", err)
		}
	}
}
