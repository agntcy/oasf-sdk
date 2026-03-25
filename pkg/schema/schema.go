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

// VersionsResponse represents the response from the api/versions endpoint.
type VersionsResponse struct {
	Default  VersionInfo   `json:"default"`
	Versions []VersionInfo `json:"versions"`
}

// VersionInfo represents schema server version metadata.
// The API has evolved over time, so we support both legacy and v0.6.0 fields.
type VersionInfo struct {
	Version       string `json:"version,omitempty"`
	SchemaVersion string `json:"schema_version"`
	URL           string `json:"url,omitempty"`
	APIURL        string `json:"api_url,omitempty"`
	ServerVersion string `json:"server_version,omitempty"`
	APIVersion    string `json:"api_version,omitempty"`
}

// SchemaCategoryNode represents a nested taxonomy node returned by *_categories endpoints.
type SchemaCategoryNode struct {
	ID          int                           `json:"id"`
	Name        string                        `json:"name"`
	Description string                        `json:"description,omitempty"`
	Category    bool                          `json:"category,omitempty"`
	Caption     string                        `json:"caption,omitempty"`
	Deprecated  bool                          `json:"deprecated,omitempty"`
	Classes     map[string]SchemaCategoryNode `json:"classes,omitempty"`
}

// SchemaCategories is the top-level category map keyed by category slug.
type SchemaCategories map[string]SchemaCategoryNode

// SchemaOption is a function that configures schema options.
type SchemaOption func(*schemaOptions)

// SchemaType represents the type of schema to fetch.
type SchemaType string

const (
	// SchemaTypeObjects represents object schemas (agent, record).
	SchemaTypeObjects SchemaType = "objects"
	// SchemaTypeModules represents module schemas.
	SchemaTypeModules SchemaType = "modules"
	// SchemaTypeSkills represents skill schemas.
	SchemaTypeSkills SchemaType = "skills"
	// SchemaTypeDomains represents domain schemas.
	SchemaTypeDomains SchemaType = "domains"
)

// schemaOptions holds the options for schema operations.
type schemaOptions struct {
	schemaVersion string
}

// WithSchemaVersion sets the schema version to use.
func WithSchemaVersion(schemaVersion string) SchemaOption {
	return func(opts *schemaOptions) {
		opts.schemaVersion = schemaVersion
	}
}

// Schema provides access to OASF schema definitions via API.
type Schema struct {
	schemaURL            string // Normalized schema URL
	httpClient           *http.Client
	defaultSchemaVersion string // Cached default version
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

// GetDefaultSchemaVersion returns the default schema version, caching it after first fetch.
// The default schema version is fetched from the server's api/versions endpoint.
func (s *Schema) GetDefaultSchemaVersion(ctx context.Context) (string, error) {
	if s.defaultSchemaVersion != "" {
		return s.defaultSchemaVersion, nil
	}

	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return "", err
	}

	s.defaultSchemaVersion = schemaVersionFromVersionInfo(versionsResp.Default)
	if s.defaultSchemaVersion == "" {
		return "", errors.New("default schema version is missing from /api/versions response")
	}

	return s.defaultSchemaVersion, nil
}

// GetAvailableSchemaVersions returns a list of all supported schema versions from the OASF server.
// It fetches the versions from the api/versions endpoint.
func (s *Schema) GetAvailableSchemaVersions(ctx context.Context) ([]string, error) {
	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return nil, err
	}

	// Extract schema version strings from the response
	schemaVersions := make([]string, 0, len(versionsResp.Versions))
	for _, v := range versionsResp.Versions {
		schemaVersion := schemaVersionFromVersionInfo(v)
		if schemaVersion != "" {
			schemaVersions = append(schemaVersions, schemaVersion)
		}
	}

	return schemaVersions, nil
}

// constructSchemaURL builds the schema URL from options.
// Format: /schema/<version>/<type>/<name>.
func (s *Schema) constructSchemaURL(version string, schemaType SchemaType, name string) string {
	return fmt.Sprintf("%s/schema/%s/%s/%s", s.schemaURL, version, schemaType, name)
}

func schemaVersionFromVersionInfo(versionInfo VersionInfo) string {
	if versionInfo.SchemaVersion != "" {
		return versionInfo.SchemaVersion
	}

	return versionInfo.Version
}

// GetSchema is a generic function to fetch schema content from the OASF API.
// It constructs the URL as /schema/<version>/<type>/<name>.
// schemaType must be one of: objects, modules, skills, or domains.
// name specifies the specific schema name (e.g., "agent", "record", or specific module/skill/domain name).
// If no version is provided via options, the default version is used.
func (s *Schema) GetSchema(ctx context.Context, schemaType SchemaType, name string, opts ...SchemaOption) ([]byte, error) {
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Use provided schemaVersion or fetch default
	schemaVersion := options.schemaVersion
	if schemaVersion == "" {
		var err error

		schemaVersion, err = s.GetDefaultSchemaVersion(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default version: %w", err)
		}
	}

	schemaURL := s.constructSchemaURL(schemaVersion, schemaType, name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, schemaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request to %s: %w", schemaURL, err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request to %s: %w", schemaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to fetch schema from URL %s: HTTP %d, body: %s", schemaURL, resp.StatusCode, string(body))
	}

	schemaData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema response from URL %s: %w", schemaURL, err)
	}

	return schemaData, nil
}

// GetSchemaCategories fetches nested taxonomy categories from /api/<version>/<endpoint>.
// The endpoint must be one of: module_categories, skill_categories, domain_categories.
func (s *Schema) GetSchemaCategories(ctx context.Context, endpoint string, opts ...SchemaOption) (SchemaCategories, error) {
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	version := options.schemaVersion
	if version == "" {
		var err error

		version, err = s.GetDefaultSchemaVersion(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get default version: %w", err)
		}
	}

	categoriesURL := fmt.Sprintf("%s/api/%s/%s", s.schemaURL, version, endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, categoriesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request to %s: %w", categoriesURL, err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request to %s: %w", categoriesURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("failed to fetch categories from URL %s: HTTP %d, body: %s", categoriesURL, resp.StatusCode, string(body))
	}

	var categories SchemaCategories
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&categories); err != nil {
		return nil, fmt.Errorf("failed to decode categories response from URL %s: %w", categoriesURL, err)
	}

	return categories, nil
}

// GetRecordSchemaContent returns the raw JSON schema content for a given version.
// If no version is provided via options, the default version from the server is used.
// It fetches the "record" object schema.
// Returns an error if the version is not found or if there's an issue fetching the schema.
func (s *Schema) GetRecordSchemaContent(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	// Parse options to get version if provided
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Use the generic GetSchema function
	return s.GetSchema(ctx, SchemaTypeObjects, "record", opts...)
}

// GetSchemaSkills is a convenience function to fetch skill categories.
// If no version is provided via options, the default version from the server is used.
// Returns nested skill categories from /api/<version>/skill_categories.
func (s *Schema) GetSchemaSkills(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, "skill_categories", opts...)
}

// GetSchemaDomains is a convenience function to fetch domain categories.
// If no version is provided via options, the default version from the server is used.
// Returns nested domain categories from /api/<version>/domain_categories.
func (s *Schema) GetSchemaDomains(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, "domain_categories", opts...)
}

// GetSchemaModules is a convenience function to fetch module categories.
// If no version is provided via options, the default version from the server is used.
// Returns nested module categories from /api/<version>/module_categories.
func (s *Schema) GetSchemaModules(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, "module_categories", opts...)
}
