// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// TestValidateWithSchemaURL_StrictMode tests the strict mode validation behavior
func TestValidateWithSchemaURL_StrictMode(t *testing.T) {
	tests := []struct {
		name             string
		strict           bool
		mockResponse     ValidationResponse
		expectedValid    bool
		expectedMsgCount int
		expectError      bool
	}{
		{
			name:   "strict mode - no errors or warnings",
			strict: true,
			mockResponse: ValidationResponse{
				Warnings:     []ValidationError{},
				Errors:       []ValidationError{},
				ErrorCount:   0,
				WarningCount: 0,
			},
			expectedValid:    true,
			expectedMsgCount: 0,
			expectError:      false,
		},
		{
			name:   "strict mode - only errors",
			strict: true,
			mockResponse: ValidationResponse{
				Warnings: []ValidationError{},
				Errors: []ValidationError{
					{
						Error:         "attribute_required_missing",
						Message:       "Required attribute is missing",
						AttributePath: "data.name",
					},
				},
				ErrorCount:   1,
				WarningCount: 0,
			},
			expectedValid:    false,
			expectedMsgCount: 1,
			expectError:      false,
		},
		{
			name:   "strict mode - only warnings (should fail)",
			strict: true,
			mockResponse: ValidationResponse{
				Warnings: []ValidationError{
					{
						Error:         "deprecated_attribute",
						Message:       "Attribute is deprecated",
						AttributePath: "data.old_field",
					},
				},
				Errors:       []ValidationError{},
				ErrorCount:   0,
				WarningCount: 1,
			},
			expectedValid:    false, // Strict mode treats warnings as errors
			expectedMsgCount: 1,
			expectError:      false,
		},
		{
			name:   "strict mode - errors and warnings",
			strict: true,
			mockResponse: ValidationResponse{
				Warnings: []ValidationError{
					{
						Error:         "deprecated_attribute",
						Message:       "Attribute is deprecated",
						AttributePath: "data.old_field",
					},
				},
				Errors: []ValidationError{
					{
						Error:         "attribute_required_missing",
						Message:       "Required attribute is missing",
						AttributePath: "data.name",
					},
				},
				ErrorCount:   1,
				WarningCount: 1,
			},
			expectedValid:    false,
			expectedMsgCount: 2,
			expectError:      false,
		},
		{
			name:   "non-strict mode - no errors or warnings",
			strict: false,
			mockResponse: ValidationResponse{
				Warnings:     []ValidationError{},
				Errors:       []ValidationError{},
				ErrorCount:   0,
				WarningCount: 0,
			},
			expectedValid:    true,
			expectedMsgCount: 0,
			expectError:      false,
		},
		{
			name:   "non-strict mode - only errors",
			strict: false,
			mockResponse: ValidationResponse{
				Warnings: []ValidationError{},
				Errors: []ValidationError{
					{
						Error:         "attribute_required_missing",
						Message:       "Required attribute is missing",
						AttributePath: "data.name",
					},
				},
				ErrorCount:   1,
				WarningCount: 0,
			},
			expectedValid:    false,
			expectedMsgCount: 1,
			expectError:      false,
		},
		{
			name:   "non-strict mode - only warnings (should pass)",
			strict: false,
			mockResponse: ValidationResponse{
				Warnings: []ValidationError{
					{
						Error:         "deprecated_attribute",
						Message:       "Attribute is deprecated",
						AttributePath: "data.old_field",
					},
				},
				Errors:       []ValidationError{},
				ErrorCount:   0,
				WarningCount: 1,
			},
			expectedValid:    true, // Non-strict mode ignores warnings for validation result
			expectedMsgCount: 1,    // But warnings are still included in messages
			expectError:      false,
		},
		{
			name:   "non-strict mode - errors and warnings",
			strict: false,
			mockResponse: ValidationResponse{
				Warnings: []ValidationError{
					{
						Error:         "deprecated_attribute",
						Message:       "Attribute is deprecated",
						AttributePath: "data.old_field",
					},
				},
				Errors: []ValidationError{
					{
						Error:         "attribute_required_missing",
						Message:       "Required attribute is missing",
						AttributePath: "data.name",
					},
				},
				ErrorCount:   1,
				WarningCount: 1,
			},
			expectedValid:    false, // Still fails because of errors
			expectedMsgCount: 2,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.mockResponse)
			}))
			defer server.Close()

			// Create a validator
			validator, err := New()
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			// Create a test record
			record, err := structpb.NewStruct(map[string]interface{}{
				"schema_version": "0.8.0",
				"data": map[string]interface{}{
					"name": "test",
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test record: %v", err)
			}

			// Validate with the mock server URL
			var valid bool
			var messages []string
			if tt.strict {
				valid, messages, err = validator.ValidateRecord(record, WithSchemaURL(server.URL), WithStrict(true))
			} else {
				valid, messages, err = validator.ValidateRecord(record, WithSchemaURL(server.URL), WithStrict(false))
			}

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if valid != tt.expectedValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.expectedValid, valid)
			}

			if len(messages) != tt.expectedMsgCount {
				t.Errorf("Expected %d messages, got %d messages: %v", tt.expectedMsgCount, len(messages), messages)
			}
		})
	}
}

