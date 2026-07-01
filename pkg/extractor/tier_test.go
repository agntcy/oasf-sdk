// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"slices"
	"testing"
)

func mkScored(scores ...float64) []ScoredClass {
	out := make([]ScoredClass, len(scores))
	for i, s := range scores {
		out[i] = ScoredClass{Score: s}
	}

	return out
}

func tiersOf(scs []ScoredClass) []int {
	out := make([]int, len(scs))
	for i := range scs {
		out[i] = scs[i].Tier
	}

	return out
}

func scoresOf(scs []ScoredClass) []float64 {
	out := make([]float64, len(scs))
	for i := range scs {
		out[i] = scs[i].Score
	}

	return out
}

func TestAssignTiersWorkedExample(t *testing.T) {
	// The exact "Code review" example from the design doc, at ratio 0.97.
	skills := mkScored(0.376, 0.349, 0.345, 0.326, 0.248)
	assignTiers(skills, defaultTierRatio)

	if got, want := tiersOf(skills), []int{1, 2, 2, 3, 4}; !slices.Equal(got, want) {
		t.Errorf("skills tiers = %v, want %v", got, want)
	}

	domains := mkScored(0.231, 0.175, 0.174, 0.159, 0.159)
	assignTiers(domains, defaultTierRatio)

	if got, want := tiersOf(domains), []int{1, 2, 2, 3, 3}; !slices.Equal(got, want) {
		t.Errorf("domains tiers = %v, want %v", got, want)
	}
}

func TestAssignTiersZeroScoreBreaks(t *testing.T) {
	// A zero predecessor forces a break (the ratio is undefined).
	scs := mkScored(0.5, 0.0, 0.0)
	assignTiers(scs, defaultTierRatio)

	if got, want := tiersOf(scs), []int{1, 2, 3}; !slices.Equal(got, want) {
		t.Errorf("tiers = %v, want %v", got, want)
	}
}

func TestSelectByTiersFirstTier(t *testing.T) {
	cfg := queryConfig{tiers: 1, tierRatio: defaultTierRatio, minScore: 0, minResults: 1}

	out := selectByTiers(mkScored(0.376, 0.349, 0.345, 0.326, 0.248), cfg)
	if got, want := scoresOf(out), []float64{0.376}; !slices.Equal(got, want) {
		t.Errorf("tier 1 = %v, want %v", got, want)
	}
}

func TestSelectByTiersTwoTiers(t *testing.T) {
	cfg := queryConfig{tiers: 2, tierRatio: defaultTierRatio, minScore: 0, minResults: 1}

	out := selectByTiers(mkScored(0.376, 0.349, 0.345, 0.326, 0.248), cfg)
	if got, want := scoresOf(out), []float64{0.376, 0.349, 0.345}; !slices.Equal(got, want) {
		t.Errorf("tiers 1-2 = %v, want %v", got, want)
	}
}

func TestSelectByTiersMinScoreSplitsTier(t *testing.T) {
	// All three are tier 1 (ratios >= 0.97), but minScore drops the last.
	cfg := queryConfig{tiers: 1, tierRatio: defaultTierRatio, minScore: 0.485, minResults: 1}

	out := selectByTiers(mkScored(0.5, 0.49, 0.48), cfg)
	if got, want := scoresOf(out), []float64{0.5, 0.49}; !slices.Equal(got, want) {
		t.Errorf("min-score split = %v, want %v", got, want)
	}
}

func TestSelectByTiersMinResultsFloor(t *testing.T) {
	// Tier 1 (0.04) is below minScore, so nothing passes the filter; the
	// MinResults floor still yields the top result.
	cfg := queryConfig{tiers: 1, tierRatio: defaultTierRatio, minScore: 0.05, minResults: 1}

	out := selectByTiers(mkScored(0.04, 0.03), cfg)
	if got, want := scoresOf(out), []float64{0.04}; !slices.Equal(got, want) {
		t.Errorf("min-results floor = %v, want %v", got, want)
	}
}
