// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import "strings"

// verbatimCaptionExistingWeight is the weight applied to the existing token
// overlap score when a verbatim caption match is detected: the result blends
// the token overlap with a guaranteed 0.5 floor, ensuring verbatim caption
// matches always score above 0.5.
const (
	verbatimCaptionExistingWeight = 0.5
	verbatimCaptionBaseBonus      = 0.5
)

// lexicalScore (the "D" strategy) measures keyword overlap between the query
// and a class. It returns a value in [0,1].
//
// The base score is the fraction of the class's distinctive tokens (drawn from
// its hierarchical name and caption) that appear anywhere in the query. A
// phrase bonus is added when the class caption appears verbatim as a substring
// of the query, which strongly rewards users who type a label name directly.
func lexicalScore(queryLower string, queryTokens map[string]struct{}, ic *indexedClass) float64 {
	if len(ic.lexTokens) == 0 {
		return 0
	}

	matched := 0

	for tok := range ic.lexTokens {
		if _, ok := queryTokens[tok]; ok {
			matched++
		}
	}

	score := float64(matched) / float64(len(ic.lexTokens))

	// Verbatim caption match is a very strong signal.
	if len(ic.captionLower) > 3 && strings.Contains(queryLower, ic.captionLower) {
		score = score*verbatimCaptionExistingWeight + verbatimCaptionBaseBonus
	}

	if score > 1 {
		score = 1
	}

	return score
}
