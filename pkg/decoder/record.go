// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoder

import (
	"errors"
	"fmt"

	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/decoding/v1"
	typesv1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha1"
	typesv1alpha2 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/agntcy/oasf/types/v1alpha2"
	"google.golang.org/protobuf/types/known/structpb"
)

// Field name in the record that indicates the schema version.
const schemaVersionField = "schema_version"

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

	// Decode data based on schema version
	switch schemaVersion {
	case "0.7.0":
		record, err := ProtoToStruct[typesv1alpha1.Record](record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &decodingv1.DecodeRecordResponse{
			Record: &decodingv1.DecodeRecordResponse_V1Alpha1{
				V1Alpha1: record,
			},
		}, nil

	case "0.8.0":
		record, err := ProtoToStruct[typesv1alpha2.Record](record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &decodingv1.DecodeRecordResponse{
			Record: &decodingv1.DecodeRecordResponse_V1Alpha2{
				V1Alpha2: record,
			},
		}, nil

	case "1.0.0-rc.1":
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
		return nil, fmt.Errorf("unsupported OASF version: %s", schemaVersion)
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
