// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"google.golang.org/protobuf/types/known/structpb"
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

// buildArtifactDescriptor creates an OCI-like artifact descriptor struct from raw bytes.
// The payload is base64-encoded into the `data` field; `size` and `digest` are derived
// from the raw bytes so they remain verifiable independently of any serialization choice.
func buildArtifactDescriptor(payload []byte, mediaType string) *structpb.Struct {
	encoded := base64.StdEncoding.EncodeToString(payload)
	sum := sha256.Sum256(payload)
	digest := fmt.Sprintf("sha256:%x", sum)

	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"media_type": {Kind: &structpb.Value_StringValue{StringValue: mediaType}},
			"size":       {Kind: &structpb.Value_NumberValue{NumberValue: float64(len(payload))}},
			"digest":     {Kind: &structpb.Value_StringValue{StringValue: digest}},
			"data":       {Kind: &structpb.Value_StringValue{StringValue: encoded}},
		},
	}
}

// artifactDataBytes returns the raw bytes stored in module.artifact.data (base64-decoded).
// Returns nil if the artifact field is absent, empty, or the base64 encoding is invalid.
func artifactDataBytes(moduleStruct *structpb.Struct) []byte {
	encoded := moduleStruct.GetFields()["artifact"].GetStructValue().GetFields()["data"].GetStringValue()
	if encoded == "" {
		return nil
	}

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil
	}

	return decoded
}

// structFromArtifactData decodes the base64-encoded JSON stored in module.artifact.data
// and returns it as a structpb.Struct. Returns nil if the artifact is absent or invalid.
func structFromArtifactData(moduleStruct *structpb.Struct) *structpb.Struct {
	raw := artifactDataBytes(moduleStruct)
	if raw == nil {
		return nil
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}

	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}

	return s
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
