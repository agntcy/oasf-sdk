// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package e2e

import _ "embed"

//go:embed fixtures/valid_v0.7.0_record.json
var validV070Record []byte

//go:embed fixtures/invalid_v0.7.0_record.json
var invalidV070Record []byte

//go:embed fixtures/translation_record.json
var translationRecord []byte
