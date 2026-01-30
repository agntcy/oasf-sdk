// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"google.golang.org/protobuf/types/known/structpb"
)

// Constants for repeated strings.
const (
	serverTypeLocal     = "local"
	serverTypeSSE       = "sse"
	serverTypeHTTP      = "http"
	packageTypeNPM      = "npm"
	packageTypePyPI     = "pypi"
	packageTypeOCI      = "oci"
	packageTypeNuGet    = "nuget"
	packageTypeMCPB     = "mcpb"
	defaultVersion      = "v1.0.0"
	connectionTypeStdio = "stdio"
	connectionTypeSSE   = "sse"
	connectionTypeHTTP  = "streamable-http"
)

const (
	MCPModuleName = "integration/mcp"
	A2AModuleName = "integration/a2a"
	targetSchema  = "1.0.0-rc.1"
)

func getModuleDataFromRecord(record *structpb.Struct, moduleName string) (bool, *structpb.Struct) {
	// Find module by exact name match
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

		// Exact match required (e.g., "integration/a2a")
		if nameField.GetStringValue() == moduleName {
			return true, moduleStruct.GetFields()["data"].GetStructValue()
		}
	}

	return false, nil
}
