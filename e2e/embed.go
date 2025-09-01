// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

//go:embed fixtures/valid_v0.5.0_record.json
var validV050Record []byte

//go:embed fixtures/valid_v0.6.0_record.json
var validV060Record []byte

//go:embed fixtures/invalid_v0.6.0_record.json
var invalidV060Record []byte

//go:embed fixtures/translation_record.json
var translationRecord []byte
