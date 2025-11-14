// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoder

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

// JsonToProto converts a JSON object to a proto object.
func JsonToProto(data []byte) (*structpb.Struct, error) {
	var result *structpb.Struct
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to protobuf struct: %w", err)
	}

	return result, nil
}

// StructToProto converts a Go struct to a proto object.
func StructToProto(goObj any) (*structpb.Struct, error) {
	jsonBytes, err := json.Marshal(goObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Go struct to JSON: %w", err)
	}

	return JsonToProto(jsonBytes)
}

// ProtoToStruct converts a proto object to a Go struct.
func ProtoToStruct[T any](obj *structpb.Struct) (*T, error) {
	// Convert protobuf Struct to JSON
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal protobuf struct to JSON: %w", err)
	}

	// Unmarshal JSON to the target Go type
	var result T
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to Go struct: %w", err)
	}

	return &result, nil
}
