// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"time"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/validation/v1/validationv1grpc"
	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/validation/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Validation Service E2E", func() {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%s", "0.0.0.0", "31234"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).NotTo(HaveOccurred())

	client := validationv1grpc.NewValidationServiceClient(conn)

	testCases := []struct {
		name       string
		jsonData   []byte
		schemaURL  string
		shouldPass bool
	}{
		{
			name:       "valid_record_0.8.0.json",
			jsonData:   validV080Record,
			schemaURL:  "https://schema.oasf.outshift.com",
			shouldPass: true,
		},
		{
			name:       "valid_record_0.8.0.json with explicit schema URL",
			jsonData:   validV080Record,
			schemaURL:  "https://schema.oasf.outshift.com",
			shouldPass: true,
		},
		{
			name:      "invalid_record_0.8.0.json",
			jsonData:  invalidV080Record,
			schemaURL: "https://schema.oasf.outshift.com",
		},
		{
			name:       "valid_record_0.7.0.json",
			jsonData:   validV070Record,
			schemaURL:  "https://schema.oasf.outshift.com",
			shouldPass: true,
		},
		{
			name:       "valid_record_0.7.0.json with explicit schema URL",
			jsonData:   validV070Record,
			schemaURL:  "https://schema.oasf.outshift.com",
			shouldPass: true,
		},
		{
			name:      "invalid_record_0.7.0.json",
			jsonData:  invalidV070Record,
			schemaURL: "https://schema.oasf.outshift.com",
		},
		{
			name:       "valid_record_1.0.0.json",
			jsonData:   validV100Record,
			schemaURL:  "https://schema.oasf.outshift.com",
			shouldPass: true,
		},
		{
			name:       "valid_record_1.0.0.json with explicit schema URL",
			jsonData:   validV100Record,
			schemaURL:  "https://schema.oasf.outshift.com",
			shouldPass: true,
		},
		{
			name:      "invalid_record_1.0.0.json",
			jsonData:  invalidV100Record,
			schemaURL: "https://schema.oasf.outshift.com",
		},
	}

	for _, tc := range testCases {
		Context(tc.name, func() {
			It("should return with no errors", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				encodedRecord, err := decoder.JsonToProto(tc.jsonData)
				Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

				req := &validationv1.ValidateRecordRequest{
					Record:    encodedRecord,
					SchemaUrl: tc.schemaURL,
				}

				resp, err := client.ValidateRecord(ctx, req)

				Expect(err).NotTo(HaveOccurred(), "ValidateRecord should not fail", err)
				if tc.shouldPass {
					Expect(resp.GetIsValid()).To(BeTrue(), "Expected valid record", resp.GetErrors())
					Expect(resp.GetErrors()).To(BeEmpty(), "Expected no validation errors", resp.GetErrors())
				} else {
					Expect(resp.GetIsValid()).To(BeFalse(), "Expected invalid record", resp.GetErrors())
					Expect(resp.GetErrors()).NotTo(BeEmpty(), "Expected validation errors", resp.GetErrors())
				}
			})
		})
	}
})
