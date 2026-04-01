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
	"sync"
	"time"
)

const (
	defaultHTTPTimeoutSeconds = 30
	apiVersionsPath           = "/api/versions"
	skillCategoriesEndpoint   = "skill_categories"
	domainCategoriesEndpoint  = "domain_categories"
	moduleCategoriesEndpoint  = "module_categories"
)

// VersionsResponse represents the response from the api/versions endpoint.
type VersionsResponse struct {
	Default  VersionInfo   `json:"default"`
	Versions []VersionInfo `json:"versions"`
}

// VersionInfo represents schema server version metadata.
type VersionInfo struct {
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
	jsonSchema              map[string]map[string]jsonSchemaCacheEntry
}

type jsonSchemaCacheEntry struct {
	schemaType SchemaType
	name       string
	data       []byte
}

// SchemaOption is a function that configures schema query options.
type SchemaOption func(*schemaOptions)

// ConstructorOption is a function that configures schema client behavior.
type ConstructorOption func(*constructorOptions)

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

type constructorOptions struct {
	enableCache bool
}

// WithCache enables or disables dynamic in-memory caching.
// Disabled by default.
func WithCache(enabled bool) ConstructorOption {
	return func(opts *constructorOptions) {
		opts.enableCache = enabled
	}
}

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
	schemaURL    string // Normalized schema URL
	httpClient   *http.Client
	cacheEnabled bool
	cacheMu      sync.RWMutex
	cache        *schemaCache
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
func New(schemaURL string, opts ...ConstructorOption) (*Schema, error) {
	if schemaURL == "" {
		return nil, errors.New("schema URL is required")
	}

	options := &constructorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	return &Schema{
		schemaURL:    normalizeURL(schemaURL),
		cacheEnabled: options.enableCache,
		httpClient: &http.Client{
			Timeout: defaultHTTPTimeoutSeconds * time.Second,
		},
		cache: &schemaCache{
			schemaSkills:  map[string]SchemaCategories{},
			schemaDomains: map[string]SchemaCategories{},
			schemaModules: map[string]SchemaCategories{},
			jsonSchema:    map[string]map[string]jsonSchemaCacheEntry{},
		},
	}, nil
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

func (s *Schema) extractVersions(resp *VersionsResponse) (string, []string, error) {
	defaultSchemaVersion := resp.Default.SchemaVersion
	if defaultSchemaVersion == "" {
		return "", nil, errors.New("default schema version is missing from /api/versions response")
	}

	schemaVersions := make([]string, 0, len(resp.Versions))
	for _, v := range resp.Versions {
		if v.SchemaVersion != "" {
			schemaVersions = append(schemaVersions, v.SchemaVersion)
		}
	}

	return defaultSchemaVersion, schemaVersions, nil
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
func (s *Schema) GetDefaultSchemaVersion(ctx context.Context) (string, error) {
	if s.cacheEnabled {
		s.cacheMu.RLock()

		defaultSchemaVersion := s.cache.defaultSchemaVersion
		s.cacheMu.RUnlock()

		if defaultSchemaVersion != "" {
			return defaultSchemaVersion, nil
		}
	}

	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return "", err
	}

	defaultSchemaVersion, schemaVersions, err := s.extractVersions(versionsResp)
	if err != nil {
		return "", err
	}

	if s.cacheEnabled {
		s.cacheMu.Lock()
		s.cache.defaultSchemaVersion = defaultSchemaVersion

		s.cache.availableSchemaVersions = append([]string(nil), schemaVersions...)
		s.cacheMu.Unlock()
	}

	return defaultSchemaVersion, nil
}

// GetAvailableSchemaVersions returns supported schema versions.
func (s *Schema) GetAvailableSchemaVersions(ctx context.Context) ([]string, error) {
	if s.cacheEnabled {
		s.cacheMu.RLock()

		cachedVersions := append([]string(nil), s.cache.availableSchemaVersions...)
		s.cacheMu.RUnlock()

		if len(cachedVersions) > 0 {
			return cachedVersions, nil
		}
	}

	versionsResp, err := s.getVersionsResponse(ctx)
	if err != nil {
		return nil, err
	}

	defaultSchemaVersion, schemaVersions, err := s.extractVersions(versionsResp)
	if err != nil {
		return nil, err
	}

	if s.cacheEnabled {
		s.cacheMu.Lock()
		s.cache.defaultSchemaVersion = defaultSchemaVersion

		s.cache.availableSchemaVersions = append([]string(nil), schemaVersions...)
		s.cacheMu.Unlock()
	}

	return schemaVersions, nil
}

func (s *Schema) ensureVersionSupported(ctx context.Context, schemaVersion string) error {
	versions, err := s.GetAvailableSchemaVersions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get available versions: %w", err)
	}

	if slices.Contains(versions, schemaVersion) {
		return nil
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
	if !s.cacheEnabled {
		return nil, false
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	switch endpoint {
	case skillCategoriesEndpoint:
		c, ok := s.cache.schemaSkills[schemaVersion]
		if !ok {
			return nil, false
		}

		return cloneCategories(c), true
	case domainCategoriesEndpoint:
		c, ok := s.cache.schemaDomains[schemaVersion]
		if !ok {
			return nil, false
		}

		return cloneCategories(c), true
	case moduleCategoriesEndpoint:
		c, ok := s.cache.schemaModules[schemaVersion]
		if !ok {
			return nil, false
		}

		return cloneCategories(c), true
	default:
		return nil, false
	}
}

func (s *Schema) setCachedCategories(endpoint string, schemaVersion string, categories SchemaCategories) {
	if !s.cacheEnabled {
		return
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	switch endpoint {
	case skillCategoriesEndpoint:
		s.cache.schemaSkills[schemaVersion] = cloneCategories(categories)
	case domainCategoriesEndpoint:
		s.cache.schemaDomains[schemaVersion] = cloneCategories(categories)
	case moduleCategoriesEndpoint:
		s.cache.schemaModules[schemaVersion] = cloneCategories(categories)
	}
}

func jsonSchemaCacheKey(schemaType SchemaType, name string) string {
	return fmt.Sprintf("%s|%s", schemaType, name)
}

func (s *Schema) getCachedJSONSchema(schemaVersion string, schemaType SchemaType, name string) ([]byte, bool) {
	if !s.cacheEnabled {
		return nil, false
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	byVersion, ok := s.cache.jsonSchema[schemaVersion]
	if !ok {
		return nil, false
	}

	entry, ok := byVersion[jsonSchemaCacheKey(schemaType, name)]
	if !ok {
		return nil, false
	}

	return append([]byte(nil), entry.data...), true
}

func (s *Schema) setCachedJSONSchema(schemaVersion string, schemaType SchemaType, name string, data []byte) {
	if !s.cacheEnabled {
		return
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	byVersion, ok := s.cache.jsonSchema[schemaVersion]
	if !ok {
		byVersion = map[string]jsonSchemaCacheEntry{}
		s.cache.jsonSchema[schemaVersion] = byVersion
	}

	byVersion[jsonSchemaCacheKey(schemaType, name)] = jsonSchemaCacheEntry{
		schemaType: schemaType,
		name:       name,
		data:       append([]byte(nil), data...),
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

// ClearCache removes the current cache snapshot.
func (s *Schema) ClearCache() {
	if !s.cacheEnabled {
		return
	}

	s.cacheMu.Lock()
	s.cache = &schemaCache{
		schemaSkills:  map[string]SchemaCategories{},
		schemaDomains: map[string]SchemaCategories{},
		schemaModules: map[string]SchemaCategories{},
		jsonSchema:    map[string]map[string]jsonSchemaCacheEntry{},
	}
	s.cacheMu.Unlock()
}

// constructSchemaURL builds the schema URL from options.
// Format: /schema/<version>/<type>/<name>.
func (s *Schema) constructSchemaURL(version string, schemaType SchemaType, name string) string {
	return fmt.Sprintf("%s/schema/%s/%s/%s", s.schemaURL, version, schemaType, name)
}

// GetJSONSchema is a generic function to fetch JSON schema content from the OASF API.
// It constructs the URL as /schema/<version>/<type>/<name>.
func (s *Schema) GetJSONSchema(ctx context.Context, schemaType SchemaType, name string, opts ...SchemaOption) ([]byte, error) {
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	schemaVersion, err := s.resolveVersion(ctx, options.schemaVersion)
	if err != nil {
		return nil, err
	}

	if cached, ok := s.getCachedJSONSchema(schemaVersion, schemaType, name); ok {
		return cached, nil
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

	s.setCachedJSONSchema(schemaVersion, schemaType, name, schemaData)

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

	categories, err := s.fetchCategoriesForVersion(ctx, version, endpoint)
	if err != nil {
		return nil, err
	}

	s.setCachedCategories(endpoint, version, categories)

	return categories, nil
}

// GetRecordJSONSchema returns the record JSON schema content for a given version.
func (s *Schema) GetRecordJSONSchema(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	return s.GetJSONSchema(ctx, SchemaTypeObjects, "record", opts...)
}

// GetSchemaSkills returns nested skill categories from /api/<version>/skill_categories.
func (s *Schema) GetSchemaSkills(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, skillCategoriesEndpoint, opts...)
}

// GetSchemaDomains returns nested domain categories from /api/<version>/domain_categories.
func (s *Schema) GetSchemaDomains(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, domainCategoriesEndpoint, opts...)
}

// GetSchemaModules returns nested module categories from /api/<version>/module_categories.
func (s *Schema) GetSchemaModules(ctx context.Context, opts ...SchemaOption) (SchemaCategories, error) {
	return s.GetSchemaCategories(ctx, moduleCategoriesEndpoint, opts...)
}
