// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoder

import (
	"errors"
	"fmt"
	"strings"

	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/decoding/v1"
	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	typesv1alpha2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"github.com/Masterminds/semver/v3"
	"google.golang.org/protobuf/types/known/structpb"
)

// Field name in the record that indicates the schema version.
const schemaVersionField = "schema_version"

const expectedVersionParts = 3 // Expected number of parts (major.minor.patch) in a version string

// getProtoVersionForSchemaVersion determines which proto version to use based on the OASF schema version.
//   - 0.7.x -> v1alpha1
//   - 0.8.x -> v1alpha2
//   - 1.x.x -> v1
func getProtoVersionForSchemaVersion(schemaVersion string) (string, error) {
	// Reject versions with "v" prefix as OASF versions don't use it
	if strings.HasPrefix(schemaVersion, "v") || strings.HasPrefix(schemaVersion, "V") {
		return "", fmt.Errorf("invalid version format: %s (OASF versions must not have 'v' prefix, expected format: major.minor.patch)", schemaVersion)
	}

	// Validate that version has exactly 3 parts (major.minor.patch)
	// semver.NewVersion() accepts formats like "1.0" and normalizes them, but we require strict x.x.x format
	parts := strings.Split(schemaVersion, ".")
	if len(parts) != expectedVersionParts {
		return "", fmt.Errorf("invalid version format: %s (expected exactly %d parts: major.minor.patch)", schemaVersion, expectedVersionParts)
	}

	// Parse version using semver library for validation and parsing
	version, err := semver.NewVersion(schemaVersion)
	if err != nil {
		return "", fmt.Errorf("failed to parse schema version %s: %w (expected semver format: major.minor.patch)", schemaVersion, err)
	}

	major := version.Major()
	minor := version.Minor()

	// Map major.minor versions to proto versions
	// Proto changes only occur with major version changes
	if major == 0 && minor == 7 {
		return "v1alpha1", nil
	}

	if major == 0 && minor == 8 {
		return "v1alpha2", nil
	}

	if major == 1 {
		return "v1", nil
	}

	return "", fmt.Errorf("unsupported OASF version: %s (major version %d not supported)", schemaVersion, major)
}

// DecodeRecord decodes a Record object into a structured format based on its schema version.
func DecodeRecord(record *structpb.Struct) (*decodingv1.DecodeRecordResponse, error) {
	// Validate input
	if record == nil {
		return nil, errors.New("request is nil")
	}

	// Get schema version
	schemaVersion, err := GetRecordSchemaVersion(record)
	if err != nil {
		return nil, err
	}

	// Determine which proto version to use based on the schema version
	protoVersion, err := getProtoVersionForSchemaVersion(schemaVersion)
	if err != nil {
		return nil, err
	}

	// Decode data based on proto version
	switch protoVersion {
	case "v1alpha1":
		record, err := ProtoToStruct[typesv1alpha1.Record](record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &decodingv1.DecodeRecordResponse{
			Record: &decodingv1.DecodeRecordResponse_V1Alpha1{
				V1Alpha1: record,
			},
		}, nil

	case "v1alpha2":
		record, err := ProtoToStruct[typesv1alpha2.Record](record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &decodingv1.DecodeRecordResponse{
			Record: &decodingv1.DecodeRecordResponse_V1Alpha2{
				V1Alpha2: record,
			},
		}, nil

	case "v1":
		record, err := ProtoToStruct[typesv1.Record](record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &decodingv1.DecodeRecordResponse{
			Record: &decodingv1.DecodeRecordResponse_V1{
				V1: record,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported proto version: %s (for schema version %s)", protoVersion, schemaVersion)
	}
}

// GetRecordSchemaVersion extracts the schema version from the Record object.
func GetRecordSchemaVersion(record *structpb.Struct) (string, error) {
	if record == nil {
		return "", errors.New("request is nil")
	}

	// Extract the schema_version field from the record
	fieldSchemaVersion := record.GetFields()[schemaVersionField]
	if fieldSchemaVersion == nil {
		return "", fmt.Errorf("%s field is missing", schemaVersionField)
	}

	return fieldSchemaVersion.GetStringValue(), nil
}
