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
	"sync/atomic"
	"time"
)

const defaultHTTPTimeoutSeconds = 30
const apiVersionsPath = "/api/versions"

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

type schemaCache struct {
	defaultSchemaVersion    string
	availableSchemaVersions []string
	schemaSkills            map[string]SchemaCategories
	schemaDomains           map[string]SchemaCategories
	schemaModules           map[string]SchemaCategories
}

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

// WithVersion sets the schema version to use.
// Deprecated: use WithSchemaVersion instead.
func WithVersion(schemaVersion string) SchemaOption {
	return WithSchemaVersion(schemaVersion)
}

// Schema provides access to OASF schema definitions via API.
type Schema struct {
	schemaURL  string // Normalized schema URL
	httpClient *http.Client
	cache      atomic.Pointer[schemaCache]
}

// normalizeURL normalizes a schema URL by removing trailing slashes and adding protocol if missing.
func normalizeURL(schemaURL string) string {
	normalizedURL := strings.TrimSuffix(schemaURL, "/")
	if !strings.HasPrefix(normalizedURL, "http://") && !strings.HasPrefix(normalizedURL, "https://") {
		normalizedURL = "http://" + normalizedURL
	}

	return normalizedURL
}

// New creates a new Schema instance with the given schema base URL.
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

func schemaVersionFromVersionInfo(versionInfo VersionInfo) string {
	if versionInfo.SchemaVersion != "" {
		return versionInfo.SchemaVersion
	}

	return versionInfo.Version
}

// cloneCategories deep-copies SchemaCategories before returning it to callers.
// SchemaCategories is a map type, so returning cached maps directly would expose
// mutable internal cache state and allow external code to modify it.
func cloneCategories(src SchemaCategories) SchemaCategories {
	dst := make(SchemaCategories, len(src))
	for k, v := range src {
		copied := v
		if len(v.Classes) > 0 {
			copied.Classes = cloneCategories(v.Classes)
		}
		dst[k] = copied
	}

	return dst
}

// getVersionsResponse fetches the versions response from the server.
func (s *Schema) getVersionsResponse(ctx context.Context) (*VersionsResponse, error) {
	versionsURL := s.schemaURL + apiVersionsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, versionsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request to %s: %w", versionsURL, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GET request to %s: %w", versionsURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch versions from URL %s: HTTP %d, body: %s", versionsURL, resp.StatusCode, string(body))
	}

	var versionsResp VersionsResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&versionsResp); err != nil {
		return nil, fmt.Errorf("failed to decode versions response from URL %s: %w", versionsURL, err)
	}

	return &versionsResp, nil
}

// GetDefaultSchemaVersion returns the default schema version.
// If cache exists, it is served from cache.
func (s *Schema) GetDefaultSchemaVersion(ctx context.Context) (string, error) {
	if cached := s.cache.Load(); cached != nil && cached.defaultSchemaVersion != "" {
		return cached.defaultSchemaVersion, nil
	}

	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return "", err
	}

	defaultSchemaVersion := schemaVersionFromVersionInfo(versionsResp.Default)
	if defaultSchemaVersion == "" {
		return "", errors.New("default schema version is missing from /api/versions response")
	}

	return defaultSchemaVersion, nil
}

// GetAvailableSchemaVersions returns supported schema versions.
// If cache exists, it is served from cache.
func (s *Schema) GetAvailableSchemaVersions(ctx context.Context) ([]string, error) {
	if cached := s.cache.Load(); cached != nil && len(cached.availableSchemaVersions) > 0 {
		return append([]string(nil), cached.availableSchemaVersions...), nil
	}

	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return nil, err
	}

	schemaVersions := make([]string, 0, len(versionsResp.Versions))
	for _, v := range versionsResp.Versions {
		schemaVersion := schemaVersionFromVersionInfo(v)
		if schemaVersion != "" {
			schemaVersions = append(schemaVersions, schemaVersion)
		}
	}

	return schemaVersions, nil
}

func (s *Schema) ensureVersionSupported(ctx context.Context, schemaVersion string) error {
	versions, err := s.GetAvailableSchemaVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available versions: %w", err)
	}

	for _, v := range versions {
		if v == schemaVersion {
			return nil
		}
	}

	return fmt.Errorf("schema version %q is not supported", schemaVersion)
}

func (s *Schema) resolveVersion(ctx context.Context, schemaVersion string) (string, error) {
	if schemaVersion != "" {
		if err := s.ensureVersionSupported(ctx, schemaVersion); err != nil {
			return "", err
		}
		return schemaVersion, nil
	}

	defaultSchemaVersion, err := s.GetDefaultSchemaVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get default version: %w", err)
	}

	return defaultSchemaVersion, nil
}

