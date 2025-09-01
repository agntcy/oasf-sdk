// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	translationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/translation/v1/translationv1grpc"
	translationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/translation/v1"
	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
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
			var record objectsv3.Record
			err := json.Unmarshal(translationRecord, &record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &translationv1.RecordToVSCodeCopilotRequest{
				Record: &record,
			}

			resp, err := client.RecordToVSCodeCopilot(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToVSCodeCopilot should not fail")
			Expect(resp.Data).NotTo(BeNil(), "Expected MCP config data in response")

			mcpData := resp.Data.AsMap()
			Expect(mcpData).NotTo(BeEmpty(), "MCP config data should not be empty")
			Expect(mcpData).To(HaveKey("mcpConfig"), "MCP config should have mcpConfig")

			mcpConfig, ok := mcpData["mcpConfig"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "mcpConfig should be a map")
			Expect(mcpConfig).To(HaveKey("servers"), "Should contain servers config")

			servers, ok := mcpConfig["servers"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "servers should be a map")

			github, ok := servers["github"].(map[string]interface{})
			Expect(ok).To(BeTrue(), "github should be a map")
			Expect(github).To(HaveKey("command"), "github should have command")
			Expect(github["command"]).To(Equal("docker"), "command should be docker")
		})
	})

	Context("A2A Card Extraction", func() {
		It("should extract A2A card from translation record", func() {
			var record objectsv3.Record
			err := json.Unmarshal(translationRecord, &record)
			Expect(err).NotTo(HaveOccurred(), "Failed to unmarshal translation record")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &translationv1.RecordToA2ARequest{
				Record: &record,
			}

			resp, err := client.RecordToA2A(ctx, req)
			Expect(err).NotTo(HaveOccurred(), "RecordToA2A should not fail")
			Expect(resp.Data).NotTo(BeNil(), "Expected A2A card data in response")

			a2aData := resp.Data.AsMap()
			Expect(a2aData).NotTo(BeEmpty(), "A2A card data should not be empty")

			a2aCard, ok := a2aData["a2aCard"].(map[string]interface{})
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
