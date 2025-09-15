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

	Context("MCP Config Generation", func() {
		It("should generate github MCP config matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToGHCopilotRequest{
				Record: encodedRecord,
			}

			resp, err := client.RecordToGHCopilot(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToGHCopilot should not fail")
			Expect(resp.Data).NotTo(BeNil(), "Expected MCP config data in response")

			// Convert response to JSON for comparison
			actualJSON, err := json.MarshalIndent(resp.Data.AsMap(), "", "  ")
			Expect(err).NotTo(HaveOccurred(), "Failed to marshal MCP config to JSON")

			// Parse expected output
			var expectedOutput map[string]interface{}
			err = json.Unmarshal(expectedGHCopilotOutput, &expectedOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal expected MCP output")

			// Parse actual output for comparison
			var actualOutput map[string]interface{}
			err = json.Unmarshal(actualJSON, &actualOutput)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal actual MCP output")

			// Compare structure against expected output
			Expect(actualOutput).To(Equal(expectedOutput), "MCP config should match expected output")
		})
	})

	Context("A2A Card Extraction", func() {
		It("should extract A2A card matching expected output", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := decoder.JsonToProto(translationRecord)
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
	})
})
