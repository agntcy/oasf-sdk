// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

const defaultHTTPTimeoutSeconds = 30

const schemaVersion031 = "0.3.1"

// Supported schema versions.
var supportedVersions = []string{"0.3.1", "0.7.0", "0.8.0"}

// Schema provides access to OASF schema definitions via API.
type Schema struct {
	schemaURL  string
	httpClient *http.Client
}

// New creates a new Schema instance with the given schema base URL.
// The base URL should point to the OASF schema API endpoint (e.g., https://schema.oasf.outshift.com).
func New(schemaURL string) (*Schema, error) {
	if schemaURL == "" {
		return nil, errors.New("schema URL is required")
	}

	return &Schema{
		schemaURL: schemaURL,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeoutSeconds * time.Second,
		},
	}, nil
}

// GetAvailableSchemaVersions returns a list of all supported schema versions.
func GetAvailableSchemaVersions() []string {
	return supportedVersions
}

// constructRecordSchemaURL builds the full schema URL from a base URL and schema version.
func (s *Schema) constructRecordSchemaURL(schemaVersion string) (string, error) {
	// Check if version is supported (check both original and normalized)
	if !slices.Contains(supportedVersions, schemaVersion) {
		return "", fmt.Errorf("unsupported schema version: %s", schemaVersion)
	}

	// Normalize the base URL (remove trailing slash if present)
	normalizedURL := strings.TrimSuffix(s.schemaURL, "/")

	// Add protocol if missing (default to http:// for localhost or IP addresses)
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "http://" + normalizedURL
	}

	// Determine the object type based on schema version
	// Version 0.3.1 uses "agent", while later versions use "record"
	objectType := "record"
	if schemaVersion == schemaVersion031 {
		objectType = "agent"
	}

	// Construct the full schema URL
	return fmt.Sprintf("%s/schema/%s/objects/%s", normalizedURL, schemaVersion, objectType), nil
}

// GetRecordSchemaContent returns the raw JSON schema content for a given version.
// Returns an error if the version is not found or if there's an issue fetching the schema.
func (s *Schema) GetRecordSchemaContent(ctx context.Context, version string) ([]byte, error) {
	schemaURL, err := s.constructRecordSchemaURL(version)
	if err != nil {
		return nil, err
	}

	// Create GET request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, schemaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request to %s: %w", schemaURL, err)
	}

	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request to %s: %w", schemaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to fetch schema from URL %s: HTTP %d, body: %s", schemaURL, resp.StatusCode, string(body))
	}

	// Read response body
	schemaData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema response from URL %s: %w", schemaURL, err)
	}

	return schemaData, nil
}

// GetSchemaKey is a generic function to extract any $defs category from a schema.
// For example, extracting skills, domains, modules, or any other $defs key.
// Returns the category definitions as JSON bytes, or an error if not found.
func (s *Schema) GetSchemaKey(ctx context.Context, version, defsKey string) ([]byte, error) {
	schemaData, err := s.GetRecordSchemaContent(ctx, version)
	if err != nil {
		return nil, err
	}

	var schemaMap map[string]any
	if err := json.Unmarshal(schemaData, &schemaMap); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}

	defs, ok := schemaMap["$defs"].(map[string]any)
	if !ok {
		return nil, errors.New("schema does not contain $defs section")
	}

	category, ok := defs[defsKey].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("schema does not contain '%s' definitions in $defs", defsKey)
	}

	categoryJSON, err := json.Marshal(category)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %w", defsKey, err)
	}

	return categoryJSON, nil
}

// GetSchemaSkills is a convenience function to extract skills from a schema.
// Returns the skills as JSON bytes, or an error if the version is not found or parsing fails.
func (s *Schema) GetSchemaSkills(ctx context.Context, version string) ([]byte, error) {
	return s.GetSchemaKey(ctx, version, "skills")
}

// GetSchemaDomains is a convenience function to extract domains from a schema.
// Returns the domains as JSON bytes, or an error if the version is not found or parsing fails.
func (s *Schema) GetSchemaDomains(ctx context.Context, version string) ([]byte, error) {
	return s.GetSchemaKey(ctx, version, "domains")
}

// GetSchemaModules is a convenience function to extract modules from a schema.
// Returns the modules as JSON bytes, or an error if the version is not found or parsing fails.
func (s *Schema) GetSchemaModules(ctx context.Context, version string) ([]byte, error) {
	return s.GetSchemaKey(ctx, version, "modules")
}
