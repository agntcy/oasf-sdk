// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package record

import "google.golang.org/protobuf/types/known/structpb"

// GetModule returns the full module struct for the provided module name.
// The returned struct contains all module fields (e.g. "name", "data", "artifact", "id").
func GetModule(record *structpb.Struct, moduleName string) (bool, *structpb.Struct) {
	if record == nil {
		return false, nil
	}

	modules, ok := record.GetFields()["modules"]
	if !ok {
		return false, nil
	}

	for _, module := range modules.GetListValue().GetValues() {
		moduleStruct := module.GetStructValue()
		if moduleStruct == nil {
			continue
		}

		nameField := moduleStruct.GetFields()["name"]
		if nameField == nil {
			continue
		}

		if nameField.GetStringValue() == moduleName {
			return true, moduleStruct
		}
	}

	return false, nil
}
