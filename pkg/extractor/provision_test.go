// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import "testing"

func TestNormalizeModelName(t *testing.T) {
	cases := map[string]string{
		"all-MiniLM-L6-v2":                       "sentence-transformers/all-MiniLM-L6-v2",
		"sentence-transformers/all-MiniLM-L6-v2": "sentence-transformers/all-MiniLM-L6-v2",
		"org/custom-bert":                        "org/custom-bert",
	}
	for in, want := range cases {
		if got := normalizeModelName(in); got != want {
			t.Errorf("normalizeModelName(%q) = %q, want %q", in, want, got)
		}
	}
}
