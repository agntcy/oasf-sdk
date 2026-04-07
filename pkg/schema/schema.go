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

// TaxonomyItem represents a nested taxonomy node returned by *_categories endpoints.
type TaxonomyItem struct {
	ID          int                     `json:"id"`
	Name        string                  `json:"name"`
	Description string                  `json:"description,omitempty"`
	Category    bool                    `json:"category,omitempty"`
	Caption     string                  `json:"caption,omitempty"`
	Deprecated  bool                    `json:"deprecated,omitempty"`
	Classes     map[string]TaxonomyItem `json:"classes,omitempty"`
}

// Taxonomy is the top-level category map keyed by category slug.
type Taxonomy map[string]TaxonomyItem

type schemaCache struct {
	defaultSchemaVersion    string
	availableSchemaVersions []string
	skills                  map[string]Taxonomy
	domains                 map[string]Taxonomy
	modules                 map[string]Taxonomy
	jsonSchema              map[string]map[string]jsonSchemaCacheEntry
}

type jsonSchemaCacheEntry struct {
	schemaType EntityType
	name       string
	data       []byte
}

// SchemaOption is a function that configures schema query options.
type SchemaOption func(*schemaOptions)

// ConstructorOption is a function that configures schema client behavior.
type ConstructorOption func(*constructorOptions)

// EntityType represents the type of schema to fetch.
type EntityType string

const (
	// EntityTypeObjects represents object schemas (agent, record).
	EntityTypeObjects EntityType = "objects"
	// EntityTypeModules represents module schemas.
	EntityTypeModules EntityType = "modules"
	// EntityTypeSkills represents skill schemas.
	EntityTypeSkills EntityType = "skills"
	// EntityTypeDomains represents domain schemas.
	EntityTypeDomains EntityType = "domains"
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
			skills:     map[string]Taxonomy{},
			domains:    map[string]Taxonomy{},
			modules:    map[string]Taxonomy{},
			jsonSchema: map[string]map[string]jsonSchemaCacheEntry{},
		},
	}, nil
}

// cloneTaxonomy deep-copies Taxonomy before returning it to callers.
// Taxonomy is a map type, so returning cached maps directly would expose
// mutable internal cache state and allow external code to modify it.
func cloneTaxonomy(src Taxonomy) Taxonomy {
	return cloneTaxonomyItems(map[string]TaxonomyItem(src))
}

