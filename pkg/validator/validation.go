// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/types/known/structpb"
)

type Validator struct {
	schemas    map[string]*gojsonschema.Schema
	httpClient *http.Client
}

// ValidationError represents a single validation error from the API
type ValidationError struct {
	Error         string `json:"error"`
	Message       string `json:"message"`
	Value         any    `json:"value,omitempty"`
	Attribute     string `json:"attribute,omitempty"`
	ValueType     string `json:"value_type,omitempty"`
	AttributePath string `json:"attribute_path,omitempty"`
	ExpectedType  string `json:"expected_type,omitempty"`
}

// ValidationResponse represents the response from the validator API
type ValidationResponse struct {
	Warnings     []ValidationError `json:"warnings"`
	Errors       []ValidationError `json:"errors"`
	ErrorCount   int               `json:"error_count"`
	WarningCount int               `json:"warning_count"`
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
	// Get schema version from the record
	schemaVersion, err := decoder.GetRecordSchemaVersion(record)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema version from record: %w", err)
	}

	// Normalize the base URL (remove trailing slash if present)
	baseURL := strings.TrimSuffix(schemaURL, "/")

	// Construct the validation URL
	validationURL := fmt.Sprintf("%s/api/%s/validate/object/record", baseURL, schemaVersion)

	// Convert record to JSON for the POST request
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record to JSON: %w", err)
	}

	// Create POST request
	req, err := http.NewRequest("POST", validationURL, bytes.NewBuffer(recordJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request to %s: %w", validationURL, err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send POST request to %s: %w", validationURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to validate record at URL %s: HTTP %d", validationURL, resp.StatusCode)
	}

	// Parse response
	var validationResp ValidationResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&validationResp); err != nil {
		return nil, fmt.Errorf("failed to decode validation response from URL %s: %w", validationURL, err)
	}

	// Convert errors to string format
	var errors []string
	for _, err := range validationResp.Errors {
		errorMsg := fmt.Sprintf("Validation Error: %s", err.Message)
		if err.AttributePath != "" {
			errorMsg = fmt.Sprintf("Validation Error at %s: %s", err.AttributePath, err.Message)
		}
		errors = append(errors, errorMsg)
	}

	// Also include warnings as errors for consistency with the existing API
	for _, warning := range validationResp.Warnings {
		warningMsg := fmt.Sprintf("Validation Warning: %s", warning.Message)
		if warning.AttributePath != "" {
			warningMsg = fmt.Sprintf("Validation Warning at %s: %s", warning.AttributePath, warning.Message)
		}
		errors = append(errors, warningMsg)
	}

	return errors, nil
}
