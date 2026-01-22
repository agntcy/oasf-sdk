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
	"strings"
	"time"
)

const defaultHTTPTimeoutSeconds = 30

const schemaVersion031 = "0.3.1"

// VersionsResponse represents the response from the api/versions endpoint.
type VersionsResponse struct {
	Default struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	} `json:"default"`
	Versions []struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	} `json:"versions"`
}

// SchemaOption is a function that configures schema options.
type SchemaOption func(*schemaOptions)

// schemaOptions holds the options for schema operations.
type schemaOptions struct {
	version string
}

// WithVersion sets the schema version to use.
func WithVersion(version string) SchemaOption {
	return func(opts *schemaOptions) {
		opts.version = version
	}
}

// Schema provides access to OASF schema definitions via API.
type Schema struct {
	schemaURL      string // Normalized schema URL
	httpClient     *http.Client
	defaultVersion string // Cached default version
}

// normalizeURL normalizes a schema URL by removing trailing slashes and adding protocol if missing.
func normalizeURL(schemaURL string) string {
	// Normalize the base URL (remove trailing slash if present)
	normalizedURL := strings.TrimSuffix(schemaURL, "/")

	// Add protocol if missing (default to http:// for localhost or IP addresses)
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "http://" + normalizedURL
	}

	return normalizedURL
}

// New creates a new Schema instance with the given schema base URL.
// The base URL should point to the OASF schema API endpoint (e.g., https://schema.oasf.outshift.com).
// The URL will be normalized (trailing slashes removed, protocol added if missing).
func New(schemaURL string) (*Schema, error) {
	if schemaURL == "" {
		return nil, errors.New("schema URL is required")
	}

	return &Schema{
		schemaURL: normalizeURL(schemaURL),
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeoutSeconds * time.Second,
		},
	}, nil
}

const apiVersionsPath = "/api/versions"

// getVersionsResponse fetches the versions response from the server.
func (s *Schema) getVersionsResponse(ctx context.Context) (*VersionsResponse, error) {
	// Construct the versions endpoint URL
	versionsURL := s.schemaURL + apiVersionsPath

	// Create GET request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request to %s: %w", versionsURL, err)
	}

	req.Header.Set("Accept", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request to %s: %w", versionsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to fetch versions from URL %s: HTTP %d, body: %s", versionsURL, resp.StatusCode, string(body))
	}

	// Read and parse response
	var versionsResp VersionsResponse

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&versionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode versions response from URL %s: %w", versionsURL, err)
	}

	return &versionsResp, nil
}

// GetDefaultVersion returns the default schema version, caching it after first fetch.
// The default version is fetched from the server's api/versions endpoint.
func (s *Schema) GetDefaultVersion(ctx context.Context) (string, error) {
	if s.defaultVersion != "" {
		return s.defaultVersion, nil
	}

	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return "", err
	}

	s.defaultVersion = versionsResp.Default.Version

	return s.defaultVersion, nil
}

// GetAvailableSchemaVersions returns a list of all supported schema versions from the OASF server.
// It fetches the versions from the api/versions endpoint.
func (s *Schema) GetAvailableSchemaVersions(ctx context.Context) ([]string, error) {
	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return nil, err
	}

	// Extract version strings from the response
	versions := make([]string, 0, len(versionsResp.Versions))
	for _, v := range versionsResp.Versions {
		versions = append(versions, v.Version)
	}

	return versions, nil
}

// constructRecordSchemaURL builds the full schema URL from a base URL and schema version.
// Note: We don't validate version here anymore since versions are fetched dynamically.
// The API will return an error if the version is not supported.
func (s *Schema) constructRecordSchemaURL(schemaVersion string) string {
	// Determine the object type based on schema version
	// Version 0.3.1 uses "agent", while later versions use "record"
	objectType := "record"
	if schemaVersion == schemaVersion031 {
		objectType = "agent"
	}

	// Construct the full schema URL (schemaURL is already normalized)
	return fmt.Sprintf("%s/schema/%s/objects/%s", s.schemaURL, schemaVersion, objectType)
}

// GetRecordSchemaContent returns the raw JSON schema content for a given version.
// If no version is provided via options, the default version from the server is used.
// Returns an error if the version is not found or if there's an issue fetching the schema.
func (s *Schema) GetRecordSchemaContent(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Use provided version or fetch default
	version := options.version
	if version == "" {
		var err error

		version, err = s.GetDefaultVersion(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default version: %w", err)
		}
	}

	schemaURL := s.constructRecordSchemaURL(version)

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
// If no version is provided via options, the default version from the server is used.
// Returns the category definitions as JSON bytes, or an error if not found.
func (s *Schema) GetSchemaKey(ctx context.Context, defsKey string, opts ...SchemaOption) ([]byte, error) {
	schemaData, err := s.GetRecordSchemaContent(ctx, opts...)
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
// If no version is provided via options, the default version from the server is used.
// Returns the skills as JSON bytes, or an error if the version is not found or parsing fails.
func (s *Schema) GetSchemaSkills(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	return s.GetSchemaKey(ctx, "skills", opts...)
}

// GetSchemaDomains is a convenience function to extract domains from a schema.
// If no version is provided via options, the default version from the server is used.
// Returns the domains as JSON bytes, or an error if the version is not found or parsing fails.
func (s *Schema) GetSchemaDomains(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	return s.GetSchemaKey(ctx, "domains", opts...)
}

// GetSchemaModules is a convenience function to extract modules from a schema.
// If no version is provided via options, the default version from the server is used.
// Returns the modules as JSON bytes, or an error if the version is not found or parsing fails.
func (s *Schema) GetSchemaModules(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	return s.GetSchemaKey(ctx, "modules", opts...)
}
