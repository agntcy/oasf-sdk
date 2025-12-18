// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

//go:embed fixtures/valid_0.8.0_record.json
var validV080Record []byte

//go:embed fixtures/invalid_0.8.0_record.json
var invalidV080Record []byte

//go:embed fixtures/valid_0.7.0_record.json
var validV070Record []byte

//go:embed fixtures/invalid_0.7.0_record.json
var invalidV070Record []byte

//go:embed fixtures/valid_v0.3.1_record.json
var validV031Record []byte

//go:embed fixtures/valid_0.3.1_record.json
var valid031Record []byte

//go:embed fixtures/invalid_v0.3.1_record.json
var invalidV031Record []byte

//go:embed fixtures/translation_0.7.0_record.json
var translationV070Record []byte

//go:embed fixtures/translation_0.8.0_record.json
var translationV080Record []byte

//go:embed fixtures/expected_v0.3.1_decoded.json
var expectedV031Decoded []byte

//go:embed fixtures/expected_0.8.0_decoded.json
var expectedV080Decoded []byte

//go:embed fixtures/expected_0.7.0_decoded.json
var expectedV070Decoded []byte

//go:embed fixtures/expected_gh_copilot_output.json
var expectedGHCopilotOutput []byte

//go:embed fixtures/expected_a2a_output.json
var expectedA2AOutput []byte

//go:embed fixtures/expected_a2atorecord_output.json
var expectedA2AToRecordOutput []byte

//go:embed fixtures/translation_mcp.json
var translationMCPRegistry []byte

//go:embed fixtures/expected_mcptorecord_output.json
var expectedMCPToRecordOutput []byte

//go:embed fixtures/weather_service_0.8.0_record.json
var weatherServiceRecord []byte

//go:embed fixtures/weather_service_output.yaml
var weatherServiceOutputYAML []byte

//go:embed fixtures/translation_mcp_minimal_local.json
var translationMCPMinimalLocal []byte

//go:embed fixtures/expected_mcp_minimal_local_output.json
var expectedMCPMinimalLocalOutput []byte

//go:embed fixtures/translation_mcp_http_headers.json
var translationMCPHTTPHeaders []byte

//go:embed fixtures/expected_mcp_http_headers_output.json
var expectedMCPHTTPHeadersOutput []byte

//go:embed fixtures/translation_mcp_sse_minimal.json
var translationMCPSSEMinimal []byte

//go:embed fixtures/expected_mcp_sse_minimal_output.json
var expectedMCPSSEMinimalOutput []byte
