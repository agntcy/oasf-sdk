package utils

import (
	"encoding/json"

	"google.golang.org/protobuf/types/known/structpb"
)

// ObjectToProto converts a generic Go object to a protobuf Struct.
func ObjectToProto(obj any) (*structpb.Struct, error) {
	jsonBytes, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	var result *structpb.Struct
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil, err
	}

	return result, nil
}
