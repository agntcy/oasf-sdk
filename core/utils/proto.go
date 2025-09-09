package utils

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
)

// JsonToProto converts a generic JSON object to a protobuf Struct.
func JsonToProto(data []byte) (*structpb.Struct, error) {
	var result *structpb.Struct
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ObjectToProto converts a generic Go object to a protobuf Struct.
func ObjectToProto(obj any) (*structpb.Struct, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	return JsonToProto(jsonBytes)
}

// ProtoToObject converts a protobuf Struct to a generic Go object.
func ProtoToObject[T any](obj *structpb.Struct) (*T, error) {
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
