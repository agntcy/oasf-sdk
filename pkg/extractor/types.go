// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

// Kind distinguishes an OASF skill from an OASF domain.
type Kind string

const (
	KindSkill  Kind = "skill"
	KindDomain Kind = "domain"
	KindModule Kind = "module"
)

// Class is a single OASF skill, domain, or module class.
type Class struct {
	// ID is the OASF numeric identifier (uid), e.g. 10304.
	ID uint64 `json:"id"`
	// Name is the hierarchical OASF name used as a routing label,
	// e.g. "natural_language_processing/information_retrieval_synthesis/sentence_similarity".
	Name string `json:"name"`
	// Caption is the human-readable label, e.g. "Sentence Similarity".
	Caption string `json:"caption"`
	// Description is the OASF prose description of the class.
	Description string `json:"description"`
}

// ScoredClass is a Class together with the score that ranked it and the
// individual signals that produced that score.
type ScoredClass struct {
	Class

	// Kind is KindSkill, KindDomain, or KindModule.
	Kind Kind `json:"kind"`
	// Versions are the OASF schema versions whose catalog contains this exact
	// class (same hierarchical name), in ascending order. A class shared by
	// several versions is reported once with all of them listed here.
	Versions []string `json:"versions"`
	// Score is the combined relevance score in [0,1]; higher is better.
	Score float64 `json:"score"`
	// Semantic is the embedding cosine-similarity component in [0,1].
	Semantic float64 `json:"semantic"`
	// Lexical is the keyword-overlap component in [0,1].
	Lexical float64 `json:"lexical"`
	// Tier is the 1-based score group this result falls in; tier 1 is the
	// closest cluster of matches. Results within a tier have near-equal scores;
	// a new tier starts at a relative drop in score (see WithTierRatio).
	Tier int `json:"tier"`
}

// Result holds the recommended skills, domains, and modules, merged across the
// searched OASF versions (see the Versions query option). Each slice is sorted
// by descending Score. Skills and Domains are guaranteed to contain at least
// MinResults entries (default 1), even when the input is a poor match. Modules
// are the exception: they use a pure-lexical gate and are empty unless the input
// literally mentions a module (no floor).
type Result struct {
	Skills   []ScoredClass `json:"skills"`
	Domains  []ScoredClass `json:"domains"`
	Modules  []ScoredClass `json:"modules"`
	Keywords []Keyword     `json:"keywords"`
}
