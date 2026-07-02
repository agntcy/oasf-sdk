// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

// Package extractor maps free-form text onto the OASF taxonomy of skills,
// domains, and modules (https://schema.oasf.outshift.com), and extracts a few
// free-text keywords from the input.
//
// Given an arbitrary input string - a short user search query or the full
// contents of an agent's SKILL.md - it returns the most relevant OASF skills
// and domains (each with its numeric id and hierarchical name). By default it
// searches every supported OASF schema version at once - classes shared across
// versions are merged and report which versions contain them - and a single
// query can be restricted to specific versions with the Versions option.
//
// Modules (structural capabilities/protocols such as mcp or a2a) come from a
// small hand-maintained list (see module.go) and are matched purely by literal
// mention: a module is returned only when the input names it, and the module
// result is empty otherwise.
//
// It also returns up to five free-text keywords (input terms ranked by
// frequency, minus common stop/kind words) for searching record titles and
// descriptions beyond the taxonomy. First iteration: keywords are not
// de-duplicated against matched skills/domains/modules.
//
// Two complementary matching strategies are combined:
//
//   - Lexical (D): keyword/substring overlap against class names and captions.
//     Deterministic, exact, great for identifiers the user types verbatim.
//   - Semantic (C): cosine similarity in an embedding space, so paraphrases and
//     synonyms match even without shared keywords. This is powered by a real,
//     in-process sentence-transformer (all-MiniLM-L6-v2) run via pure Go
//     (cybertron/spago) — no CGo, no Python, no external service.
//
// Before the package can be used, assets must be provisioned once by calling
// Provision, which downloads and converts the embedding model from HuggingFace
// and embeds taxonomy labels fetched from the configured OASF endpoint. Assets
// are stored in a local directory (default ~/.agntcy/oasf-sdk/extractor/). The
// OASF taxonomy is not embedded in the binary; Provision fetches it from the
// configured endpoint (available versions come from the server). New then loads
// the provisioned assets entirely from that local directory and performs no
// network I/O; it errors if WithOASFURL is not provided, if the URL differs from
// the one the assets were provisioned for, or if assets have not been
// provisioned. Re-run Provision to pick up an OASF instance that changed.
//
// Typical usage:
//
//	// One-time setup (downloads model + fetches taxonomy labels):
//	err := extractor.Provision(ctx, extractor.WithOASFURL("https://schema.oasf.outshift.com"))
//
//	// Repeated use:
//	e, err := extractor.New(extractor.WithOASFURL("https://schema.oasf.outshift.com"))
//	res, err := e.Extract(ctx, text)
package extractor
