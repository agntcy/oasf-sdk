// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoder

import (
	"errors"
	"fmt"

	decodingv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/decoding/v1"
	corev1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/core/v1"
	typesv1alpha0 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha0"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha1"
)

// Field name in the record that indicates the schema version.
const schemaVersionField = "schema_version"

// DecodeRecord decodes a Record object into a structured format based on its schema version.
func DecodeRecord(in *corev1.Object) (*decodingv1.DecodeRecordResponse, error) {
	// Validate input
	if in == nil || in.GetData() == nil {
		return nil, errors.New("request is nil")
	}

	// Get schema version
	schemaVersion, err := GetRecordSchemaVersion(in)
	if err != nil {
		return nil, err
	}

	// Decode data based on schema version
	switch schemaVersion {
	case "v0.3.1":
		record, err := ProtoToStruct[typesv1alpha0.Record](in.GetData())
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &decodingv1.DecodeRecordResponse{
			Record: &decodingv1.DecodeRecordResponse_V1Alpha0{
				V1Alpha0: record,
			},
		}, nil

	case "v0.7.0":
		record, err := ProtoToStruct[typesv1alpha1.Record](in.GetData())
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &decodingv1.DecodeRecordResponse{
			Record: &decodingv1.DecodeRecordResponse_V1Alpha1{
				V1Alpha1: record,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported OASF version: %s", schemaVersion)
	}
}

// GetRecordSchemaVersion extracts the schema version from the Record object.
func GetRecordSchemaVersion(in *corev1.Object) (string, error) {
	if in == nil || in.GetData() == nil {
		return "", errors.New("request is nil")
	}

	// Extract the schema_version field from the record
	fieldSchemaVersion := in.GetData().Fields[schemaVersionField]
	if fieldSchemaVersion == nil {
		return "", fmt.Errorf("%s field is missing", schemaVersionField)
	}

	return fieldSchemaVersion.GetStringValue(), nil
}
