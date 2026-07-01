// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"slices"
	"testing"
)

func keywordTexts(ks []Keyword) []string {
	out := make([]string, len(ks))
	for i, k := range ks {
		out[i] = k.Text
	}

	return out
}

func TestExtractKeywordsBasic(t *testing.T) {
	got := keywordTexts(extractKeywords("github review"))
	if !slices.Equal(got, []string{"github", "review"}) {
		t.Errorf("got %v, want [github review]", got)
	}
}

func TestExtractKeywordsDropsFiller(t *testing.T) {
	// Filler ("I want all the items that can do") and kind words must be removed.
	got := keywordTexts(extractKeywords("I want all the items that can do github review"))
	if !slices.Equal(got, []string{"github", "review"}) {
		t.Errorf("filler not removed: got %v, want [github review]", got)
	}

	if len(extractKeywords("skill domain module")) != 0 {
		t.Errorf("kind words should not become keywords")
	}
}

func TestExtractKeywordsFrequencyRanks(t *testing.T) {
	// A repeated term outranks a single occurrence; score is the count.
	ks := extractKeywords("review github review")
	if len(ks) != 2 || ks[0].Text != "review" || ks[0].Score != 2 {
		t.Fatalf("frequency ranking wrong: %+v", ks)
	}

	if ks[1].Text != "github" || ks[1].Score != 1 {
		t.Errorf("second keyword wrong: %+v", ks[1])
	}
}

func TestExtractKeywordsCapsAtFive(t *testing.T) {
	ks := extractKeywords("alpha bravo charlie delta echo foxtrot golf")
	if len(ks) != 5 {
		t.Fatalf("expected 5 keywords (cap), got %d", len(ks))
	}
	// First-appearance order is preserved among equal (frequency-1) scores.
	if !slices.Equal(keywordTexts(ks), []string{"alpha", "bravo", "charlie", "delta", "echo"}) {
		t.Errorf("cap/order wrong: %v", keywordTexts(ks))
	}
}

func TestExtractReturnsKeywords(t *testing.T) {
	r := newTestExtractor(t)

	res, err := r.Extract(context.Background(), "github review", Latest())
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(keywordTexts(res.Keywords), []string{"github", "review"}) {
		t.Errorf("Result.Keywords = %v, want [github review]", keywordTexts(res.Keywords))
	}
}
