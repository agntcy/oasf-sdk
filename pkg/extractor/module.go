// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import "strings"

// moduleIdentity is one OASF class identity of a curated module within a set of
// versions. A module can have several identities because it gets renamed across
// versions (e.g. mcp is runtime/mcp in 0.7.0 but integration/mcp from 0.8.0).
type moduleIdentity struct {
	ID          uint64
	Name        string
	Caption     string
	Description string
	Versions    []string // OASF versions carrying this exact id/name, ascending
}

// curatedModule is a manually-maintained OASF module the extractor surfaces.
// Unlike skills and domains, modules are NOT fetched or embedded: they are
// matched purely by literal mention. A module is returned when any of its
// `match` strings appears in the query - a single-word match must appear as a
// whole token, a multi-word match as a substring. Each in-scope identity is
// returned as its own result (with its versions), matching how renamed skills/
// domains surface as distinct entries. Keep this list small; extend it by hand.
type curatedModule struct {
	match      []string
	identities []moduleIdentity
}

// curatedModules is the hand-maintained module catalog. Identities mirror the
// OASF schema (id, hierarchical name, caption, description) per version, newest
// first.
var curatedModules = []curatedModule{
	{
		match: []string{"mcp"},
		identities: []moduleIdentity{
			{
				ID: 202, Name: "integration/mcp", Caption: "MCP", //nolint:mnd // OASF schema class ID
				Description: "Describes MCP servers required to run and interact with the agent.",
				Versions:    []string{"0.8.0", "1.0.0", "1.1.0"},
			},
			{
				ID: 302, Name: "runtime/mcp", Caption: "MCP", //nolint:mnd // OASF schema class ID
				Description: "Describes MCP servers required to run and interact with the agent.",
				Versions:    []string{"0.7.0"},
			},
		},
	},
	{
		match: []string{"a2a"},
		identities: []moduleIdentity{
			{
				ID: 203, Name: "integration/a2a", Caption: "A2A", //nolint:mnd // OASF schema class ID
				Description: "Describes A2A card details for communication and usage with A2A protocol.",
				Versions:    []string{"0.8.0", "1.0.0", "1.1.0"},
			},
			{
				ID: 305, Name: "runtime/a2a", Caption: "A2A", //nolint:mnd // OASF schema class ID
				Description: "Describes A2A card details for communication and usage with A2A protocol.",
				Versions:    []string{"0.7.0"},
			},
		},
	},
	{
		// "agent skill" (substring) also covers "agent skills"; "agentskills"
		// covers the single-token spelling. Bare "skill"/"skills" map here too:
		// a query mentioning "skill" means the agent-skills record type.
		match: []string{"agentskills", "agent skill", "skill", "skills"},
		identities: []moduleIdentity{
			{
				ID: 10302, Name: "core/language_model/agentskills", Caption: "Language Model Agent Skills", //nolint:mnd // OASF schema class ID
				Description: "Describes Agent Skills associated with the language model module.",
				Versions:    []string{"1.0.0", "1.1.0"},
			},
		},
	},
}

// mentioned reports whether the query literally names this module: a multi-word
// match string must appear as a substring of the (lowercased) query, while a
// single-word match must appear as a whole token of the raw (unfiltered) query.
// Modules deliberately match raw tokens, not the stopword-filtered set, so
// stopwords such as "skill" can still name a module.
func (m curatedModule) mentioned(queryLower string, queryTokens map[string]struct{}) bool {
	for _, s := range m.match {
		if strings.Contains(s, " ") {
			if strings.Contains(queryLower, s) {
				return true
			}
		} else if _, ok := queryTokens[s]; ok {
			return true
		}
	}

	return false
}

// scoreModules returns the curated modules the query literally mentions. Each
// identity in version scope is one result, carrying the in-scope subset of its
// versions; identities with no in-scope version are dropped. Results follow the
// curated order (modules in list order, identities newest-first). When nothing
// matches it returns an empty slice: modules have no MinResults floor and no
// all-modules fallback, and they ignore the semantic signal and the tier/
// min-score selection used for skills and domains.
func scoreModules(queryLower string, queryTokens map[string]struct{}, want map[string]struct{}) []ScoredClass {
	out := make([]ScoredClass, 0)

	for _, m := range curatedModules {
		if !m.mentioned(queryLower, queryTokens) {
			continue
		}

		for _, id := range m.identities {
			versions := inScopeVersions(id.Versions, want)
			if len(versions) == 0 {
				continue
			}

			out = append(out, ScoredClass{
				Class:    Class{ID: id.ID, Name: id.Name, Caption: id.Caption, Description: id.Description},
				Kind:     KindModule,
				Versions: versions,
				Score:    1,
				Lexical:  1,
			})
		}
	}

	return out
}

// inScopeVersions returns the subset of have that is in want, preserving have's
// order. An empty want (no scope restriction) returns have unchanged.
func inScopeVersions(have []string, want map[string]struct{}) []string {
	if len(want) == 0 {
		return have
	}

	out := make([]string, 0, len(have))
	for _, v := range have {
		if _, ok := want[v]; ok {
			out = append(out, v)
		}
	}

	return out
}