// TestValidateWithSchemaURL_ConstraintFailed tests that constraint information is included in error messages
func TestValidateWithSchemaURL_ConstraintFailed(t *testing.T) {
	// Create a mock response with constraint_failed error
	mockResponse := ValidationResponse{
		Warnings: []ValidationError{},
		Errors: []ValidationError{
			{
				Error:   "constraint_failed",
				Message: "Constraint failed: \"at_least_one\" from object \"mcp_server\" at \"data.servers[0]\"; expected at least one constraint attribute, but got none.",
				Constraint: map[string]interface{}{
					"at_least_one": []interface{}{"url", "command"},
				},
				ObjectName:    "mcp_server",
				AttributePath: "data.servers[0]",
			},
		},
		ErrorCount:   1,
		WarningCount: 0,
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create a validator
	validator, err := New()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test record
	record, err := structpb.NewStruct(map[string]interface{}{
		"schema_version": "0.8.0",
		"data": map[string]interface{}{
			"servers": []interface{}{
				map[string]interface{}{},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	// Validate with the mock server URL
	valid, messages, err := validator.ValidateRecord(record, WithSchemaURL(server.URL))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if valid {
		t.Error("Expected validation to fail")
	}

	if len(messages) != 1 {
		t.Fatalf("Expected 1 error message, got %d", len(messages))
	}

	// Check that the error message includes constraint information
	errorMsg := messages[0]
	expectedConstraintJSON := `{"at_least_one":["url","command"]}`

	if !contains(errorMsg, "Constraint:") {
		t.Errorf("Error message should mention 'Constraint:'. Got: %s", errorMsg)
	}

	if !contains(errorMsg, expectedConstraintJSON) {
		t.Errorf("Error message should include constraint JSON '%s'. Got: %s", expectedConstraintJSON, errorMsg)
	}
}

// TestValidateWithSchemaURL_DefaultStrictMode tests that strict mode is enabled by default
func TestValidateWithSchemaURL_DefaultStrictMode(t *testing.T) {
	// Create a mock response with only warnings
	mockResponse := ValidationResponse{
		Warnings: []ValidationError{
			{
				Error:         "deprecated_attribute",
				Message:       "Attribute is deprecated",
				AttributePath: "data.old_field",
			},
		},
		Errors:       []ValidationError{},
		ErrorCount:   0,
		WarningCount: 1,
	}

	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mockResponse)
	}))
	defer server.Close()

	// Create a validator
	validator, err := New()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test record
	record, err := structpb.NewStruct(map[string]interface{}{
		"schema_version": "0.8.0",
		"data": map[string]interface{}{
			"name": "test",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	// Validate WITHOUT specifying WithStrict - should default to strict=true
	valid, messages, err := validator.ValidateRecord(record, WithSchemaURL(server.URL))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// In default strict mode, warnings should cause validation to fail
	if valid {
		t.Error("Expected validation to fail in default strict mode with warnings")
	}

	if len(messages) != 1 {
		t.Errorf("Expected 1 warning message, got %d", len(messages))
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
