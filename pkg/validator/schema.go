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
