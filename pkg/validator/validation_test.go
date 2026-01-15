// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

// TestValidateWithSchemaURL tests validation behavior with errors and warnings.
func TestValidateWithSchemaURL(t *testing.T) {
	tests := []struct {
		name                 string
		mockResponse         ValidationResponse
		expectedValid        bool
		expectedErrorCount   int
		expectedWarningCount int
		expectError          bool
	}{
		{
			name: "no errors or warnings",
			mockResponse: ValidationResponse{
				Warnings:     []ValidationError{},
				Errors:       []ValidationError{},
				ErrorCount:   0,
				WarningCount: 0,
			},
			expectedValid:        true,
			expectedErrorCount:   0,
			expectedWarningCount: 0,
			expectError:          false,
		},
		{
			name: "only errors",
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
			expectedValid:        false,
			expectedErrorCount:   1,
			expectedWarningCount: 0,
			expectError:          false,
		},
		{
			name: "only warnings",
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
			expectedValid:        true, // Warnings don't affect validity
			expectedErrorCount:   0,
			expectedWarningCount: 1,
			expectError:          false,
		},
		{
			name: "errors and warnings",
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
			expectedValid:        false, // Fails because of errors
			expectedErrorCount:   1,
			expectedWarningCount: 1,
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				if err := json.NewEncoder(w).Encode(tt.mockResponse); err != nil {
					t.Errorf("Failed to encode mock response: %v", err)
				}
			}))
			defer server.Close()

			// Create a validator with the mock server URL
			validator, err := New(server.URL)
			if err != nil {
				t.Fatalf("Failed to create validator: %v", err)
			}

			// Create a test record
			record, err := structpb.NewStruct(map[string]any{
				"schema_version": "0.8.0",
				"data": map[string]any{
					"name": "test",
				},
			})
			if err != nil {
				t.Fatalf("Failed to create test record: %v", err)
			}

			// Validate with the mock server URL
			valid, errors, warnings, err := validator.ValidateRecord(context.Background(), record)

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

			if len(errors) != tt.expectedErrorCount {
				t.Errorf("Expected %d errors, got %d errors: %v", tt.expectedErrorCount, len(errors), errors)
			}

			if len(warnings) != tt.expectedWarningCount {
				t.Errorf("Expected %d warnings, got %d warnings: %v", tt.expectedWarningCount, len(warnings), warnings)
			}
		})
	}
}

// TestValidateWithSchemaURL_ConstraintFailed tests that constraint information is included in error messages.
func TestValidateWithSchemaURL_ConstraintFailed(t *testing.T) {
	// Create a mock response with constraint_failed error
	mockResponse := ValidationResponse{
		Warnings: []ValidationError{},
		Errors: []ValidationError{
			{
				Error:   "constraint_failed",
				Message: "Constraint failed: \"at_least_one\" from object \"mcp_server\" at \"data.servers[0]\"; expected at least one constraint attribute, but got none.",
				Constraint: map[string]any{
					"at_least_one": []any{"url", "command"},
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

		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	// Create a validator with the mock server URL
	validator, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test record
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "0.8.0",
		"data": map[string]any{
			"servers": []any{
				map[string]any{},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	// Validate with the mock server URL
	valid, errors, warnings, err := validator.ValidateRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if valid {
		t.Error("Expected validation to fail")
	}

	if len(errors) != 1 {
		t.Fatalf("Expected 1 error message, got %d", len(errors))
	}

	if len(warnings) != 0 {
		t.Fatalf("Expected 0 warnings, got %d", len(warnings))
	}

	// Check that the error message includes constraint information
	errorMsg := errors[0]
	expectedConstraintJSON := `{"at_least_one":["url","command"]}`

	if !contains(errorMsg, "Constraint:") {
		t.Errorf("Error message should mention 'Constraint:'. Got: %s", errorMsg)
	}

	if !contains(errorMsg, expectedConstraintJSON) {
		t.Errorf("Error message should include constraint JSON '%s'. Got: %s", expectedConstraintJSON, errorMsg)
	}
}

// TestValidateWithSchemaURL_WarningsOnly tests that warnings don't affect validity.
func TestValidateWithSchemaURL_WarningsOnly(t *testing.T) {
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

		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	// Create a validator with the mock server URL
	validator, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Create a test record
	record, err := structpb.NewStruct(map[string]any{
		"schema_version": "0.8.0",
		"data": map[string]any{
			"name": "test",
		},
	})
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	// Validate with the mock server URL
	valid, errors, warnings, err := validator.ValidateRecord(context.Background(), record)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Warnings don't affect validity, so it should be valid
	if !valid {
		t.Error("Expected validation to pass with only warnings")
	}

	if len(errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(errors))
	}

	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(warnings))
	}
}

// Helper function to check if a string contains a substring.
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
