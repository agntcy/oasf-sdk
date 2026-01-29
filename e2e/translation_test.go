// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/translation/v1/translationv1grpc"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/translation/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Translation Service E2E", func() {
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%s", "0.0.0.0", "31234"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	Expect(err).NotTo(HaveOccurred())

	client := translationv1grpc.NewTranslationServiceClient(conn)

	Context("GH Copilot config Generation", func() { //nolint:dupl
		It("should generate github GH Copilot config from 0.8.0 record matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV080Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToGHCopilotRequest{Record: encodedRecord}

			resp, err := client.RecordToGHCopilot(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToGHCopilot should not fail")
			Expect(resp.GetData()).NotTo(BeNil(), "Expected GH Copilot config data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetData().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedGHCopilotOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "GH Copilot config should match expected output")
		})

		It("should generate github GH Copilot config from 0.7.0 record (backward compatibility)", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV070Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToGHCopilotRequest{
				Record: encodedRecord,
			}

			resp, err := client.RecordToGHCopilot(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToGHCopilot should not fail for 0.7.0 record")
			Expect(resp.GetData()).NotTo(BeNil(), "Expected GH Copilot config data in response")

			// Verify we got valid MCP config structure
			mcpConfig := resp.GetData().AsMap()["mcpConfig"]
			Expect(mcpConfig).NotTo(BeNil(), "Expected mcpConfig in response")
		})
	})

	Context("A2A Card Extraction", func() { //nolint:dupl
		It("should extract A2A card from 0.8.0 record matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV080Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToA2ARequest{Record: encodedRecord}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail")
			Expect(resp.GetData()).NotTo(BeNil(), "Expected A2A card data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetData().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output (0.7.0 and 0.8.0 records have the same card data structure)
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedA2AFrom070080Output, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "A2A card should match expected output")
		})

		It("should extract A2A card from 0.7.0 record matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV070Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToA2ARequest{Record: encodedRecord}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail for 0.7.0 record")
			Expect(resp.GetData()).NotTo(BeNil(), "Expected A2A card data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetData().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output (0.7.0 and 0.8.0 records have the same card data structure)
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedA2AFrom070080Output, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "A2A card should match expected output")
		})
	})

	Context("A2A to Record Translation", func() {
		It("should convert A2A card to OASF record matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Convert A2A card to OASF record
			encodedA2ACard, err := decoder.JsonToProto(a2aCard)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode A2A card")

			req := &translationv1.A2AToRecordRequest{
				Data: encodedA2ACard,
			}

			resp, err := client.A2AToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "A2AToRecord should not fail")
			Expect(resp.GetRecord()).NotTo(BeNil(), "Expected OASF record in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetRecord().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedA2AToRecordOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// created_at is dynamically generated, so verify it exists and is valid RFC3339, then exclude from comparison
			actualCreatedAt, ok := actualOutput["created_at"].(string)
			Expect(ok).To(BeTrue(), "created_at should be present")
			Expect(actualCreatedAt).NotTo(BeEmpty(), "created_at should not be empty")
			_, err = time.Parse(time.RFC3339, actualCreatedAt)
			Expect(err).NotTo(HaveOccurred(), "created_at should be valid RFC3339 timestamp")

			// Remove dynamic fields from comparison
			delete(actualOutput, "created_at")
			delete(expectedOutput, "created_at")
			delete(expectedOutput, "_comment_created_at") // Remove comment field

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "OASF record should match expected output")
		})

		It("should convert 1.0.0 OASF record back to A2A card matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Convert 1.0.0 OASF record to proto
			encodedRecord, err := decoder.JsonToProto(translationV100Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode OASF record")

			req := &translationv1.RecordToA2ARequest{
				Record: encodedRecord,
			}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail")
			Expect(resp.GetData()).NotTo(BeNil(), "Expected A2A card data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetData().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedA2AOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "A2A card should match expected output")
		})

		It("should convert A2A-to-record output back to A2A card matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Convert expected A2A-to-record output to proto
			encodedRecord, err := decoder.JsonToProto(expectedA2AToRecordOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode OASF record")

			req := &translationv1.RecordToA2ARequest{
				Record: encodedRecord,
			}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail")
			Expect(resp.GetData()).NotTo(BeNil(), "Expected A2A card data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetData().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedA2AOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "A2A card should match expected output")
		})
	})

	Context("MCP to Record Translation", func() {
		It("should convert MCP Registry entry to OASF record matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPRegistry)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode MCP Registry data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail")
			Expect(resp.GetRecord()).NotTo(BeNil(), "Expected OASF record in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetRecord().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedMCPToRecordOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// created_at is dynamically generated, so verify it exists and is valid RFC3339, then exclude from comparison
			actualCreatedAt, ok := actualOutput["created_at"].(string)
			Expect(ok).To(BeTrue(), "created_at should be present")
			Expect(actualCreatedAt).NotTo(BeEmpty(), "created_at should not be empty")
			_, err = time.Parse(time.RFC3339, actualCreatedAt)
			Expect(err).NotTo(HaveOccurred(), "created_at should be valid RFC3339 timestamp")

			// Remove dynamic fields from comparison
			delete(actualOutput, "created_at")
			delete(expectedOutput, "created_at")
			delete(expectedOutput, "_comment_created_at") // Remove comment field

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "OASF record should match expected output")
		})

		It("should convert minimal local MCP server (pypi, no runtimeHint) to OASF record", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPMinimalLocal)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode minimal local MCP data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail for minimal local")
			Expect(resp.GetRecord()).NotTo(BeNil(), "Expected OASF record in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetRecord().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedMCPMinimalLocalOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// created_at is dynamically generated, so verify it exists and is valid RFC3339, then exclude from comparison
			actualCreatedAt, ok := actualOutput["created_at"].(string)
			Expect(ok).To(BeTrue(), "created_at should be present")
			Expect(actualCreatedAt).NotTo(BeEmpty(), "created_at should not be empty")
			_, err = time.Parse(time.RFC3339, actualCreatedAt)
			Expect(err).NotTo(HaveOccurred(), "created_at should be valid RFC3339 timestamp")

			// Remove dynamic fields from comparison
			delete(actualOutput, "created_at")
			delete(expectedOutput, "created_at")
			delete(expectedOutput, "_comment_created_at") // Remove comment field

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "Minimal local OASF record should match expected output")
		})

		It("should convert HTTP remote MCP server with headers to OASF record", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPHTTPHeaders)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode MCP HTTP headers data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail for HTTP headers")
			Expect(resp.GetRecord()).NotTo(BeNil(), "Expected OASF record in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetRecord().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedMCPHTTPHeadersOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// created_at is dynamically generated, so verify it exists and is valid RFC3339, then exclude from comparison
			actualCreatedAt, ok := actualOutput["created_at"].(string)
			Expect(ok).To(BeTrue(), "created_at should be present")
			Expect(actualCreatedAt).NotTo(BeEmpty(), "created_at should not be empty")
			_, err = time.Parse(time.RFC3339, actualCreatedAt)
			Expect(err).NotTo(HaveOccurred(), "created_at should be valid RFC3339 timestamp")

			// Remove dynamic fields from comparison
			delete(actualOutput, "created_at")
			delete(expectedOutput, "created_at")
			delete(expectedOutput, "_comment_created_at") // Remove comment field

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "HTTP headers OASF record should match expected output")
		})

		It("should convert SSE minimal MCP server to OASF record", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPSSEMinimal)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode MCP SSE minimal data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail for SSE minimal")
			Expect(resp.GetRecord()).NotTo(BeNil(), "Expected OASF record in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.GetRecord().AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal response to JSON")

			// Parse expected output
			var expectedOutput map[string]any
			err = json.Unmarshal(expectedMCPSSEMinimalOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected output")

			// Parse actual output for comparison
			var actualOutput map[string]any
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual output")

			// created_at is dynamically generated, so verify it exists and is valid RFC3339, then exclude from comparison
			actualCreatedAt, ok := actualOutput["created_at"].(string)
			Expect(ok).To(BeTrue(), "created_at should be present")
			Expect(actualCreatedAt).NotTo(BeEmpty(), "created_at should not be empty")
			_, err = time.Parse(time.RFC3339, actualCreatedAt)
			Expect(err).NotTo(HaveOccurred(), "created_at should be valid RFC3339 timestamp")

			// Remove dynamic fields from comparison
			delete(actualOutput, "created_at")
			delete(expectedOutput, "created_at")
			delete(expectedOutput, "_comment_created_at") // Remove comment field

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "SSE minimal OASF record should match expected output")
		})
	})
})
