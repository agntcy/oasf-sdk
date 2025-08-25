// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	validationv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/validation/v1"
	objectsv3 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/objects/v3"
	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed schemas/*.json
var embeddedSchemas embed.FS

type ValidationService struct {
	schemas    map[string]*gojsonschema.Schema
	httpClient *http.Client
}

func NewValidationService() (*ValidationService, error) {
	schemas, err := loadEmbeddedSchemas()
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded schemas: %w", err)
	}

	return &ValidationService{
		schemas: schemas,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (v ValidationService) ValidateRecord(req *validationv1.ValidateRecordRequest) (bool, []string, error) {
	if req.Record == nil {
		return false, []string{"record cannot be nil"}, nil
	}

	if req.SchemaUrl != "" {
		schemaErrors, err := v.validateWithSchemaURL(req.Record, req.SchemaUrl)
		if err != nil {
			return false, nil, fmt.Errorf("schema URL validation failed: %w", err)
		}

		return len(schemaErrors) == 0, schemaErrors, nil
	}

	schema, schemaExists := v.schemas[req.Record.SchemaVersion]
	if !schemaExists {
		var availableVersions []string
		for version := range v.schemas {
			availableVersions = append(availableVersions, version)
		}

		return false, nil, fmt.Errorf("no schema found for version %s. Available versions: %v", req.Record.SchemaVersion, availableVersions)
	}

	schemaErrors, err := v.validateWithJSONSchema(req.Record, schema)
	if err != nil {
		return false, nil, fmt.Errorf("JSON schema validation failed: %w", err)
	}

	return len(schemaErrors) == 0, schemaErrors, nil
}

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

func (v ValidationService) validateWithJSONSchema(record *objectsv3.Record, schema *gojsonschema.Schema) ([]string, error) {
	marshaler := &protojson.MarshalOptions{
		UseProtoNames: true,
	}
	jsonBytes, err := marshaler.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal record to JSON: %w", err)
	}

	var recordData interface{}
	if err := json.Unmarshal(jsonBytes, &recordData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	documentLoader := gojsonschema.NewGoLoader(recordData)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return nil, fmt.Errorf("schema validation error: %w", err)
	}

	var errors []string
	if !result.Valid() {
		for _, desc := range result.Errors() {
			errors = append(errors, fmt.Sprintf("JSON Schema: %s", desc.String()))
		}
	}

	return errors, nil
}

func (v ValidationService) validateWithSchemaURL(record *objectsv3.Record, schemaURL string) ([]string, error) {
	resp, err := v.httpClient.Get(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema from URL %s: %w", schemaURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch schema from URL %s: HTTP %d", schemaURL, resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	var schemaData interface{}
	if err := decoder.Decode(&schemaData); err != nil {
		return nil, fmt.Errorf("failed to decode schema JSON from URL %s: %w", schemaURL, err)
	}

	schemaBytes, err := json.Marshal(schemaData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema from URL %s: %w", schemaURL, err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaBytes))
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to compile schema from URL %s: %w", schemaURL, err)
	}

	return v.validateWithJSONSchema(record, schema)
}
