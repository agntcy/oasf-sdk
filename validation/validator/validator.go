// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	corev1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/core/v1"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type Validator struct {
	schemas    map[string]*gojsonschema.Schema
	httpClient *http.Client
}

func New() (*Validator, error) {
	schemas, err := loadEmbeddedSchemas()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded schemas: %w", err)
	}

	return &Validator{
		schemas: schemas,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (v *Validator) ValidateRecord(req *corev1.DecodedRecord, schemaURL string) (bool, []string, error) {
	if req.GetRecord() == nil {
		return false, nil, fmt.Errorf("record is nil")
	}

	if schemaURL != "" {
		schemaErrors, err := v.validateWithSchemaURL(req, schemaURL)
		if err != nil {
			return false, nil, fmt.Errorf("schema URL validation failed: %w", err)
		}

		return len(schemaErrors) == 0, schemaErrors, nil
	}

	schemaVersion := getSchemaVersion(req)
	schema, schemaExists := v.schemas[schemaVersion]
	if !schemaExists {
		var availableVersions []string
		for version := range v.schemas {
			availableVersions = append(availableVersions, version)
		}

		return false, nil, fmt.Errorf("no schema found for version %s. Available versions: %v", schemaVersion, availableVersions)
	}

	schemaErrors, err := v.validateWithJSONSchema(req, schema)
	if err != nil {
		return false, nil, fmt.Errorf("JSON schema validation failed: %w", err)
	}

	return len(schemaErrors) == 0, schemaErrors, nil
}

func (v *Validator) validateWithJSONSchema(req *corev1.DecodedRecord, schema *gojsonschema.Schema) ([]string, error) {
	marshaler := &protojson.MarshalOptions{
		UseProtoNames: true,
	}
	jsonBytes, err := marshaler.Marshal(getRecordObject(req))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record to JSON: %w", err)
	}

	var recordData map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &recordData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Convert size fields from strings to integers for validation
	if locators, ok := recordData["locators"].([]any); ok {
		for _, locatorIntf := range locators {
			if locator, ok := locatorIntf.(map[string]any); ok {
				if sizeStr, ok := locator["size"].(string); ok {
					// Try to convert string to integer
					var size int64
					if _, err := fmt.Sscanf(sizeStr, "%d", &size); err == nil {
						locator["size"] = size
					}
				}
			}
		}
	}

	documentLoader := gojsonschema.NewGoLoader(recordData)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %w", err)
	}

	var errors []string
	if !result.Valid() {
		for _, desc := range result.Errors() {
			errors = append(errors, fmt.Sprintf("JSON Schema: %s", desc.String()))
		}
	}

	return errors, nil
}

func (v *Validator) validateWithSchemaURL(req *corev1.DecodedRecord, schemaURL string) ([]string, error) {
	resp, err := v.httpClient.Get(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from URL %s: %w", schemaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch schema from URL %s: HTTP %d", schemaURL, resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	var schemaData interface{}
	if err := decoder.Decode(&schemaData); err != nil {
		return nil, fmt.Errorf("failed to decode schema JSON from URL %s: %w", schemaURL, err)
	}

	schemaBytes, err := json.Marshal(schemaData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema from URL %s: %w", schemaURL, err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaBytes))
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema from URL %s: %w", schemaURL, err)
	}

	return v.validateWithJSONSchema(req, schema)
}

func getSchemaVersion(req *corev1.DecodedRecord) string {
	switch req.GetRecord().(type) {
	case *corev1.DecodedRecord_V1Alpha1:
		return req.GetV1Alpha1().SchemaVersion
	default:
		return ""
	}
}

func getRecordObject(req *corev1.DecodedRecord) proto.Message {
	switch req.GetRecord().(type) {
	case *corev1.DecodedRecord_V1Alpha1:
		return req.GetV1Alpha1()
	default:
		return nil
	}
}