func (s *Schema) getCachedCategories(endpoint string, schemaVersion string) (SchemaCategories, bool) {
	cached := s.cache.Load()
	if cached == nil {
		return nil, false
	}

	switch endpoint {
	case "skill_categories":
		c, ok := cached.schemaSkills[schemaVersion]
		if !ok {
			return nil, false
		}
		return cloneCategories(c), true
	case "domain_categories":
		c, ok := cached.schemaDomains[schemaVersion]
		if !ok {
			return nil, false
		}
		return cloneCategories(c), true
	case "module_categories":
		c, ok := cached.schemaModules[schemaVersion]
		if !ok {
			return nil, false
		}
		return cloneCategories(c), true
	default:
		return nil, false
	}
}

func (s *Schema) fetchCategoriesForVersion(ctx context.Context, version string, endpoint string) (SchemaCategories, error) {
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

// Cache preloads and snapshots supported versions and nested categories.
// Snapshot is version-keyed and immutable until Cache is called again.
func (s *Schema) Cache(ctx context.Context) error {
	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return err
	}

	defaultSchemaVersion := schemaVersionFromVersionInfo(versionsResp.Default)
	if defaultSchemaVersion == "" {
		return errors.New("default schema version is missing from /api/versions response")
	}

	schemaVersions := make([]string, 0, len(versionsResp.Versions))
	for _, v := range versionsResp.Versions {
		schemaVersion := schemaVersionFromVersionInfo(v)
		if schemaVersion != "" {
			schemaVersions = append(schemaVersions, schemaVersion)
		}
	}

	next := &schemaCache{
		defaultSchemaVersion:    defaultSchemaVersion,
		availableSchemaVersions: append([]string(nil), schemaVersions...),
		schemaSkills:            make(map[string]SchemaCategories, len(schemaVersions)),
		schemaDomains:           make(map[string]SchemaCategories, len(schemaVersions)),
		schemaModules:           make(map[string]SchemaCategories, len(schemaVersions)),
	}

	for _, version := range schemaVersions {
		skills, err := s.fetchCategoriesForVersion(ctx, version, "skill_categories")
		if err != nil {
			return fmt.Errorf("failed to cache skills for version %s: %w", version, err)
		}
		domains, err := s.fetchCategoriesForVersion(ctx, version, "domain_categories")
		if err != nil {
			return fmt.Errorf("failed to cache domains for version %s: %w", version, err)
		}
		modules, err := s.fetchCategoriesForVersion(ctx, version, "module_categories")
		if err != nil {
			return fmt.Errorf("failed to cache modules for version %s: %w", version, err)
		}

		next.schemaSkills[version] = cloneCategories(skills)
		next.schemaDomains[version] = cloneCategories(domains)
		next.schemaModules[version] = cloneCategories(modules)
	}

	s.cache.Store(next)
	return nil
}

// constructSchemaURL builds the schema URL from options.
// Format: /schema/<version>/<type>/<name>.
func (s *Schema) constructSchemaURL(version string, schemaType SchemaType, name string) string {
	return fmt.Sprintf("%s/schema/%s/%s/%s", s.schemaURL, version, schemaType, name)
}

// GetSchema is a generic function to fetch schema content from the OASF API.
// It constructs the URL as /schema/<version>/<type>/<name>.
func (s *Schema) GetSchema(ctx context.Context, schemaType SchemaType, name string, opts ...SchemaOption) ([]byte, error) {
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	schemaVersion, err := s.resolveVersion(ctx, options.schemaVersion)
	if err != nil {
		return nil, err
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
func (s *Schema) GetSchemaCategories(ctx context.Context, endpoint string, opts ...SchemaOption) (SchemaCategories, error) {
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	version, err := s.resolveVersion(ctx, options.schemaVersion)
	if err != nil {
		return nil, err
	}

	if categories, ok := s.getCachedCategories(endpoint, version); ok {
		return categories, nil
	}

	return s.fetchCategoriesForVersion(ctx, version, endpoint)
}

// GetRecordSchemaContent returns the raw JSON schema content for a given version.
func (s *Schema) GetRecordSchemaContent(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	return s.GetSchema(ctx, SchemaTypeObjects, "record", opts...)
}

// GetSchemaSkills returns nested skill categories from /api/<version>/skill_categories.
func (s *Schema) GetSchemaSkills(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, "skill_categories", opts...)
}

// GetSchemaDomains returns nested domain categories from /api/<version>/domain_categories.
func (s *Schema) GetSchemaDomains(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, "domain_categories", opts...)
}

// GetSchemaModules returns nested module categories from /api/<version>/module_categories.
func (s *Schema) GetSchemaModules(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, "module_categories", opts...)
}
