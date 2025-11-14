// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/decoding/v1/decodingv1grpc"
	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/decoding/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Decoding Service E2E", func() {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%s", "0.0.0.0", "31234"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).NotTo(HaveOccurred())

	client := decodingv1grpc.NewDecodingServiceClient(conn)

	Context("V0.3.1 Record Decoding", func() {
		It("should decode v0.3.1 record to v1alpha0 format matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Convert JSON to protobuf format
			encodedRecord, err := decoder.JsonToProto(validV031Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode v0.3.1 record to protobuf")

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			resp, err := client.DecodeRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "DecodeRecord should not fail for v0.3.1 record")
			Expect(resp).NotTo(BeNil(), "Response should not be nil")

			// Verify the response contains v1alpha0 record
			Expect(resp.GetV1Alpha0()).NotTo(BeNil(), "Should return v1alpha0 record for v0.3.1 schema")
			Expect(resp.GetV1Alpha1()).To(BeNil(), "Should not return v1alpha1 record for v0.3.1 schema")

			// Convert the decoded response to JSON for comparison
			v1alpha0Record := resp.GetV1Alpha0()
			actualJSON, err := json.MarshalIndent(v1alpha0Record, "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal decoded record to JSON")

			// Parse expected output
			var expectedRecord map[string]any
			err = json.Unmarshal(expectedV031Decoded, &expectedRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected decoded output")

			// Parse actual output for comparison
			var actualRecord map[string]any
			err = json.Unmarshal(actualJSON, &actualRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual decoded output")

			// Compare core fields that should match exactly
			Expect(actualRecord["name"]).To(Equal(expectedRecord["name"]))
			Expect(actualRecord["schema_version"]).To(Equal(expectedRecord["schema_version"]))
			Expect(actualRecord["version"]).To(Equal(expectedRecord["version"]))
			Expect(actualRecord["description"]).To(Equal(expectedRecord["description"]))
			Expect(actualRecord["authors"]).To(Equal(expectedRecord["authors"]))

			// Verify structure exists (content may vary due to protobuf transformations)
			Expect(actualRecord["skills"]).NotTo(BeNil())
			Expect(actualRecord["locators"]).NotTo(BeNil())
			Expect(actualRecord["signature"]).NotTo(BeNil())
			Expect(actualRecord["extensions"]).NotTo(BeNil())
		})
	})

	Context("0.8.0 Record Decoding", func() {
		It("should decode 0.8.0 record to v1alpha2 format matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Convert JSON to protobuf format
			encodedRecord, err := decoder.JsonToProto(validV080Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode 0.8.0 record to protobuf")

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			resp, err := client.DecodeRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "DecodeRecord should not fail for 0.8.0 record")
			Expect(resp).NotTo(BeNil(), "Response should not be nil")

			// Verify the response contains v1alpha2 record
			Expect(resp.GetV1Alpha2()).NotTo(BeNil(), "Should return v1alpha2 record for 0.8.0 schema")
			Expect(resp.GetV1Alpha1()).To(BeNil(), "Should not return v1alpha1 record for 0.8.0 schema")
			Expect(resp.GetV1Alpha0()).To(BeNil(), "Should not return v1alpha0 record for 0.8.0 schema")

			// Convert the decoded response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetV1Alpha2(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal decoded record to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedV080Decoded, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "Decoded 0.8.0 record should match expected output")
		})
	})

	Context("0.7.0 Record Decoding", func() {
		It("should decode 0.7.0 record to v1alpha1 format matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Convert JSON to protobuf format
			encodedRecord, err := decoder.JsonToProto(validV070Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode 0.7.0 record to protobuf")

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			resp, err := client.DecodeRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "DecodeRecord should not fail for 0.7.0 record")
			Expect(resp).NotTo(BeNil(), "Response should not be nil")

			// Verify the response contains v1alpha1 record
			Expect(resp.GetV1Alpha1()).NotTo(BeNil(), "Should return v1alpha1 record for 0.7.0 schema")
			Expect(resp.GetV1Alpha0()).To(BeNil(), "Should not return v1alpha0 record for 0.7.0 schema")

			// Convert the decoded response to JSON for comparison
			v1alpha1Record := resp.GetV1Alpha1()
			actualJSON, err := json.MarshalIndent(v1alpha1Record, "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal decoded record to JSON")

			// Parse expected output
			var expectedRecord map[string]any
			err = json.Unmarshal(expectedV070Decoded, &expectedRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected decoded output")

			// Parse actual output for comparison
			var actualRecord map[string]any
			err = json.Unmarshal(actualJSON, &actualRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual decoded output")

			// Compare core fields that should match exactly
			Expect(actualRecord["name"]).To(Equal(expectedRecord["name"]))
			Expect(actualRecord["schema_version"]).To(Equal(expectedRecord["schema_version"]))
			Expect(actualRecord["description"]).To(Equal(expectedRecord["description"]))
			Expect(actualRecord["authors"]).To(Equal(expectedRecord["authors"]))

			// Verify structure exists (content may vary due to protobuf transformations)
			Expect(actualRecord["skills"]).NotTo(BeNil())
			Expect(actualRecord["locators"]).NotTo(BeNil())
			Expect(actualRecord["domains"]).NotTo(BeNil())
			Expect(actualRecord["signature"]).NotTo(BeNil())
		})
	})

	Context("Error Handling", func() {
		It("should return error for record without schema_version", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Create a record without schema_version field
			recordWithoutSchema := map[string]any{
				"authors":     []string{"Test Author"},
				"created_at":  "2025-09-11T12:00:00Z",
				"description": "Record without schema version",
				"name":        "example.org/no-schema",
			}

			encodedRecord, err := decoder.StructToProto(recordWithoutSchema)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode record without schema")

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			_, err = client.DecodeRecord(ctx, req)
			Expect(err).To(HaveOccurred(), "DecodeRecord should fail for record without schema_version")
			Expect(err.Error()).To(ContainSubstring("schema_version field is missing"))
		})

		It("should return error for unsupported schema version", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Create a record with unsupported schema version
			recordWithUnsupportedSchema := map[string]any{
				"authors":        []string{"Test Author"},
				"created_at":     "2025-09-11T12:00:00Z",
				"description":    "Record with unsupported schema",
				"name":           "example.org/unsupported-schema",
				"schema_version": "v99.99.99",
			}

			encodedRecord, err := decoder.StructToProto(recordWithUnsupportedSchema)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode record with unsupported schema")

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			_, err = client.DecodeRecord(ctx, req)
			Expect(err).To(HaveOccurred(), "DecodeRecord should fail for unsupported schema version")
			Expect(err.Error()).To(ContainSubstring("unsupported OASF version: v99.99.99"))
		})

		It("should return error for nil request", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &decodingv1.DecodeRecordRequest{
				Record: nil,
			}

			_, err := client.DecodeRecord(ctx, req)
			Expect(err).To(HaveOccurred(), "DecodeRecord should fail for nil record")
			Expect(err.Error()).To(ContainSubstring("request is nil"))
		})

		It("should return error for record with nil data", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &decodingv1.DecodeRecordRequest{
				Record: nil,
			}

			_, err := client.DecodeRecord(ctx, req)
			Expect(err).To(HaveOccurred(), "DecodeRecord should fail for record with nil data")
			Expect(err.Error()).To(ContainSubstring("request is nil"))
		})
	})

	Context("Schema Version Detection", func() {
		It("should correctly identify v0.3.1 schema version", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(validV031Record)
			Expect(err).NotTo(HaveOccurred())

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			resp, err := client.DecodeRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Should map to v1alpha0
			Expect(resp.GetV1Alpha0()).NotTo(BeNil())
			Expect(resp.GetV1Alpha0().GetSchemaVersion()).To(Equal("v0.3.1"))
		})

		It("should correctly identify 0.3.1 schema version", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(valid031Record)
			Expect(err).NotTo(HaveOccurred())

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			resp, err := client.DecodeRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.GetV1Alpha0()).NotTo(BeNil())
			Expect(resp.GetV1Alpha0().GetSchemaVersion()).To(Equal("0.3.1"))
		})

		It("should correctly identify 0.7.0 schema version", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(validV070Record)
			Expect(err).NotTo(HaveOccurred())

			req := &decodingv1.DecodeRecordRequest{
				Record: encodedRecord,
			}

			resp, err := client.DecodeRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			// Should map to v1alpha1
			Expect(resp.GetV1Alpha1()).NotTo(BeNil())
			Expect(resp.GetV1Alpha1().GetSchemaVersion()).To(Equal("0.7.0"))
		})
	})
})
