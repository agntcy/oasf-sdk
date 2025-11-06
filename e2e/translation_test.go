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

	Context("GH Copilot config Generation", func() {
		It("should generate github GH Copilot config from 0.8.0 record matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV080Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToGHCopilotRequest{
				Record: encodedRecord,
			}

			resp, err := client.RecordToGHCopilot(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToGHCopilot should not fail")
			Expect(resp.Data).NotTo(BeNil(), "Expected GH Copilot config data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.Data.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal GH Copilot config to JSON")

			// Parse expected output
			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedGHCopilotOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected GH Copilot output")

			// Parse actual output for comparison
			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual GH Copilot output")

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
			Expect(resp.Data).NotTo(BeNil(), "Expected GH Copilot config data in response")

			// Verify we got valid MCP config structure
			mcpConfig := resp.Data.AsMap()["mcpConfig"]
			Expect(mcpConfig).NotTo(BeNil(), "Expected mcpConfig in response")
		})
	})

	Context("A2A Card Extraction", func() {
		It("should extract A2A card from 0.8.0 record matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV080Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToA2ARequest{
				Record: encodedRecord,
			}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail")
			Expect(resp.Data).NotTo(BeNil(), "Expected A2A card data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.Data.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal A2A card to JSON")

			// Parse expected output
			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedA2AOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected A2A output")

			// Parse actual output for comparison
			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual A2A output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "A2A card should match expected output")
		})

		It("should extract A2A card from 0.7.0 record (backward compatibility)", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV070Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToA2ARequest{
				Record: encodedRecord,
			}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail for 0.7.0 record")
			Expect(resp.Data).NotTo(BeNil(), "Expected A2A card data in response")

			// Verify we got valid A2A card structure
			a2aCard := resp.Data.AsMap()["a2aCard"]
			Expect(a2aCard).NotTo(BeNil(), "Expected a2aCard in response")
		})
	})

	Context("A2A to Record Translation", func() {
		It("should convert A2A card to OASF record matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedA2AData, err := decoder.JsonToProto(expectedA2AOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode A2A data")

			req := &translationv1.A2AToRecordRequest{
				Data: encodedA2AData,
			}

			resp, err := client.A2AToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "A2AToRecord should not fail")
			Expect(resp.Record).NotTo(BeNil(), "Expected OASF record in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.Record.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal record to JSON")

			// Parse expected output
			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedA2AToRecordOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected A2AToRecord output")

			// Parse actual output for comparison
			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual A2AToRecord output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "OASF record should match expected output")
		})
	})

	Context("MCP to Record Translation", func() {
		It("should convert MCP Registry entry to OASF record matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPRegistry)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode MCP Registry data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail")
			Expect(resp.Record).NotTo(BeNil(), "Expected OASF record in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.Record.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal record to JSON")

			// Parse expected output
			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedMCPToRecordOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected MCPToRecord output")

			// Parse actual output for comparison
			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual MCPToRecord output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "OASF record should match expected output")
		})

		It("should convert minimal local MCP server (pypi, no runtimeHint) to OASF record", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPMinimalLocal)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode minimal local MCP data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail for minimal local")
			Expect(resp.Record).NotTo(BeNil(), "Expected OASF record in response")

			actualJSON, err := json.MarshalIndent(resp.Record.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal record to JSON")

			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedMCPMinimalLocalOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected minimal local output")

			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual minimal local output")

			Expect(actualOutput).To(Equal(expectedOutput), "Minimal local OASF record should match expected output")
		})

		It("should convert HTTP remote MCP server with headers to OASF record", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPHTTPHeaders)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode HTTP headers MCP data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail for HTTP with headers")
			Expect(resp.Record).NotTo(BeNil(), "Expected OASF record in response")

			actualJSON, err := json.MarshalIndent(resp.Record.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal record to JSON")

			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedMCPHTTPHeadersOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected HTTP headers output")

			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual HTTP headers output")

			Expect(actualOutput).To(Equal(expectedOutput), "HTTP headers OASF record should match expected output")
		})

		It("should convert minimal SSE remote MCP server to OASF record", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedMCPData, err := decoder.JsonToProto(translationMCPSSEMinimal)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode SSE minimal MCP data")

			req := &translationv1.MCPToRecordRequest{
				Data: encodedMCPData,
			}

			resp, err := client.MCPToRecord(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "MCPToRecord should not fail for SSE minimal")
			Expect(resp.Record).NotTo(BeNil(), "Expected OASF record in response")

			actualJSON, err := json.MarshalIndent(resp.Record.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal record to JSON")

			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedMCPSSEMinimalOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected SSE minimal output")

			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual SSE minimal output")

			Expect(actualOutput).To(Equal(expectedOutput), "SSE minimal OASF record should match expected output")
		})
	})
})
