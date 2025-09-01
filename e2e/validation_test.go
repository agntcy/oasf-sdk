// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"time"

	validationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/validation/v1/validationv1grpc"
	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/validation/v1"
	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
)

var _ = Describe("Validation Service E2E", func() {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%s", "0.0.0.0", "31235"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).NotTo(HaveOccurred())

	client := validationv1grpc.NewValidationServiceClient(conn)

	testCases := []struct {
		name       string
		jsonData   []byte
		schemaURL  string
		shouldPass bool
	}{
		{
			name:       "valid_record_v0.5.0.json",
			jsonData:   validV050Record,
			shouldPass: true,
		},
		{
			name:       "valid_record_v0.6.0.json",
			jsonData:   validV060Record,
			shouldPass: true,
		},
		{
			name:       "valid_record_v0.6.0.json with explicit schema URL",
			jsonData:   validV060Record,
			schemaURL:  "https://schema.oasf.outshift.com/schema/0.6.0/objects/record",
			shouldPass: true,
		},
		{
			name:     "invalid_record_v0.6.0.json",
			jsonData: invalidV060Record,
		},
	}

	for _, tc := range testCases {
		Context(tc.name, func() {
			It("should return valid with no errors", func() {
				var record objectsv3.Record
				err := protojson.Unmarshal(tc.jsonData, &record)
				Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal record from %s", tc.name)

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				req := &validationv1.ValidateRecordRequest{
					Record:    &record,
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
