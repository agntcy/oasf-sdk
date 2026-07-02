// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import "sort"

// Keyword is a free-text term extracted from the input, for searching record
// titles/descriptions beyond the OASF taxonomy. Score is the term's frequency
// in the input.
type Keyword struct {
	Text  string  `json:"text"`
	Score float64 `json:"score"`
}

// maxKeywords caps how many keywords Extract returns.
const maxKeywords = 5

// keywordStopwords are words with no value as free-text search keywords: common
// English function/filler words plus OASF boilerplate and the kind words
// ("skill"/"domain"/"module"). It is intentionally separate from (and larger
// than) the catalog `stopwords` used for lexical class matching, so broadening
// it never affects skill/domain scoring.
var keywordStopwords = map[string]struct{}{
	// articles, conjunctions, prepositions
	"a": {}, "an": {}, "and": {}, "or": {}, "the": {}, "to": {}, "of": {}, "for": {},
	"in": {}, "on": {}, "with": {}, "by": {}, "as": {}, "from": {}, "at": {}, "into": {},
	"about": {}, "than": {}, "then": {}, "so": {}, "if": {}, "but": {}, "not": {}, "no": {},
	"over": {}, "under": {}, "out": {}, "up": {}, "down": {},
	// pronouns / determiners
	"i": {}, "you": {}, "we": {}, "they": {}, "he": {}, "she": {}, "it": {}, "its": {},
	"me": {}, "my": {}, "our": {}, "your": {}, "their": {}, "them": {}, "this": {},
	"that": {}, "these": {}, "those": {}, "all": {}, "any": {}, "some": {}, "each": {},
	"every": {}, "more": {}, "most": {}, "only": {}, "also": {}, "such": {}, "there": {},
	"here": {}, "what": {}, "which": {}, "who": {}, "when": {}, "where": {}, "how": {},
	// auxiliary / common verbs and filler
	"is": {}, "are": {}, "be": {}, "was": {}, "were": {}, "has": {}, "have": {}, "had": {},
	"do": {}, "does": {}, "did": {}, "done": {}, "doing": {}, "can": {}, "could": {},
	"will": {}, "would": {}, "should": {}, "want": {}, "wants": {}, "wanted": {},
	"need": {}, "needs": {}, "find": {}, "finds": {}, "search": {}, "show": {},
	"list": {}, "get": {}, "give": {}, "like": {}, "using": {}, "use": {}, "used": {},
	"based": {}, "please": {}, "related": {}, "etc": {},
	// nouns that carry no search signal here
	"item": {}, "items": {}, "thing": {}, "things": {}, "data": {},
	// OASF boilerplate + kind words
	"capability": {}, "capabilities": {}, "ability": {}, "abilities": {},
	"skill": {}, "skills": {}, "domain": {}, "domains": {}, "module": {}, "modules": {},
	"agent": {}, "agents": {},
}

// extractKeywords returns up to maxKeywords content terms from text, ranked by
// term frequency (ties broken by first appearance). First iteration: no
// de-duplication against matched skills/domains/modules.
func extractKeywords(text string) []Keyword {
	freq := make(map[string]int)
	order := make([]string, 0)

	for _, t := range tokenize(text) {
		if _, skip := keywordStopwords[t]; skip {
			continue
		}

		if _, seen := freq[t]; !seen {
			order = append(order, t)
		}

		freq[t]++
	}

	kws := make([]Keyword, 0, len(order))
	for _, t := range order {
		kws = append(kws, Keyword{Text: t, Score: float64(freq[t])})
	}

	// Stable sort by score descending keeps first-appearance order for ties.
	sort.SliceStable(kws, func(i, j int) bool {
		return kws[i].Score > kws[j].Score
	})

	if len(kws) > maxKeywords {
		kws = kws[:maxKeywords]
	}

	return kws
}
