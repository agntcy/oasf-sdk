// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator_test

import (
	"encoding/json"
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/agntcy/oasf-sdk/pkg/translator"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestA2ARoundtripPreservesFields demonstrates that the refactored translator
// preserves all fields during A2A -> OASF -> A2A roundtrip translation.
// This addresses the issue where manually defined Go types would drop fields.
func TestA2ARoundtripPreservesFields(t *testing.T) {
	// Create an A2A card with fields that may not be in manually defined structs
	originalA2AJSON := `{
		"a2aCard": {
			"name": "test-agent",
			"description": "Test agent for roundtrip validation",
			"url": "https://example.com",
			"protocol_version": "1.0",
			"version": "1.0.0",
			"capabilities": {
				"streaming": true,
				"pushNotifications": false,
				"customCapability": true
			},
			"defaultInputModes": ["text", "audio"],
			"defaultOutputModes": ["text", "video"],
			"skills": [
				{
					"id": "skill1",
					"name": "Test Skill",
					"description": "A test skill",
					"customField": "This field would be lost with manual types"
				}
			],
			"provider": {
				"name": "Test Provider",
				"url": "https://provider.example.com"
			},
			"documentation_url": "https://docs.example.com",
			"icon_url": "https://icon.example.com/icon.png"
		}
	}`

	// Parse the original A2A card
	originalA2A, err := decoder.JsonToProto([]byte(originalA2AJSON))
	if err != nil {
		t.Fatalf("Failed to parse original A2A JSON: %v", err)
	}

	// Step 1: Convert A2A -> OASF Record
	record, err := translator.A2AToRecord(originalA2A)
	if err != nil {
		t.Fatalf("Failed to convert A2A to Record: %v", err)
	}

	// Step 2: Extract A2A card from OASF Record
	extractedA2A, err := translator.RecordToA2A(record)
	if err != nil {
		t.Fatalf("Failed to extract A2A from Record: %v", err)
	}

	// Debug: Print the extracted structure
	extractedDebugJSON, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(extractedA2A)
	t.Logf("Extracted A2A structure:\n%s", string(extractedDebugJSON))

	// Step 3: Compare original and extracted A2A cards
	originalJSON, err := protojson.Marshal(originalA2A.GetFields()["a2aCard"].GetStructValue())
	if err != nil {
		t.Fatalf("Failed to marshal original A2A: %v", err)
	}

	extractedJSON, err := protojson.Marshal(extractedA2A)
	if err != nil {
		t.Fatalf("Failed to marshal extracted A2A: %v", err)
	}

	// Parse both JSONs to compare as maps (to handle field ordering)
	var originalMap, extractedMap map[string]any
	if err := json.Unmarshal(originalJSON, &originalMap); err != nil {
		t.Fatalf("Failed to unmarshal original JSON: %v", err)
	}

	if err := json.Unmarshal(extractedJSON, &extractedMap); err != nil {
		t.Fatalf("Failed to unmarshal extracted JSON: %v", err)
	}

	// Verify all original fields are present
	verifyFieldsPresent(t, originalMap, extractedMap, "")
}

// verifyFieldsPresent recursively checks that all fields in original are present in extracted.
func verifyFieldsPresent(t *testing.T, original, extracted map[string]any, path string) {
	t.Helper()

	for key, originalValue := range original {
		currentPath := key
		if path != "" {
			currentPath = path + "." + key
		}

		extractedValue, exists := extracted[key]
		if !exists {
			t.Errorf("Field %s is missing in extracted A2A card (value was: %v)", currentPath, originalValue)

			continue
		}

		// Recursively check nested maps
		if originalMap, ok := originalValue.(map[string]any); ok {
			if extractedMap, ok := extractedValue.(map[string]any); ok {
				verifyFieldsPresent(t, originalMap, extractedMap, currentPath)
			}
		}

		// Check arrays
		if originalArray, ok := originalValue.([]any); ok {
			if extractedArray, ok := extractedValue.([]any); ok {
				if len(originalArray) != len(extractedArray) {
					t.Errorf("Field %s array length mismatch: original=%d, extracted=%d",
						currentPath, len(originalArray), len(extractedArray))
				}
			}
		}
	}
}
