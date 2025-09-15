// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/types/known/structpb"
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

// ValidateRecord validates a record against a specified schema URL or its embedded schema version.
func (v *Validator) ValidateRecord(record *structpb.Struct, options ...Option) (bool, []string, error) {
	// Apply options
	opts := &option{}
	for _, o := range options {
		o(opts)
	}

	// Validate against schema URL if provided
	if opts.schemaURL != "" {
		schemaErrors, err := v.validateWithSchemaURL(record, opts.schemaURL)
		if err != nil {
			return false, nil, fmt.Errorf("schema URL validation failed: %w", err)
		}

		return len(schemaErrors) == 0, schemaErrors, nil
	}

	// Get schema version
	schemaVersion, err := decoder.GetRecordSchemaVersion(record)
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
	schemaErrors, err := v.validateWithJSONSchema(record, schema)
	if err != nil {
		return false, nil, fmt.Errorf("JSON schema validation failed: %w", err)
	}

	return len(schemaErrors) == 0, schemaErrors, nil
}

func (v *Validator) validateWithJSONSchema(record *structpb.Struct, schema *gojsonschema.Schema) ([]string, error) {
	// Validate JSON against schema
	documentLoader := gojsonschema.NewGoLoader(record)
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

func (v *Validator) validateWithSchemaURL(record *structpb.Struct, schemaURL string) ([]string, error) {
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
