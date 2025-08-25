// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	validationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/validation/v1/validationv1grpc"
	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/validation/v1"
	"github.com/agntcy/oasf-sdk/validation/service"
)

type validationCtrl struct {
	validationv1grpc.UnimplementedValidationServiceServer
	validationService *service.ValidationService
}

func NewValidationController() (validationv1grpc.ValidationServiceServer, error) {
	validationService, err := service.NewValidationService()
	if err != nil {
		return nil, fmt.Errorf("failed to create validation service: %w", err)
	}

	return &validationCtrl{
		UnimplementedValidationServiceServer: validationv1grpc.UnimplementedValidationServiceServer{},
		validationService:                    validationService,
	}, nil
}

func (v validationCtrl) ValidateRecord(_ context.Context, req *validationv1.ValidateRecordRequest) (*validationv1.ValidateRecordResponse, error) {
	slog.Info("Received ValidateRecord request", "request", req)

	isValid, errors, err := v.validationService.ValidateRecord(req)
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
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to receive record: %w", err)
		}

		validateReq := &validationv1.ValidateRecordRequest{
			Record:    req.Record,
			SchemaUrl: req.SchemaUrl,
		}

		isValid, errors, validationErr := v.validationService.ValidateRecord(validateReq)
		if validationErr != nil {
			return fmt.Errorf("failed to validate record: %w", validationErr)
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
