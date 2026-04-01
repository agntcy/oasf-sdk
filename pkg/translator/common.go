// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
)

// Constants for repeated strings.
const (
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
	MCPModuleName        = "integration/mcp"
	A2AModuleName        = "integration/a2a"
	A2ACardSchemaVersion = "v1.0.0"
	OASFMajorVersion     = 1
	DefaultSchemaVersion = "1.0.0"
)

// TranslatorOption is a function that configures translator options.
type TranslatorOption func(*translatorOptions)

// translatorOptions holds the options for translation operations.
type translatorOptions struct {
	version string
}

// WithVersion sets the schema version to use for translation.
// If not provided, the default version "1.0.0" will be used.
func WithVersion(version string) TranslatorOption {
	return func(opts *translatorOptions) {
		opts.version = version
	}
}

// validateMajorVersion checks if the version string has the supported major version or not.
func validateMajorVersion(versionStr string) error {
	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return fmt.Errorf("failed to parse schema version %s: %w", versionStr, err)
	}

	major := version.Major()
	if major != OASFMajorVersion {
		return fmt.Errorf("unsupported schema version: %s (major version %d not supported, only version 1.x.x is supported for record generation)", versionStr, major)
	}

	return nil
}
