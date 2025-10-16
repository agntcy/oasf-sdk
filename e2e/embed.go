// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

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

//go:embed fixtures/translation_record.json
var translationRecord []byte

//go:embed fixtures/expected_v0.3.1_decoded.json
var expectedV031Decoded []byte

//go:embed fixtures/expected_0.7.0_decoded.json
var expectedV070Decoded []byte

//go:embed fixtures/expected_gh_copilot_output.json
var expectedGHCopilotOutput []byte

//go:embed fixtures/expected_a2a_output.json
var expectedA2AOutput []byte

//go:embed fixtures/expected_a2atorecord_output.json
var expectedA2AToRecordOutput []byte
