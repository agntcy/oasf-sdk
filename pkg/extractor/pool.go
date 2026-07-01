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
// it equals that chunk's similarity, so short queries are unaffected.
func poolScores(sims []float64) float64 {
	if len(sims) == 0 {
		return 0
	}

	n := min(max(semanticPoolTopN, 1), len(sims))

	cp := make([]float64, len(sims))
	copy(cp, sims)
	sort.Sort(sort.Reverse(sort.Float64Slice(cp)))

	var sum float64
	for i := range n {
		sum += cp[i]
	}

	return sum / float64(n)
}
