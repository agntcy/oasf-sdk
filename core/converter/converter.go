// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package converter

import (
	"errors"
	"fmt"

	corev1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/core/v1"
	typesv1alpha1 "buf.build/gen/go/agntcy/oasf/protocolbuffers/go/types/v1alpha1"
	"github.com/agntcy/oasf-sdk/core/utils"
)

func GetSchemaVersion(in *corev1.EncodedRecord) (string, error) {
	if in == nil || in.GetRecord() == nil {
		return "", errors.New("request is nil")
	}

	// Extract the schema_version field from the record
	fieldSchemaVersion := in.GetRecord().Fields["schema_version"]
	if fieldSchemaVersion == nil {
		return "", errors.New("schema_version field is missing")
	}

	return fieldSchemaVersion.GetStringValue(), nil
}

func Decode(in *corev1.EncodedRecord) (*corev1.DecodedRecord, error) {
	// Validate input
	if in == nil || in.GetRecord() == nil {
		return nil, errors.New("request is nil")
	}

	// Get schema version
	schemaVersion, err := GetSchemaVersion(in)
	if err != nil {
		return nil, err
	}

	// Decode data based on schema version
	switch schemaVersion {
	case "v0.7.0":
		record, err := utils.ProtoToObject[typesv1alpha1.Record](in.GetRecord())
		if err != nil {
			return nil, fmt.Errorf("failed to convert %s record: %w", schemaVersion, err)
		}

		return &corev1.DecodedRecord{
			Record: &corev1.DecodedRecord_V1Alpha1{
				V1Alpha1: record,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported OASF version: %s", schemaVersion)
	}
}

func Encode(in *corev1.DecodedRecord) (*corev1.EncodedRecord, error) {
	// Extract the inner record based on its type
	var record *typesv1alpha1.Record
	switch v := in.GetRecord().(type) {
	case *corev1.DecodedRecord_V1Alpha1:
		record = v.V1Alpha1
	default:
		return nil, errors.New("unsupported record type")
	}

	// Convert record to proto
	encoded, err := utils.ObjectToProto(record)
	if err != nil {
		return nil, fmt.Errorf("failed to convert record to proto: %w", err)
	}

	return &corev1.EncodedRecord{
		Record: encoded,
	}, nil
}
