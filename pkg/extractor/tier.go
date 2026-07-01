// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

// defaultTierRatio is the relative-drop threshold separating score tiers: a
// result starts a new tier when its score is less than this fraction of the
// score directly above it. The ratio is scale-invariant, so it behaves the same
// for strong and weak queries. Override with WithTierRatio.
const defaultTierRatio = 0.97

// assignTiers sets Tier (1-based) on each element of a slice already sorted by
// descending Score. A new tier starts when score[i]/score[i-1] < ratio, or when
// the preceding score is <= 0 (the ratio is undefined there, so force a break).
func assignTiers(sorted []ScoredClass, ratio float64) {
	tier := 0

	var prev float64

	for i := range sorted {
		switch {
		case i == 0:
			tier = 1
		case prev <= 0 || sorted[i].Score/prev < ratio:
			tier++
		}

		sorted[i].Tier = tier
		prev = sorted[i].Score
	}
}

// selectByTiers assigns tier numbers to the score-sorted list, then keeps the
// results in the first cfg.tiers tiers that also clear cfg.minScore. If that
// leaves fewer than cfg.minResults, it appends the next-highest-scored results
// until the floor is met (mirroring the MinResults guarantee). It mutates
// sorted (to record tier numbers). sorted must be in descending Score order.
func selectByTiers(sorted []ScoredClass, cfg queryConfig) []ScoredClass {
	assignTiers(sorted, cfg.tierRatio)

	out := make([]ScoredClass, 0, len(sorted))

	for i := range sorted {
		if sorted[i].Tier <= cfg.tiers && sorted[i].Score >= cfg.minScore {
			out = append(out, sorted[i])
		}
	}

	// The kept set is a contiguous prefix of sorted: tiers are non-decreasing
	// with index, and (scores being descending) minScore only ever drops a
	// suffix of the selected tiers. So len(out) is the next index to consider
	// for the MinResults floor.
	for i := len(out); i < len(sorted) && len(out) < cfg.minResults; i++ {
		out = append(out, sorted[i])
	}

	return out
}
