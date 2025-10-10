// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package validator

import (
	"embed"
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
		return nil, fmt.Errorf("no valid JSON schema files found in embedded schemas")
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

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filename := entry.Name()
		version := strings.TrimSuffix(filename, ".json")
		versions = append(versions, version)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no valid JSON schema files found in embedded schemas")
	}

	return versions, nil
}
