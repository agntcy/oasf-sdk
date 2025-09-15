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
			name:       "valid_record_v0.7.0.json",
			jsonData:   validV070Record,
			shouldPass: true,
		},
		{
			name:       "valid_record_v0.7.0.json with explicit schema URL",
			jsonData:   validV070Record,
			schemaURL:  "https://schema.oasf.outshift.com/schema/0.7.0/objects/record",
			shouldPass: true,
		},
		{
			name:     "invalid_record_v0.7.0.json",
			jsonData: invalidV070Record,
		},
		{
			name:       "valid_record_v0.3.1.json",
			jsonData:   validV031Record,
			shouldPass: true,
		},
		{
			name:     "invalid_record_v0.3.1.json",
			jsonData: invalidV031Record,
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
					Expect(resp.IsValid).To(BeTrue(), "Expected valid record", resp.Errors)
					Expect(resp.Errors).To(BeEmpty(), "Expected no validation errors", resp.Errors)
				} else {
					Expect(resp.IsValid).To(BeFalse(), "Expected invalid record", resp.Errors)
					Expect(resp.Errors).NotTo(BeEmpty(), "Expected validation errors", resp.Errors)
				}
			})
		})
	}
})
