// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testSchemaVersion = "0.7.0"

const invalidVersion = "99.99.99"

// mockSchemaResponse returns a mock schema JSON response with $defs section.
func mockSchemaResponse() map[string]any {
	return map[string]any{
		"$schema": "http://json-schema.org/draft-07/schema#",
		"type":    "object",
		"$defs": map[string]any{
			"skills": map[string]any{
				"text_classification": map[string]any{
					"id":   10001,
					"name": "text_classification",
				},
				"natural_language_processing": map[string]any{
					"id":   10002,
					"name": "natural_language_processing",
				},
			},
			"domains": map[string]any{
				"lean_manufacturing": map[string]any{
					"id":   20001,
					"name": "lean_manufacturing",
				},
				"artificial_intelligence": map[string]any{
					"id":   20002,
					"name": "artificial_intelligence",
				},
			},
			"objects": map[string]any{
				"record": map[string]any{
					"type": "object",
				},
			},
			"modules": map[string]any{
				"mcp_server": map[string]any{
					"type": "object",
				},
			},
		},
	}
}

// createMockServer creates a mock HTTP server for schema tests.
func createMockServer(t *testing.T, version string, expectError bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Handle versions endpoint
		if r.URL.Path == "/api/versions" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			versionsResp := VersionsResponse{
				Default: struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					Version: "0.8.0",
					URL:     r.Host + "/api/0.8.0",
				},
				Versions: []struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					{Version: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{Version: "0.8.0", URL: r.Host + "/0.8.0/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		// Verify the URL path matches expected pattern
		expectedPath := "/schema/" + version + "/objects/record"

		if !contains(r.URL.Path, expectedPath) {
			t.Errorf("Expected URL path to contain %s, got %s", expectedPath, r.URL.Path)
		}

		if expectError {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		mockSchema := mockSchemaResponse()
		if err := json.NewEncoder(w).Encode(mockSchema); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
}

// validateSchemaContent validates that schema content is valid JSON.
func validateSchemaContent(t *testing.T, content []byte) {
	t.Helper()

	if len(content) == 0 {
		t.Errorf("GetRecordSchemaContent() returned empty content")
	}

	var jsonMap map[string]any
	if err := json.Unmarshal(content, &jsonMap); err != nil {
		t.Errorf("GetRecordSchemaContent() returned invalid JSON: %v", err)
	}
}

func TestGetRecordSchemaContent(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "valid version 0.7.0",
			version:     "0.7.0",
			expectError: false,
		},
		{
			name:        "valid version 0.8.0",
			version:     "0.8.0",
			expectError: false,
		},
		{
			name:        "invalid version",
			version:     invalidVersion,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServer(t, tt.version, tt.expectError)
			defer server.Close()

			schema, err := New(server.URL)
			if err != nil {
				t.Fatalf("Failed to create schema: %v", err)
			}

			content, err := schema.GetRecordSchemaContent(context.Background(), WithVersion(tt.version))
			if tt.expectError {
				if err == nil {
					t.Errorf("GetRecordSchemaContent() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("GetRecordSchemaContent() unexpected error: %v", err)
			}

			validateSchemaContent(t, content)
		})
	}
}

func TestGetSchema(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		typ         SchemaType
		schemaName  string
		expectError bool
	}{
		{
			name:        "valid objects/record for 0.8.0",
			version:     "0.8.0",
			typ:         SchemaTypeObjects,
			schemaName:  "record",
			expectError: false,
		},
		{
			name:        "valid modules",
			version:     "0.8.0",
			typ:         SchemaTypeModules,
			schemaName:  "integration/mcp",
			expectError: false,
		},
		{
			name:        "valid skills",
			version:     "0.8.0",
			typ:         SchemaTypeSkills,
			schemaName:  "natural_language_processing",
			expectError: false,
		},
		{
			name:        "valid domains",
			version:     "0.8.0",
			typ:         SchemaTypeDomains,
			schemaName:  "artificial_intelligence",
			expectError: false,
		},
		{
			name:        "invalid version",
			version:     invalidVersion,
			typ:         SchemaTypeObjects,
			schemaName:  "record",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServerWithVersionCheck(t, tt.expectError && tt.version == invalidVersion)
			defer server.Close()

			schema, err := New(server.URL)
			if err != nil {
				t.Fatalf("Failed to create schema: %v", err)
			}

			var opts []SchemaOption

			if tt.version != "" {
				opts = append(opts, WithVersion(tt.version))
			}

			content, err := schema.GetSchema(context.Background(), tt.typ, tt.schemaName, opts...)
			if tt.expectError {
				if err == nil {
					t.Errorf("GetSchema() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("GetSchema() unexpected error: %v", err)
			}

			validateSchemaContent(t, content)
		})
	}
}

// createMockServerWithVersionCheck creates a mock server that validates versions.
func createMockServerWithVersionCheck(t *testing.T, checkVersion bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle versions endpoint
		if r.URL.Path == apiVersionsPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			versionsResp := VersionsResponse{
				Default: struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					Version: "0.8.0",
					URL:     r.Host + "/api/0.8.0",
				},
				Versions: []struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					{Version: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{Version: "0.8.0", URL: r.Host + "/0.8.0/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		// Check if this is a schema request with invalid version
		if checkVersion && strings.Contains(r.URL.Path, "/schema/") {
			// Extract version from path (e.g., /schema/99.99.99/objects/record)
			pathParts := strings.Split(r.URL.Path, "/")
			if len(pathParts) >= 3 {
				version := pathParts[2]
				// Check if version is invalid
				if version == invalidVersion {
					w.WriteHeader(http.StatusNotFound)

					return
				}
			}
		}

		// Validate URL format: /schema/<version>/<type>/<name>
		if strings.Contains(r.URL.Path, "/schema/") {
			pathParts := strings.Split(r.URL.Path, "/")
			if len(pathParts) < 5 {
				w.WriteHeader(http.StatusBadRequest)

				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		mockSchema := mockSchemaResponse()
		if err := json.NewEncoder(w).Encode(mockSchema); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
}

// validateSchemaKeyResult validates the result from GetSchemaKey.
func validateSchemaKeyResult(t *testing.T, result []byte, expectEmpty bool) {
	t.Helper()

	if expectEmpty {
		if len(result) > 2 { // More than just {}
			t.Errorf("GetSchemaKey() expected empty result but got data")
		}

		return
	}

	if len(result) == 0 {
		t.Errorf("GetSchemaKey() returned empty result")
	}

	var jsonMap map[string]any
	if err := json.Unmarshal(result, &jsonMap); err != nil {
		t.Errorf("GetSchemaKey() returned invalid JSON: %v", err)
	}
}

func TestGetSchemaKey(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		defsKey     string
		expectError bool
		expectEmpty bool
	}{
		{
			name:        "valid skills key",
			version:     "0.7.0",
			defsKey:     "skills",
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "valid domains key",
			version:     "0.7.0",
			defsKey:     "domains",
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "valid objects key",
			version:     "0.7.0",
			defsKey:     "objects",
			expectError: false,
			expectEmpty: false,
		},
		{
			name:        "invalid key",
			version:     "0.7.0",
			defsKey:     "nonexistent",
			expectError: true,
		},
		{
			name:        "invalid version",
			version:     invalidVersion,
			defsKey:     "skills",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServerWithVersionCheck(t, tt.expectError && tt.version == invalidVersion)
			defer server.Close()

			schema, err := New(server.URL)
			if err != nil {
				t.Fatalf("Failed to create schema: %v", err)
			}

			result, err := schema.GetSchemaKey(context.Background(), tt.defsKey, WithVersion(tt.version))
			if tt.expectError {
				if err == nil {
					t.Errorf("GetSchemaKey() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("GetSchemaKey() unexpected error: %v", err)

				return
			}

			validateSchemaKeyResult(t, result, tt.expectEmpty)
		})
	}
}

// validateSkillsResult validates the result from GetSchemaSkills.
func validateSkillsResult(t *testing.T, skills []byte) {
	t.Helper()

	if len(skills) == 0 {
		t.Errorf("GetSchemaSkills() returned empty skills")
	}

	var skillsMap map[string]any
	if err := json.Unmarshal(skills, &skillsMap); err != nil {
		t.Errorf("GetSchemaSkills() returned invalid JSON: %v", err)
	}

	if len(skillsMap) == 0 {
		t.Errorf("GetSchemaSkills() returned empty skills map")
	}
}

//nolint:dupl // Test functions intentionally follow similar patterns
func TestGetSchemaSkills(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "valid version 0.7.0",
			version:     "0.7.0",
			expectError: false,
		},
		{
			name:        "valid version 0.8.0",
			version:     "0.8.0",
			expectError: false,
		},
		{
			name:        "invalid version",
			version:     invalidVersion,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServerWithVersionCheck(t, tt.expectError && tt.version == invalidVersion)
			defer server.Close()

			schema, err := New(server.URL)
			if err != nil {
				t.Fatalf("Failed to create schema: %v", err)
			}

			skills, err := schema.GetSchemaSkills(context.Background(), WithVersion(tt.version))
			if tt.expectError {
				if err == nil {
					t.Errorf("GetSchemaSkills() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("GetSchemaSkills() unexpected error: %v", err)
			}

			validateSkillsResult(t, skills)
		})
	}
}

// validateDomainsResult validates the result from GetSchemaDomains.
func validateDomainsResult(t *testing.T, domains []byte, version string) {
	t.Helper()

	var domainsMap map[string]any
	if err := json.Unmarshal(domains, &domainsMap); err != nil {
		t.Errorf("GetSchemaDomains() returned invalid JSON: %v", err)
	}

	if version == testSchemaVersion {
		if len(domainsMap) == 0 {
			t.Errorf("GetSchemaDomains() returned empty domains map for version %s", version)
		}

		if _, ok := domainsMap["lean_manufacturing"]; !ok {
			t.Logf("Warning: Expected domain 'lean_manufacturing' not found in version %s", version)
		}
	}
}

//nolint:dupl // Test functions intentionally follow similar patterns
func TestGetSchemaDomains(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
	}{
		{
			name:        "valid version 0.7.0",
			version:     "0.7.0",
			expectError: false,
		},
		{
			name:        "valid version 0.8.0",
			version:     "0.8.0",
			expectError: false,
		},
		{
			name:        "invalid version",
			version:     invalidVersion,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockServerWithVersionCheck(t, tt.expectError && tt.version == invalidVersion)
			defer server.Close()

			schema, err := New(server.URL)
			if err != nil {
				t.Fatalf("Failed to create schema: %v", err)
			}

			domains, err := schema.GetSchemaDomains(context.Background(), WithVersion(tt.version))
			if tt.expectError {
				if err == nil {
					t.Errorf("GetSchemaDomains() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("GetSchemaDomains() unexpected error: %v", err)
			}

			validateDomainsResult(t, domains, tt.version)
		})
	}
}

// createMockServerWithVersionsEndpoint creates a mock server that handles both versions and schema endpoints.
func createMockServerWithVersionsEndpoint(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle versions endpoint
		if r.URL.Path == apiVersionsPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			versionsResp := VersionsResponse{
				Default: struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					Version: "0.8.0",
					URL:     r.Host + "/api/0.8.0",
				},
				Versions: []struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					{Version: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{Version: "0.8.0", URL: r.Host + "/0.8.0/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		mockSchema := mockSchemaResponse()
		if err := json.NewEncoder(w).Encode(mockSchema); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
}

// createVersionsMockServer creates a mock server for versions endpoint.
func createVersionsMockServer(t *testing.T, mockResponse VersionsResponse) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != apiVersionsPath {
			t.Errorf("Expected path %s, got %s", apiVersionsPath, r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if err := json.NewEncoder(w).Encode(mockResponse); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
}

// validateVersionsResult validates the versions returned from GetAvailableSchemaVersions.
func validateVersionsResult(t *testing.T, versions []string) {
	t.Helper()

	if len(versions) == 0 {
		t.Error("GetAvailableSchemaVersions() returned no versions")
	}

	expectedVersions := map[string]bool{
		"0.7.0": true,
		"0.8.0": true,
	}

	foundVersions := make(map[string]bool)
	for _, v := range versions {
		foundVersions[v] = true
	}

	for expected := range expectedVersions {
		if !foundVersions[expected] {
			t.Errorf("Expected version %s not found in available versions", expected)
		}
	}
}

func TestGetAvailableSchemaVersions(t *testing.T) {
	mockResponse := VersionsResponse{
		Default: struct {
			Version string `json:"version"`
			URL     string `json:"url"`
		}{
			Version: "0.8.0",
			URL:     "http://schema.oasf.outshift.com:8000/api/0.8.0",
		},
		Versions: []struct {
			Version string `json:"version"`
			URL     string `json:"url"`
		}{
			{Version: "0.7.0", URL: "http://schema.oasf.outshift.com:8000/0.7.0/api"},
			{Version: "0.8.0", URL: "http://schema.oasf.outshift.com:8000/0.8.0/api"},
		},
	}

	t.Run("valid versions response", func(t *testing.T) {
		server := createVersionsMockServer(t, mockResponse)
		defer server.Close()

		schema, err := New(server.URL)
		if err != nil {
			t.Fatalf("Failed to create schema: %v", err)
		}

		versions, err := schema.GetAvailableSchemaVersions(context.Background())
		if err != nil {
			t.Errorf("GetAvailableSchemaVersions() unexpected error: %v", err)
		}

		validateVersionsResult(t, versions)
	})

	t.Run("empty URL", func(t *testing.T) {
		_, err := New("")
		if err == nil {
			t.Errorf("New() expected error but got none")
		}
	})
}

// Helper function to compare schema section counts between dedicated getter and full schema.
func compareSchemaSection(t *testing.T, schema *Schema, version string, sectionName string, getSection func(context.Context, ...SchemaOption) ([]byte, error)) {
	t.Helper()

	fullSchema, err := schema.GetRecordSchemaContent(context.Background(), WithVersion(version))
	if err != nil {
		t.Fatalf("Failed to get full schema: %v", err)
	}

	var fullSchemaMap map[string]any
	if err := json.Unmarshal(fullSchema, &fullSchemaMap); err != nil {
		t.Fatalf("Failed to parse full schema: %v", err)
	}

	sectionData, err := getSection(context.Background(), WithVersion(version))
	if err != nil {
		t.Fatalf("Failed to get %s: %v", sectionName, err)
	}

	var sectionMap map[string]any
	if err := json.Unmarshal(sectionData, &sectionMap); err != nil {
		t.Fatalf("Failed to parse %s: %v", sectionName, err)
	}

	// Extract section from full schema
	defs, ok := fullSchemaMap["$defs"].(map[string]any)
	if !ok {
		t.Fatalf("Expected $defs to be map[string]any")
	}

	fullSchemaSection, ok := defs[sectionName].(map[string]any)
	if !ok {
		t.Fatalf("Expected %s to be map[string]any", sectionName)
	}

	// Compare the number of items
	if len(sectionMap) != len(fullSchemaSection) {
		t.Errorf("%s count mismatch: getter returned %d items, full schema has %d items",
			sectionName, len(sectionMap), len(fullSchemaSection))
	}
}

func TestGetSchemaSkillsVsFullSchema(t *testing.T) {
	// Create a mock HTTP server
	server := createMockServerWithVersionsEndpoint(t)
	defer server.Close()

	// Create a schema instance with the mock server URL
	schema, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// This test ensures that GetSchemaSkills returns the same skills
	// section as in the full schema
	compareSchemaSection(t, schema, testSchemaVersion, "skills", schema.GetSchemaSkills)
}

func TestGetSchemaDomainsVsFullSchema(t *testing.T) {
	// Create a mock HTTP server
	server := createMockServerWithVersionsEndpoint(t)
	defer server.Close()

	// Create a schema instance with the mock server URL
	schema, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// This test ensures that GetSchemaDomains returns the same domains
	// section as in the full schema
	compareSchemaSection(t, schema, testSchemaVersion, "domains", schema.GetSchemaDomains)
}

func TestGetSchemaModules(t *testing.T) {
	// Create a mock HTTP server
	server := createMockServerWithVersionsEndpoint(t)
	defer server.Close()

	// Create a schema instance with the mock server URL
	schema, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	// Note: modules may not exist in all schema versions
	_, err = schema.GetSchemaModules(context.Background(), WithVersion("0.7.0"))
	// We don't assert on error since modules might not exist
	// This test mainly ensures the function doesn't panic
	_ = err
}

func TestDefaultVersion(t *testing.T) {
	defaultVersion := "0.8.0"

	var requestedVersions []string

	// Create a mock HTTP server that tracks which version was requested
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle versions endpoint
		if r.URL.Path == apiVersionsPath {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			versionsResp := VersionsResponse{
				Default: struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					Version: defaultVersion,
					URL:     r.Host + "/api/" + defaultVersion,
				},
				Versions: []struct {
					Version string `json:"version"`
					URL     string `json:"url"`
				}{
					{Version: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{Version: defaultVersion, URL: r.Host + "/" + defaultVersion + "/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		// Track which schema version was requested
		if strings.Contains(r.URL.Path, "/schema/") {
			pathParts := strings.Split(r.URL.Path, "/")
			if len(pathParts) >= 3 {
				requestedVersions = append(requestedVersions, pathParts[2])
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		mockSchema := mockSchemaResponse()
		if err := json.NewEncoder(w).Encode(mockSchema); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	schema, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	testCases := []struct {
		name     string
		testFunc func() error
		minCount int
	}{
		{
			name: "GetRecordSchemaContent",
			testFunc: func() error {
				_, err := schema.GetRecordSchemaContent(context.Background())

				return err
			},
			minCount: 1,
		},
		{
			name: "GetSchemaSkills",
			testFunc: func() error {
				_, err := schema.GetSchemaSkills(context.Background())

				return err
			},
			minCount: 2,
		},
		{
			name: "GetSchemaDomains",
			testFunc: func() error {
				_, err := schema.GetSchemaDomains(context.Background())

				return err
			},
			minCount: 3,
		},
		{
			name: "GetSchemaKey",
			testFunc: func() error {
				_, err := schema.GetSchemaKey(context.Background(), "skills")

				return err
			},
			minCount: 4,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.testFunc()
			if err != nil {
				t.Errorf("%s() with default version unexpected error: %v", tc.name, err)
			}

			if len(requestedVersions) < tc.minCount || requestedVersions[len(requestedVersions)-1] != defaultVersion {
				t.Errorf("Expected default version %s, got %v", defaultVersion, requestedVersions)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		schemaURL   string
		expectError bool
	}{
		{
			name:        "valid URL",
			schemaURL:   "https://schema.oasf.outshift.com",
			expectError: false,
		},
		{
			name:        "empty URL",
			schemaURL:   "",
			expectError: true,
		},
		{
			name:        "URL without protocol",
			schemaURL:   "schema.oasf.outshift.com",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := New(tt.schemaURL)
			if tt.expectError {
				if err == nil {
					t.Errorf("New() expected error but got none")
				}

				if schema != nil {
					t.Errorf("New() expected nil schema but got %v", schema)
				}

				return
			}

			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
			}

			if schema == nil {
				t.Errorf("New() expected schema but got nil")
			}
		})
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