// cloneTaxonomyItems recursively deep-copies a map of TaxonomyItem nodes.
// Used for both the top-level Taxonomy and nested Classes maps, which share
// the same underlying type.
func cloneTaxonomyItems(src map[string]TaxonomyItem) map[string]TaxonomyItem {
	dst := make(map[string]TaxonomyItem, len(src))
	for k, v := range src {
		copied := v
		if len(v.Classes) > 0 {
			copied.Classes = cloneTaxonomyItems(v.Classes)
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

func (s *Schema) getCachedTaxonomy(endpoint string, schemaVersion string) (Taxonomy, bool) {
	if !s.cacheEnabled {
		return nil, false
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	switch endpoint {
	case skillCategoriesEndpoint:
		c, ok := s.cache.skills[schemaVersion]
		if !ok {
			return nil, false
		}

		return cloneTaxonomy(c), true
	case domainCategoriesEndpoint:
		c, ok := s.cache.domains[schemaVersion]
		if !ok {
			return nil, false
		}

		return cloneTaxonomy(c), true
	case moduleCategoriesEndpoint:
		c, ok := s.cache.modules[schemaVersion]
		if !ok {
			return nil, false
		}

		return cloneTaxonomy(c), true
	default:
		return nil, false
	}
}

func (s *Schema) setCachedTaxonomy(endpoint string, schemaVersion string, categories Taxonomy) {
	if !s.cacheEnabled {
		return
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	switch endpoint {
	case skillCategoriesEndpoint:
		s.cache.skills[schemaVersion] = cloneTaxonomy(categories)
	case domainCategoriesEndpoint:
		s.cache.domains[schemaVersion] = cloneTaxonomy(categories)
	case moduleCategoriesEndpoint:
		s.cache.modules[schemaVersion] = cloneTaxonomy(categories)
	}
}

func jsonSchemaCacheKey(schemaType EntityType, name string) string {
	return fmt.Sprintf("%s|%s", schemaType, name)
}

func (s *Schema) getCachedJSONSchema(schemaVersion string, schemaType EntityType, name string) ([]byte, bool) {
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

func (s *Schema) setCachedJSONSchema(schemaVersion string, schemaType EntityType, name string, data []byte) {
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

func (s *Schema) fetchTaxonomyForVersion(ctx context.Context, version string, endpoint string) (Taxonomy, error) {
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

	var taxonomy Taxonomy

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&taxonomy); err != nil {
		return nil, fmt.Errorf("failed to decode categories response from URL %s: %w", categoriesURL, err)
	}

	return taxonomy, nil
}

// ClearCache removes the current cache snapshot.
func (s *Schema) ClearCache() {
	if !s.cacheEnabled {
		return
	}

	s.cacheMu.Lock()
	s.cache = &schemaCache{
		skills:     map[string]Taxonomy{},
		domains:    map[string]Taxonomy{},
		modules:    map[string]Taxonomy{},
		jsonSchema: map[string]map[string]jsonSchemaCacheEntry{},
	}
	s.cacheMu.Unlock()
}

// constructSchemaURL builds the schema URL from options.
// Format: /schema/<version>/<type>/<name>.
func (s *Schema) constructSchemaURL(version string, schemaType EntityType, name string) string {
	return fmt.Sprintf("%s/schema/%s/%s/%s", s.schemaURL, version, schemaType, name)
}

// GetJSONSchema is a generic function to fetch JSON schema content from the OASF API.
// It constructs the URL as /schema/<version>/<type>/<name>.
func (s *Schema) GetJSONSchema(ctx context.Context, schemaType EntityType, name string, opts ...SchemaOption) ([]byte, error) {
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

// GetSchemaTaxonomy fetches nested taxonomy categories from /api/<version>/<endpoint>.
func (s *Schema) GetSchemaTaxonomy(ctx context.Context, endpoint string, opts ...SchemaOption) (Taxonomy, error) {
	options := &schemaOptions{}
	for _, opt := range opts {
		opt(options)
	}

	version, err := s.resolveVersion(ctx, options.schemaVersion)
	if err != nil {
		return nil, err
	}

	if categories, ok := s.getCachedTaxonomy(endpoint, version); ok {
		return categories, nil
	}

	categories, err := s.fetchTaxonomyForVersion(ctx, version, endpoint)
	if err != nil {
		return nil, err
	}

	s.setCachedTaxonomy(endpoint, version, categories)

	return categories, nil
}

// GetRecordJSONSchema returns the record JSON schema content for a given version.
func (s *Schema) GetRecordJSONSchema(ctx context.Context, opts ...SchemaOption) ([]byte, error) {
	return s.GetJSONSchema(ctx, EntityTypeObjects, "record", opts...)
}

// GetSchemaSkills returns nested skill categories from /api/<version>/skill_categories.
func (s *Schema) GetSchemaSkills(ctx context.Context, opts ...SchemaOption) (Taxonomy, error) {
	return s.GetSchemaTaxonomy(ctx, skillCategoriesEndpoint, opts...)
}

// GetSchemaDomains returns nested domain categories from /api/<version>/domain_categories.
func (s *Schema) GetSchemaDomains(ctx context.Context, opts ...SchemaOption) (Taxonomy, error) {
	return s.GetSchemaTaxonomy(ctx, domainCategoriesEndpoint, opts...)
}

// GetSchemaModules returns nested module categories from /api/<version>/module_categories.
func (s *Schema) GetSchemaModules(ctx context.Context, opts ...SchemaOption) (Taxonomy, error) {
	return s.GetSchemaTaxonomy(ctx, moduleCategoriesEndpoint, opts...)
}
