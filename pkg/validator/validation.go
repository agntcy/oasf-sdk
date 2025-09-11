// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/validation/v1"
	corev1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/core/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/xeipuuv/gojsonschema"
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

func (v *Validator) ValidateRecord(req *validationv1.ValidateRecordRequest) (bool, []string, error) {
	// Validate against schema URL if provided
	if req.GetSchemaUrl() != "" {
		schemaErrors, err := v.validateWithSchemaURL(req.GetRecord(), req.GetSchemaUrl())
		if err != nil {
			return false, nil, fmt.Errorf("schema URL validation failed: %w", err)
		}

		return len(schemaErrors) == 0, schemaErrors, nil
	}

	// Get schema version
	schemaVersion, err := decoder.GetRecordSchemaVersion(req.GetRecord())
	if err != nil {
		return false, nil, fmt.Errorf("failed to get schema version: %w", err)
	}

	// Find schema for given version
	schema, schemaExists := v.schemas[schemaVersion]
	if !schemaExists {
		var availableVersions []string
		for version := range v.schemas {
			availableVersions = append(availableVersions, version)
		}

		return false, nil, fmt.Errorf("no schema found for version %s. Available versions: %v", schemaVersion, availableVersions)
	}

	// Validate against embedded schema
	schemaErrors, err := v.validateWithJSONSchema(req.GetRecord(), schema)
	if err != nil {
		return false, nil, fmt.Errorf("JSON schema validation failed: %w", err)
	}

	return len(schemaErrors) == 0, schemaErrors, nil
}

func (v *Validator) validateWithJSONSchema(record *corev1.Object, schema *gojsonschema.Schema) ([]string, error) {
	// Validate JSON against schema
	documentLoader := gojsonschema.NewGoLoader(record.GetData())
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %w", err)
	}

	// Collect validation errors
	var errors []string
	if !result.Valid() {
		for _, desc := range result.Errors() {
			errors = append(errors, fmt.Sprintf("JSON Schema: %s", desc.String()))
		}
	}

	return errors, nil
}

func (v *Validator) validateWithSchemaURL(record *corev1.Object, schemaURL string) ([]string, error) {
	resp, err := v.httpClient.Get(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from URL %s: %w", schemaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch schema from URL %s: HTTP %d", schemaURL, resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	var schemaData any
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

	return v.validateWithJSONSchema(record, schema)
}
