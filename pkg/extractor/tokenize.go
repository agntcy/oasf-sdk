// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"strings"
	"unicode"
)

// tokenize lowercases the input and splits it into alphanumeric word tokens.
func tokenize(s string) []string {
	s = strings.ToLower(s)

	var (
		toks []string
		b    strings.Builder
	)

	flush := func() {
		if b.Len() > 0 {
			toks = append(toks, b.String())
			b.Reset()
		}
	}

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			flush()
		}
	}

	flush()

	return toks
}

// humanize turns a hierarchical OASF name such as
// "natural_language_processing/text_completion" into the plain phrase
// "natural language processing text completion" so it can be tokenized and
// embedded alongside captions and descriptions.
func humanize(name string) string {
	return strings.NewReplacer("/", " ", "_", " ", "-", " ").Replace(name)
}

// stopwords are common English words and OASF boilerplate that carry little
// discriminative signal for lexical matching.
var stopwords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "or": {}, "the": {}, "to": {}, "of": {}, "for": {},
	"in": {}, "on": {}, "with": {}, "by": {}, "is": {}, "are": {}, "be": {}, "as": {},
	"that": {}, "this": {}, "it": {}, "its": {}, "from": {}, "at": {}, "into": {},
	"capability": {}, "capabilities": {}, "ability": {}, "abilities": {}, "skill": {},
	"skills": {}, "domain": {}, "domains": {}, "agent": {}, "agents": {}, "using": {},
	"use": {}, "used": {}, "such": {}, "etc": {}, "based": {}, "can": {}, "data": {},
}

// tokenSet returns the unique, non-stopword tokens of s.
func tokenSet(s string) map[string]struct{} {
	set := make(map[string]struct{})

	for _, t := range tokenize(s) {
		if _, skip := stopwords[t]; skip {
			continue
		}

		set[t] = struct{}{}
	}

	return set
}
