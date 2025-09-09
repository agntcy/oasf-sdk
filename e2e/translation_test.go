// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"fmt"
	"time"

	translationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/translation/v1/translationv1grpc"
	corev1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/core/v1"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/translation/v1"
	"github.com/agntcy/oasf-sdk/core/utils"
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
		It("should generate github MCP config from translation record", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := utils.JsonToProto(translationRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToGHCopilotRequest{
				Record: &corev1.EncodedRecord{
					Record: encodedRecord,
				},
			}

			resp, err := client.RecordToGHCopilot(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToGHCopilot should not fail")
			Expect(resp.Data).NotTo(BeNil(), "Expected MCP config data in response")

			mcpData := resp.Data.AsMap()
			Expect(mcpData).NotTo(BeEmpty(), "MCP config data should not be empty")
			mcpConfig := mcpData["mcpConfig"].(map[string]any)
			Expect(mcpConfig).NotTo(BeNil(), "MCP config should be present")

			Expect(mcpConfig).To(HaveKey("servers"), "Should contain servers config")
			Expect(mcpConfig).To(HaveKey("inputs"), "Should contain inputs config")

			servers, ok := mcpConfig["servers"].(map[string]any)
			Expect(ok).To(BeTrue(), "servers should be a map")

			github, ok := servers["github"].(map[string]any)
			Expect(ok).To(BeTrue(), "github should be a map")
			Expect(github).To(HaveKey("command"), "github should have command")
			Expect(github["command"]).To(Equal("docker"), "command should be docker")
		})
	})

	Context("A2A Card Extraction", func() {
		It("should extract A2A card from translation record", func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			encodedRecord, err := utils.JsonToProto(translationRecord)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			req := &translationv1.RecordToA2ARequest{
				Record: &corev1.EncodedRecord{
					Record: encodedRecord,
				},
			}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail")
			Expect(resp.Data).NotTo(BeNil(), "Expected A2A card data in response")

			a2aData := resp.Data.AsMap()
			Expect(a2aData).NotTo(BeEmpty(), "A2A card data should not be empty")

			a2aCard, ok := a2aData["a2aCard"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "a2aCard should be a map")
			Expect(a2aCard).NotTo(BeNil(), "A2A card should be present")

			Expect(a2aCard).To(HaveKey("capabilities"), "A2A card should have capabilities")
			capabilities, ok := a2aCard["capabilities"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "capabilities should be a map")
			Expect(capabilities["streaming"]).To(BeTrue(), "streaming capability should be true")

			Expect(a2aCard).To(HaveKey("description"), "A2A card should have description")
			Expect(a2aCard["description"]).To(ContainSubstring("web searches"), "Description should mention web searches")

			Expect(a2aCard).To(HaveKey("skills"), "A2A card should have skills")
			skills, ok := a2aCard["skills"].([]interface{})
			Expect(ok).To(BeTrue(), "skills should be an array")
			Expect(skills).To(HaveLen(1), "Should have one skill")
			skill, ok := skills[0].(map[string]interface{})
			Expect(ok).To(BeTrue(), "skill should be a map")
			Expect(skill["id"]).To(Equal("browser"), "Skill ID should be browser")

			Expect(a2aCard).To(HaveKey("url"), "A2A card should have url")
			Expect(a2aCard["url"]).To(Equal("http://localhost:8000"), "URL should match extension data")
		})
	})
})
