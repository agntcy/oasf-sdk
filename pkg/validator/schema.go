// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/xeipuuv/gojsonschema"
)

//go:embed schemas/*.json
var embeddedSchemas embed.FS

func loadEmbeddedSchemas() (map[string]*gojsonschema.Schema, error) {
	schemas := make(map[string]*gojsonschema.Schema)

	entries, err := embeddedSchemas.ReadDir("schemas")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schemas directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filename := entry.Name()
		version := strings.TrimSuffix(filename, ".json")

		schemaPath := filepath.Join("schemas", filename)

		schemaData, err := embeddedSchemas.ReadFile(schemaPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read embedded schema file %s: %w", filename, err)
		}

		schemaLoader := gojsonschema.NewStringLoader(string(schemaData))

		schema, err := gojsonschema.NewSchema(schemaLoader)
		if err != nil {
			return nil, fmt.Errorf("failed to compile embedded schema %s: %w", filename, err)
		}

		schemas[version] = schema
	}

	if len(schemas) == 0 {
		return nil, errors.New("no valid JSON schema files found in embedded schemas")
	}

	return schemas, nil
}

// GetSchemaContent returns the raw JSON schema content for a given version.
// Returns an error if the version is not found or if there's an issue reading the schema.
func GetSchemaContent(version string) ([]byte, error) {
	filename := version + ".json"
	schemaPath := filepath.Join("schemas", filename)

	schemaData, err := embeddedSchemas.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("schema version '%s' not found: %w", version, err)
	}

	return schemaData, nil
}

// GetAvailableSchemaVersions returns a list of all available schema versions.
func GetAvailableSchemaVersions() ([]string, error) {
	entries, err := embeddedSchemas.ReadDir("schemas")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded schemas directory: %w", err)
	}

	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filename := entry.Name()
		version := strings.TrimSuffix(filename, ".json")
		versions = append(versions, version)
	}

	if len(versions) == 0 {
		return nil, errors.New("no valid JSON schema files found in embedded schemas")
	}

	return versions, nil
}

// GetSchemaKey is a generic function to extract any $defs category from a schema.
// For example, extracting skills, domains, modules, or any other $defs key.
// Returns the category definitions as JSON bytes, or an error if not found.
func GetSchemaKey(version, defsKey string) ([]byte, error) {
	schemaData, err := GetSchemaContent(version)
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
func GetSchemaSkills(version string) ([]byte, error) {
	return GetSchemaKey(version, "skills")
}

// GetSchemaDomains is a convenience function to extract domains from a schema.
// Returns the domains as JSON bytes, or an error if the version is not found or parsing fails.
func GetSchemaDomains(version string) ([]byte, error) {
	return GetSchemaKey(version, "domains")
}

// GetSchemaModules is a convenience function to extract modules from a schema.
// Returns the modules as JSON bytes, or an error if the version is not found or parsing fails.
func GetSchemaModules(version string) ([]byte, error) {
	return GetSchemaKey(version, "modules")
}
