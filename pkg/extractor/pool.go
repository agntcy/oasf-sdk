// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import "sort"

// semanticPoolTopN is how many of a label's best per-chunk similarities are
// averaged into its semantic score. Pure max-pooling (n=1) let a single
// tangential chunk of a long input dominate; averaging the top few rewards
// labels that match broadly while still surfacing a strong localized hit.
const semanticPoolTopN = 3

// poolScores reduces a label's per-chunk similarities to one score: the mean of
// the semanticPoolTopN largest values (clamped to [1,len]). For a single chunk
// it equals that chunk's similarity, so short queries are unaffected. It sorts
// sims in place; the caller passes throwaway per-label scratch, so no copy is
// made.
func poolScores(sims []float64) float64 {
	if len(sims) == 0 {
		return 0
	}

	n := min(max(semanticPoolTopN, 1), len(sims))

	sort.Sort(sort.Reverse(sort.Float64Slice(sims)))

	var sum float64
	for _, v := range sims[:n] {
		sum += v
	}

	return sum / float64(n)
}
