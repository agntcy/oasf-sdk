// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"google.golang.org/protobuf/types/known/structpb"
)

const defaultHTTPTimeoutSeconds = 30

type Validator struct {
	httpClient *http.Client
}

// ValidationError represents a single validation error from the API.
type ValidationError struct {
	Error         string         `json:"error"`
	Message       string         `json:"message"`
	Value         any            `json:"value,omitempty"`
	Attribute     string         `json:"attribute,omitempty"`
	ValueType     string         `json:"value_type,omitempty"`
	AttributePath string         `json:"attribute_path,omitempty"`
	ExpectedType  string         `json:"expected_type,omitempty"`
	Constraint    map[string]any `json:"constraint,omitempty"`
	ObjectName    string         `json:"object_name,omitempty"`
}

// ValidationResponse represents the response from the validator API.
type ValidationResponse struct {
	Warnings     []ValidationError `json:"warnings"`
	Errors       []ValidationError `json:"errors"`
	ErrorCount   int               `json:"error_count"`
	WarningCount int               `json:"warning_count"`
}

func New() (*Validator, error) {
	return &Validator{
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeoutSeconds * time.Second,
		},
	}, nil
}

// ValidateRecord validates a record against a specified schema URL.
// A schema URL must be provided via the WithSchemaURL option.
func (v *Validator) ValidateRecord(ctx context.Context, record *structpb.Struct, options ...Option) (bool, []string, error) {
	// Apply options with defaults
	opts := &option{
		strict: true,
	}
	for _, o := range options {
		o(opts)
	}

	// Schema URL is required
	if opts.schemaURL == "" {
		return false, nil, fmt.Errorf("schema URL is required, use WithSchemaURL option")
	}

	// Validate against schema URL
	errorMessages, warningMessages, err := v.validateWithSchemaURL(ctx, record, opts.schemaURL)
	if err != nil {
		return false, nil, fmt.Errorf("schema URL validation failed: %w", err)
	}

	// Combine errors and warnings
	errorMessages = append(errorMessages, warningMessages...)

	if opts.strict {
		// In strict mode, warnings are treated as errors
		return len(errorMessages) == 0, errorMessages, nil
	} else {
		// In non-strict mode, only errors matter for validation result
		// but we still return warnings in the messages list (excluding appended warnings from error count)
		originalErrorCount := len(errorMessages) - len(warningMessages)

		return originalErrorCount == 0, errorMessages, nil
	}
}

func (v *Validator) validateWithSchemaURL(ctx context.Context, record *structpb.Struct, schemaURL string) ([]string, []string, error) {
	// Get schema version from the record
	schemaVersion, err := decoder.GetRecordSchemaVersion(record)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get schema version from record: %w", err)
	}

	// Construct the full validation URL
	validationURL := constructValidationURL(schemaURL, schemaVersion)

	// Convert record to JSON for the POST request
	recordJSON, err := json.Marshal(record)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal record to JSON: %w", err)
	}

	// Create POST request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, validationURL, bytes.NewBuffer(recordJSON))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create POST request to %s: %w", validationURL, err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send POST request to %s: %w", validationURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("failed to validate record at URL %s: HTTP %d", validationURL, resp.StatusCode)
	}

	// Parse response
	var validationResp ValidationResponse

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&validationResp); err != nil {
		return nil, nil, fmt.Errorf("failed to decode validation response from URL %s: %w", validationURL, err)
	}

	// Convert errors to string format
	errorMessages := make([]string, 0, len(validationResp.Errors))
	for _, err := range validationResp.Errors {
		errorMsg := "Validation Error: " + err.Message
		if err.AttributePath != "" {
			errorMsg = fmt.Sprintf("Validation Error at %s: %s", err.AttributePath, err.Message)
		}

		// Add constraint information if this is a constraint_failed error
		if err.Error == "constraint_failed" && err.Constraint != nil {
			constraintJSON, marshalErr := json.Marshal(err.Constraint)
			if marshalErr == nil {
				errorMsg = fmt.Sprintf("%s Constraint: %s", errorMsg, string(constraintJSON))
			}
		}

		errorMessages = append(errorMessages, errorMsg)
	}

	// Convert warnings to string format
	warningMessages := make([]string, 0, len(validationResp.Warnings))
	for _, warning := range validationResp.Warnings {
		warningMsg := "Validation Warning: " + warning.Message
		if warning.AttributePath != "" {
			warningMsg = fmt.Sprintf("Validation Warning at %s: %s", warning.AttributePath, warning.Message)
		}

		warningMessages = append(warningMessages, warningMsg)
	}

	return errorMessages, warningMessages, nil
}

// constructValidationURL builds the full validation URL from a base URL and schema version.
func constructValidationURL(baseURL, schemaVersion string) string {
	// Normalize the base URL (remove trailing slash if present)
	normalizedURL := strings.TrimSuffix(baseURL, "/")

	// Add protocol if missing (default to http:// for localhost or IP addresses)
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "http://" + normalizedURL
	}

	// Determine the object type based on schema version
	// Version 0.3.1 uses "agent", while later versions use "record"
	objectType := "record"
	if schemaVersion == "0.3.1" || schemaVersion == "v0.3.1" {
		objectType = "agent"
	}

	// Construct the full validation URL
	return fmt.Sprintf("%s/api/%s/validate/object/%s", normalizedURL, schemaVersion, objectType)
}
