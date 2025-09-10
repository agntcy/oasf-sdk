// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package decoding

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
)

// JsonToProto converts a JSON object to a proto object.
func JsonToProto(data []byte) (*structpb.Struct, error) {
	var result *structpb.Struct
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// StructToProto converts a Go struct to a proto object.
func StructToProto(goObj any) (*structpb.Struct, error) {
	jsonBytes, err := json.Marshal(goObj)
	if err != nil {
		return nil, err
	}

	return JsonToProto(jsonBytes)
}

// ProtoToStruct converts a proto object to a Go struct.
func ProtoToStruct[T any](obj *structpb.Struct) (*T, error) {
	// Convert protobuf Struct to JSON
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	// Unmarshal JSON to the target Go type
	var result T
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
