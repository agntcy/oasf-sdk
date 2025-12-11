// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/translation/v1/translationv1grpc"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/translation/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/yaml"
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

		It("should extract A2A card from 0.7.0 record (backward compatibility)", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationV070Record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToA2ARequest{Record: encodedRecord}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail for 0.7.0 record")
			Expect(resp.GetData()).NotTo(BeNil(), "Expected A2A card data in response")

			// Verify we got valid A2A structure
			a2aCard := resp.GetData().AsMap()["a2aCard"]
			Expect(a2aCard).NotTo(BeNil(), "Expected a2aCard in response")
		})
	})

	Context("A2A to Record Translation", func() {
		It("should convert A2A card to OASF record matching expected output", func() { //nolint:dupl
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedA2AData, err := decoder.JsonToProto(expectedA2AOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to encode A2A data")

			req := &translationv1.A2AToRecordRequest{
				Data: encodedA2AData,
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

	Context("Record to Kagenti Agent Spec Translation", func() {
		It("should convert OASF 0.8.0 record to Kagenti Agent CRD for weather service", func() {
			encodedRecord, err := decoder.JsonToProto(weatherServiceRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal weather service record")

			// Call translator directly (no gRPC service for this yet)
			agent, err := translator.RecordToKagentiAgentSpec(encodedRecord)
			Expect(err).NotTo(HaveOccurred(), "RecordToKagentiAgentSpec should not fail")
			Expect(agent).NotTo(BeNil(), "Expected Agent CRD in response")

			// Convert agent to YAML for comparison
			actualYAML, err := yaml.Marshal(agent)
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal agent to YAML")

			// Normalize YAML by removing Kubernetes-managed fields and zero values
			// This removes: creationTimestamp: null, status: {}, targetPort: 0
			normalizedActual := normalizeK8sYAML(string(actualYAML))
			normalizedExpected := normalizeK8sYAML(string(weatherServiceOutputYAML))

			// Compare normalized YAML output
			Expect(normalizedActual).To(Equal(normalizedExpected), "Generated YAML should match expected output")
		})

		It("should fail when docker_image locator is missing", func() {
			invalidRecordJSON := `{
				"name": "test-agent",
				"schema_version": "0.8.0",
				"version": "v1.0.0",
				"description": "Test agent",
				"locators": []
			}`

			encodedRecord, err := decoder.JsonToProto([]byte(invalidRecordJSON))
			Expect(err).NotTo(HaveOccurred(), "Failed to parse record JSON")

			agent, err := translator.RecordToKagentiAgentSpec(encodedRecord)
			Expect(err).To(HaveOccurred(), "Should fail when docker_image locator is missing")
			Expect(agent).To(BeNil(), "Agent should be nil on error")
			Expect(err.Error()).To(ContainSubstring("docker_image"), "Error should mention missing locator")
		})
	})
})

// normalizeK8sYAML removes Kubernetes-managed fields from YAML that shouldn't be in applied resources
func normalizeK8sYAML(yamlStr string) string {
	// Remove creationTimestamp: null lines (with proper indentation)
	re := regexp.MustCompile(`(?m)^(\s+)creationTimestamp:\s+null\s*$`)
	yamlStr = re.ReplaceAllString(yamlStr, "")

	// Remove status: {} lines
	re = regexp.MustCompile(`(?m)^status:\s*\{\}\s*$`)
	yamlStr = re.ReplaceAllString(yamlStr, "")

	// Remove targetPort: 0 lines (zero value when targetPort is omitted)
	re = regexp.MustCompile(`(?m)^(\s+)targetPort:\s+0\s*$\n?`)
	yamlStr = re.ReplaceAllString(yamlStr, "")

	// Remove empty metadata blocks (metadata: followed by empty line, then next field)
	// This handles cases like "metadata:\n\n  labels:" or "metadata:\n    spec:"
	re = regexp.MustCompile(`(?m)^(\s+)metadata:\s*$\n\s*$\n(\s+)(labels|name|spec)`)
	yamlStr = re.ReplaceAllString(yamlStr, "$1$3")

	// Remove empty lines (more than one consecutive) - do this multiple times to catch all cases
	for {
		oldStr := yamlStr
		re = regexp.MustCompile(`\n\n\n+`)
		yamlStr = re.ReplaceAllString(yamlStr, "\n\n")
		if yamlStr == oldStr {
			break
		}
	}

	// Trim leading/trailing whitespace
	yamlStr = strings.TrimSpace(yamlStr)

	return yamlStr
}
