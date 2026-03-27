// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func mockCategoriesResponse() SchemaCategories {
	return SchemaCategories{
		"core": {
			ID:          1,
			Name:        "core",
			Description: "Module set for core functionalities and features.",
			Category:    true,
			Caption:     "Core",
			Classes: map[string]SchemaCategoryNode{
				"language_model": {
					ID:          103,
					Name:        "core/language_model",
					Description: "Modules for basic Language Model functionality.",
					Caption:     "Language Model",
					Classes: map[string]SchemaCategoryNode{
						"prompt": {
							ID:          10301,
							Name:        "core/language_model/prompt",
							Description: "Describes common Language Model interaction prompts to use the agent.",
							Caption:     "Language Model Prompt",
						},
					},
				},
			},
		},
		"integration": {
			ID:          2,
			Name:        "integration",
			Description: "Module set for integrating with external systems and services.",
			Category:    true,
			Caption:     "Integration",
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
				Default: VersionInfo{
					SchemaVersion: "0.8.0",
					URL:           r.Host + "/api/0.8.0",
				},
				Versions: []VersionInfo{
					{SchemaVersion: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{SchemaVersion: "0.8.0", URL: r.Host + "/0.8.0/api"},
					{SchemaVersion: "1.0.0", URL: r.Host + "/1.0.0/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		expectedPath := "/schema/" + version + "/objects/record"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
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
			name:        "valid version 1.0.0",
			version:     "1.0.0",
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
				Default: VersionInfo{
					SchemaVersion: "0.8.0",
					URL:           r.Host + "/api/0.8.0",
				},
				Versions: []VersionInfo{
					{SchemaVersion: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{SchemaVersion: "0.8.0", URL: r.Host + "/0.8.0/api"},
					{SchemaVersion: "1.0.0", URL: r.Host + "/1.0.0/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		// For invalid-version tests, all schema fetch candidates should fail.
		if checkVersion {
			w.WriteHeader(http.StatusNotFound)

			return
		}

		// Handle category endpoints.
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) == 4 && pathParts[1] == "api" &&
			(pathParts[3] == "module_categories" || pathParts[3] == "skill_categories" || pathParts[3] == "domain_categories") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(mockCategoriesResponse()); err != nil {
				t.Errorf("Failed to encode categories response: %v", err)
			}

			return
		}

		// Validate schema request shape.
		pathParts = strings.Split(r.URL.Path, "/")
		if len(pathParts) < 5 || pathParts[1] != "schema" {
			t.Errorf("Expected path /schema/<version>/<type>/<name>, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		mockSchema := mockSchemaResponse()
		if err := json.NewEncoder(w).Encode(mockSchema); err != nil {
			t.Errorf("Failed to encode mock response: %v", err)
		}
	}))
}

// validateSkillsResult validates the result from GetSchemaSkills.
func validateSkillsResult(t *testing.T, skills SchemaCategories) {
	t.Helper()

	if len(skills) == 0 {
		t.Errorf("GetSchemaSkills() returned empty skills map")
	}

	core, ok := skills["core"]
	if !ok {
		t.Fatalf("GetSchemaSkills() missing top-level 'core' category")
	}

	if _, ok := core.Classes["language_model"]; !ok {
		t.Errorf("GetSchemaSkills() missing nested 'language_model' class")
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
			name:        "valid version 1.0.0",
			version:     "1.0.0",
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
func validateDomainsResult(t *testing.T, domains SchemaCategories, version string) {
	t.Helper()

	if version == testSchemaVersion {
		if len(domains) == 0 {
			t.Errorf("GetSchemaDomains() returned empty domains map for version %s", version)
		}

		if _, ok := domains["core"]; !ok {
			t.Logf("Warning: Expected domain category 'core' not found in version %s", version)
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
			name:        "valid version 1.0.0",
			version:     "1.0.0",
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
				Default: VersionInfo{
					SchemaVersion: "0.8.0",
					URL:           r.Host + "/api/0.8.0",
				},
				Versions: []VersionInfo{
					{SchemaVersion: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{SchemaVersion: "0.8.0", URL: r.Host + "/0.8.0/api"},
					{SchemaVersion: "1.0.0", URL: r.Host + "/1.0.0/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) == 4 && pathParts[1] == "api" &&
			(pathParts[3] == "module_categories" || pathParts[3] == "skill_categories" || pathParts[3] == "domain_categories") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(mockCategoriesResponse()); err != nil {
				t.Errorf("Failed to encode categories response: %v", err)
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
		"1.0.0": true,
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
		Default: VersionInfo{
			SchemaVersion: "0.8.0",
			URL:           "http://schema.oasf.outshift.com:8000/api/0.8.0",
		},
		Versions: []VersionInfo{
			{SchemaVersion: "0.7.0", URL: "http://schema.oasf.outshift.com:8000/0.7.0/api"},
			{SchemaVersion: "0.8.0", URL: "http://schema.oasf.outshift.com:8000/0.8.0/api"},
			{SchemaVersion: "1.0.0", URL: "http://schema.oasf.outshift.com:8000/1.0.0/api"},
		},
	}
	legacyMockResponse := VersionsResponse{
		Default: VersionInfo{
			Version: "0.8.0",
			URL:     "http://schema.oasf.outshift.com:8000/api/0.8.0",
		},
		Versions: []VersionInfo{
			{Version: "0.7.0", URL: "http://schema.oasf.outshift.com:8000/0.7.0/api"},
			{Version: "0.8.0", URL: "http://schema.oasf.outshift.com:8000/0.8.0/api"},
			{Version: "1.0.0", URL: "http://schema.oasf.outshift.com:8000/1.0.0/api"},
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
	t.Run("legacy versions response", func(t *testing.T) {
		server := createVersionsMockServer(t, legacyMockResponse)
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

func TestGetDefaultSchemaVersionLegacyFormat(t *testing.T) {
	server := createVersionsMockServer(t, VersionsResponse{
		Default: VersionInfo{Version: "0.8.0"},
		Versions: []VersionInfo{
			{Version: "0.7.0"},
			{Version: "0.8.0"},
		},
	})
	defer server.Close()

	schema, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	defaultVersion, err := schema.GetDefaultSchemaVersion(context.Background())
	if err != nil {
		t.Fatalf("GetDefaultSchemaVersion() unexpected error: %v", err)
	}

	if defaultVersion != "0.8.0" {
		t.Fatalf("Expected default version 0.8.0, got %s", defaultVersion)
	}
}

// validateCategoryTree checks that categories include nested classes.
func validateCategoryTree(t *testing.T, categories SchemaCategories) {
	t.Helper()

	if len(categories) == 0 {
		t.Fatalf("Expected non-empty category response")
	}

	core, ok := categories["core"]
	if !ok {
		t.Fatalf("Expected top-level 'core' category")
	}

	languageModel, ok := core.Classes["language_model"]
	if !ok {
		t.Fatalf("Expected nested 'language_model' class")
	}

	if _, ok := languageModel.Classes["prompt"]; !ok {
		t.Fatalf("Expected nested 'prompt' class")
	}
}

func TestGetSchemaSkillsNested(t *testing.T) {
	// Create a mock HTTP server
	server := createMockServerWithVersionsEndpoint(t)
	defer server.Close()

	// Create a schema instance with the mock server URL
	schema, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	skills, err := schema.GetSchemaSkills(context.Background(), WithVersion(testSchemaVersion))
	if err != nil {
		t.Fatalf("Failed to get skills: %v", err)
	}

	validateCategoryTree(t, skills)
}

func TestGetSchemaDomainsNested(t *testing.T) {
	// Create a mock HTTP server
	server := createMockServerWithVersionsEndpoint(t)
	defer server.Close()

	// Create a schema instance with the mock server URL
	schema, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	domains, err := schema.GetSchemaDomains(context.Background(), WithVersion(testSchemaVersion))
	if err != nil {
		t.Fatalf("Failed to get domains: %v", err)
	}

	validateCategoryTree(t, domains)
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
				Default: VersionInfo{
					SchemaVersion: defaultVersion,
					URL:           r.Host + "/api/" + defaultVersion,
				},
				Versions: []VersionInfo{
					{SchemaVersion: "0.7.0", URL: r.Host + "/0.7.0/api"},
					{SchemaVersion: defaultVersion, URL: r.Host + "/" + defaultVersion + "/api"},
				},
			}

			if err := json.NewEncoder(w).Encode(versionsResp); err != nil {
				t.Errorf("Failed to encode versions response: %v", err)
			}

			return
		}

		// Track which schema version was requested on /schema/<version>/<type>/<name>.
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) >= 5 && pathParts[1] == "schema" {
			requestedVersions = append(requestedVersions, pathParts[2])
		}

		// Track which schema version was requested on /api/<version>/*_categories.
		if len(pathParts) == 4 && pathParts[1] == "api" &&
			(pathParts[3] == "module_categories" || pathParts[3] == "skill_categories" || pathParts[3] == "domain_categories") {
			requestedVersions = append(requestedVersions, pathParts[2])
		}

		// Return category responses for category endpoints.
		if len(pathParts) == 4 && pathParts[1] == "api" &&
			(pathParts[3] == "module_categories" || pathParts[3] == "skill_categories" || pathParts[3] == "domain_categories") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(mockCategoriesResponse()); err != nil {
				t.Errorf("Failed to encode categories response: %v", err)
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

func TestVersionsAreCached(t *testing.T) {
	var versionsCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == apiVersionsPath {
			atomic.AddInt32(&versionsCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(VersionsResponse{
				Default: VersionInfo{SchemaVersion: "0.8.0"},
				Versions: []VersionInfo{
					{SchemaVersion: "0.7.0"},
					{SchemaVersion: "0.8.0"},
					{SchemaVersion: "1.0.0"},
				},
			})

			return
		}

		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) == 4 && pathParts[1] == "api" &&
			(pathParts[3] == "module_categories" || pathParts[3] == "skill_categories" || pathParts[3] == "domain_categories") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	s, err := New(server.URL)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	if err := s.Cache(context.Background()); err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	if _, err := s.GetAvailableSchemaVersions(context.Background()); err != nil {
		t.Fatalf("GetAvailableSchemaVersions from cache failed: %v", err)
	}
	if _, err := s.GetDefaultSchemaVersion(context.Background()); err != nil {
		t.Fatalf("GetDefaultSchemaVersion from cache failed: %v", err)
	}
	if _, err := s.GetAvailableSchemaVersions(context.Background()); err != nil {
		t.Fatalf("GetAvailableSchemaVersions second cached call failed: %v", err)
	}

	if got := atomic.LoadInt32(&versionsCalls); got != 1 {
		t.Fatalf("expected exactly one /api/versions call, got %d", got)
	}
}

func TestSchemaCategoriesCachedByVersion(t *testing.T) {
	var categoryCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case apiVersionsPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(VersionsResponse{
				Default: VersionInfo{SchemaVersion: "0.8.0"},
				Versions: []VersionInfo{
					{SchemaVersion: "0.8.0"},
				},
			})
			return
		case "/api/0.8.0/skill_categories":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		case "/api/0.8.0/domain_categories":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		case "/api/0.8.0/module_categories":
			atomic.AddInt32(&categoryCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	s, err := New(server.URL)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	if err := s.Cache(context.Background()); err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	if _, err := s.GetSchemaModules(context.Background(), WithVersion("0.8.0")); err != nil {
		t.Fatalf("GetSchemaModules first cached call failed: %v", err)
	}
	if _, err := s.GetSchemaModules(context.Background(), WithVersion("0.8.0")); err != nil {
		t.Fatalf("GetSchemaModules second cached call failed: %v", err)
	}

	if got := atomic.LoadInt32(&categoryCalls); got != 1 {
		t.Fatalf("expected exactly one categories fetch, got %d", got)
	}
}

func TestClearCache(t *testing.T) {
	var versionsCalls int32
	var skillCalls int32
	var domainCalls int32
	var moduleCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case apiVersionsPath:
			atomic.AddInt32(&versionsCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(VersionsResponse{
				Default: VersionInfo{SchemaVersion: "0.8.0"},
				Versions: []VersionInfo{
					{SchemaVersion: "0.8.0"},
				},
			})
			return
		case "/api/0.8.0/skill_categories":
			atomic.AddInt32(&skillCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		case "/api/0.8.0/domain_categories":
			atomic.AddInt32(&domainCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		case "/api/0.8.0/module_categories":
			atomic.AddInt32(&moduleCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	s, err := New(server.URL)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	if err := s.Cache(context.Background()); err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	if _, err := s.GetSchemaSkills(context.Background(), WithVersion("0.8.0")); err != nil {
		t.Fatalf("GetSchemaSkills from cache failed: %v", err)
	}

	s.ClearCache()

	if _, err := s.GetSchemaSkills(context.Background(), WithVersion("0.8.0")); err != nil {
		t.Fatalf("GetSchemaSkills after ClearCache failed: %v", err)
	}

	if got := atomic.LoadInt32(&versionsCalls); got != 2 {
		t.Fatalf("expected two /api/versions calls, got %d", got)
	}
	if got := atomic.LoadInt32(&skillCalls); got != 2 {
		t.Fatalf("expected two skill fetches, got %d", got)
	}
	if got := atomic.LoadInt32(&domainCalls); got != 1 {
		t.Fatalf("expected one domain fetch from initial Cache, got %d", got)
	}
	if got := atomic.LoadInt32(&moduleCalls); got != 1 {
		t.Fatalf("expected one module fetch from initial Cache, got %d", got)
	}
}

func TestReloadCache(t *testing.T) {
	var skillCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case apiVersionsPath:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(VersionsResponse{
				Default: VersionInfo{SchemaVersion: "0.8.0"},
				Versions: []VersionInfo{
					{SchemaVersion: "0.8.0"},
				},
			})
			return
		case "/api/0.8.0/skill_categories":
			call := atomic.AddInt32(&skillCalls, 1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			if call == 1 {
				_ = json.NewEncoder(w).Encode(SchemaCategories{
					"skills": {ID: 1, Name: "skills-v1"},
				})
			} else {
				_ = json.NewEncoder(w).Encode(SchemaCategories{
					"skills": {ID: 1, Name: "skills-v2"},
				})
			}
			return
		case "/api/0.8.0/domain_categories", "/api/0.8.0/module_categories":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())
			return
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	defer server.Close()

	s, err := New(server.URL)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	if err := s.Cache(context.Background()); err != nil {
		t.Fatalf("Cache failed: %v", err)
	}
	first, err := s.GetSchemaSkills(context.Background(), WithVersion("0.8.0"))
	if err != nil {
		t.Fatalf("GetSchemaSkills first cached read failed: %v", err)
	}
	if first["skills"].Name != "skills-v1" {
		t.Fatalf("expected first cached value skills-v1, got %q", first["skills"].Name)
	}

	if err := s.ReloadCache(context.Background()); err != nil {
		t.Fatalf("ReloadCache failed: %v", err)
	}
	second, err := s.GetSchemaSkills(context.Background(), WithVersion("0.8.0"))
	if err != nil {
		t.Fatalf("GetSchemaSkills second cached read failed: %v", err)
	}
	if second["skills"].Name != "skills-v2" {
		t.Fatalf("expected reloaded cached value skills-v2, got %q", second["skills"].Name)
	}

	if got := atomic.LoadInt32(&skillCalls); got != 2 {
		t.Fatalf("expected two skill endpoint calls (cache + reload), got %d", got)
	}
}

func TestGetSchemaModulesRejectsUnsupportedVersionBeforeCategoryFetch(t *testing.T) {
	categoryEndpointHit := false
	versionsEndpointCalls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == apiVersionsPath {
			versionsEndpointCalls++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(VersionsResponse{
				Default: VersionInfo{SchemaVersion: "1.0.0"},
				Versions: []VersionInfo{
					{SchemaVersion: "0.8.0"},
					{SchemaVersion: "1.0.0"},
				},
			})

			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/") && strings.HasSuffix(r.URL.Path, "/module_categories") {
			categoryEndpointHit = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(mockCategoriesResponse())

			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	s, err := New(server.URL)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	_, err = s.GetSchemaModules(context.Background(), WithVersion("1.1.0"))
	if err == nil {
		t.Fatalf("Expected unsupported-version error, got nil")
	}
	if !strings.Contains(err.Error(), `schema version "1.1.0" is not supported`) {
		t.Fatalf("Unexpected error: %v", err)
	}
	if categoryEndpointHit {
		t.Fatalf("Category endpoint should not be called for unsupported version")
	}
	if versionsEndpointCalls != 1 {
		t.Fatalf("Expected one versions fetch, got %d", versionsEndpointCalls)
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
