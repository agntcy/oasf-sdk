// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/agntcy/oasf-sdk/pkg/schema"
)

// indexedClass is a merged Class enriched with the data needed to score it.
type indexedClass struct {
	Class

	versions     []string            // OASF versions whose catalog contains this name
	vec          []float32           // semantic embedding (unit length)
	lexTokens    map[string]struct{} // distinctive tokens for lexical matching
	captionLower string
}

// Extractor maps text onto the OASF skills and domains across all supported
// schema versions. Classes shared by several versions are merged into a single
// entry (keyed by hierarchical name) that records every version it appears in.
// Build one with New and reuse it; it is safe for concurrent use.
type Extractor struct {
	embedder Embedder

	// versions are the OASF schema versions this extractor covers, ascending.
	versions []string

	skills  []indexedClass
	domains []indexedClass

	// scoring defaults (overridable per query)
	// Blend weights for skills (also the package default).
	wSemantic float64
	wLexical  float64
	// Blend weights for domains. Domains benefit from a higher semantic weight:
	// the lexical signal adds spurious overlap (generic words like "data" or
	// "systems" match unrelated domains), measurably hurting domain ranking.
	wSemanticDomain float64
	wLexicalDomain  float64
	tiers           int     // number of score tiers to return per kind (default 1)
	tierRatio       float64 // a result starts a new tier when score/prev < tierRatio
	minScore        float64
	overlap         int
}

// options holds resolved constructor configuration shared by New and Provision.
type options struct {
	oasfURL   string
	modelName string
	assetDir  string
	embedder  Embedder // test injection; overrides the on-disk model

	// scoring defaults (copied onto the Extractor by New/newFromClasses).
	wSemantic       float64
	wLexical        float64
	wSemanticDomain float64
	wLexicalDomain  float64
	tiers           int
	tierRatio       float64
	minScore        float64
}

// Default scoring weights for skills.
const (
	defaultWSemanticSkill  = 0.6
	defaultWLexicalSkill   = 0.4
	defaultWSemanticDomain = 0.8
	defaultWLexicalDomain  = 0.2
	defaultMinScore        = 0.05
	defaultChunkOverlap    = 32
)

// defaultOptions returns the baseline configuration before options are applied.
func defaultOptions() options {
	return options{
		modelName:       defaultModelName,
		assetDir:        defaultAssetDir(),
		wSemantic:       defaultWSemanticSkill,
		wLexical:        defaultWLexicalSkill,
		wSemanticDomain: defaultWSemanticDomain,
		wLexicalDomain:  defaultWLexicalDomain,
		tiers:           1,
		tierRatio:       defaultTierRatio,
		minScore:        defaultMinScore,
	}
}

// defaultAssetDir returns the default asset directory
// (~/.agntcy/oasf-sdk/extractor), falling back to a temp dir if the home
// directory cannot be determined.
func defaultAssetDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}

	return filepath.Join(home, ".agntcy", "oasf-sdk", "extractor")
}

func applyOptions(o *options, opts []Option) {
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
}

// Option configures New and Provision at construction time.
type Option func(*options)

// WithOASFURL sets the OASF schema endpoint used to fetch the taxonomy. It is
// required by both New and Provision.
func WithOASFURL(url string) Option {
	return func(o *options) {
		o.oasfURL = url
	}
}

// WithModelName sets the sentence-transformer model to download+load. A bare
// name (no "/") is treated as a sentence-transformers model. Defaults to
// all-MiniLM-L6-v2.
func WithModelName(name string) Option {
	return func(o *options) {
		if name != "" {
			o.modelName = name
		}
	}
}

// WithAssetDir sets the directory holding the provisioned model, label vectors,
// and manifest. Defaults to ~/.agntcy/oasf-sdk/extractor.
func WithAssetDir(dir string) Option {
	return func(o *options) {
		if dir != "" {
			o.assetDir = dir
		}
	}
}

// WithEmbedder overrides the embedding backend. By default New loads the
// provisioned sentence-transformer from the asset dir. Supply this only to use
// a different model or, in tests, a lightweight fake.
func WithEmbedder(e Embedder) Option {
	return func(o *options) {
		if e != nil {
			o.embedder = e
		}
	}
}

// WithWeights sets the relative weight of the semantic and lexical signals in
// the skills' combined score (see WithDomainWeights for domains). They are
// normalized, so WithWeights(3,1) and WithWeights(0.75,0.25) are equivalent.
// Defaults to (0.6, 0.4).
func WithWeights(semantic, lexical float64) Option {
	return func(o *options) {
		if semantic >= 0 && lexical >= 0 && semantic+lexical > 0 {
			o.wSemantic = semantic
			o.wLexical = lexical
		}
	}
}

// WithDomainWeights sets the semantic/lexical blend used for DOMAINS only
// (skills use WithWeights). Like WithWeights the values are normalized. Domains
// default to (0.8, 0.2) because the lexical signal adds spurious cross-domain
// overlap that hurts ranking; raise lexical only if your domain captions overlap
// queries cleanly.
func WithDomainWeights(semantic, lexical float64) Option {
	return func(o *options) {
		if semantic >= 0 && lexical >= 0 && semantic+lexical > 0 {
			o.wSemanticDomain = semantic
			o.wLexicalDomain = lexical
		}
	}
}

// WithDefaultTiers sets the default number of score tiers returned per kind
// (default 1 — the closest group of matches). Use Tiers to override per query.
func WithDefaultTiers(n int) Option {
	return func(o *options) {
		if n >= 1 {
			o.tiers = n
		}
	}
}

// WithTierRatio sets the relative-drop threshold that separates score tiers: a
// result starts a new tier when its score is less than ratio times the score of
// the result directly above it. Must be in (0, 1]; default 0.97.
func WithTierRatio(ratio float64) Option {
	return func(o *options) {
		if ratio > 0 && ratio <= 1 {
			o.tierRatio = ratio
		}
	}
}

// WithDefaultMinScore sets the default score threshold below which results are
// dropped (subject to MinResults). Default 0.05.
func WithDefaultMinScore(s float64) Option {
	return func(o *options) {
		if s >= 0 {
			o.minScore = s
		}
	}
}

// newExtractor builds an Extractor shell with scoring params copied from o.
func newExtractor(o options) *Extractor {
	return &Extractor{
		embedder:        o.embedder,
		wSemantic:       o.wSemantic,
		wLexical:        o.wLexical,
		wSemanticDomain: o.wSemanticDomain,
		wLexicalDomain:  o.wLexicalDomain,
		tiers:           o.tiers,
		tierRatio:       o.tierRatio,
		minScore:        o.minScore,
		overlap:         defaultChunkOverlap,
	}
}

// New builds an Extractor from the assets a prior Provision wrote to the asset
// directory: it reads the manifest, loads the cached label vectors, loads the
// model (unless WithEmbedder overrides it), and reconstructs the in-memory
// index by fetching+merging the catalog for the manifest's versions. It errors
// if the OASF URL is empty or the assets are absent. Restrict an individual
// query to specific versions with the Versions query option.
func New(opts ...Option) (*Extractor, error) {
	o := defaultOptions()
	applyOptions(&o, opts)

	if o.oasfURL == "" {
		return nil, errors.New("OASF URL is required (use WithOASFURL)")
	}

	modelDir, labelsFile, manifestFile := assetPaths(o.assetDir)

	m, err := readManifest(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("assets not provisioned at %s; run Provision (init) first: %w", o.assetDir, err)
	}

	if m.FormatVersion != manifestFormatVersion {
		return nil, fmt.Errorf("asset format v%d != expected v%d; re-run Provision", m.FormatVersion, manifestFormatVersion)
	}

	skillVecs, domainVecs, err := readLabelVectors(labelsFile)
	if err != nil {
		return nil, fmt.Errorf("read label vectors: %w", err)
	}

	r := newExtractor(o)

	r.versions = append([]string(nil), m.TaxonomyVersions...)

	if r.embedder == nil {
		emb, err := newTransformerEmbedder(modelDir, m.ModelName)
		if err != nil {
			return nil, fmt.Errorf("load model: %w", err)
		}

		if emb.ID() != m.ModelID {
			return nil, fmt.Errorf("embedder model %q does not match provisioned model %q; re-run Provision", emb.ID(), m.ModelID)
		}

		r.embedder = emb
	}

	sc, err := schema.New(o.oasfURL, schema.WithCache(true))
	if err != nil {
		return nil, fmt.Errorf("schema client: %w", err)
	}

	ctx := context.Background()

	if r.skills, err = indexFromCache(ctx, sc, m.TaxonomyVersions, KindSkill, skillVecs, m); err != nil {
		return nil, fmt.Errorf("index skills: %w", err)
	}

	if r.domains, err = indexFromCache(ctx, sc, m.TaxonomyVersions, KindDomain, domainVecs, m); err != nil {
		return nil, fmt.Errorf("index domains: %w", err)
	}

	return r, nil
}

// Provision creates the on-disk assets (model, label vectors, manifest) under
// the asset dir so New can start without re-embedding the catalog. cybertron
// downloads+converts the model into <assetDir>/models if missing; the taxonomy
// is fetched from the OASF endpoint and embedded. Re-provisioning overwrites the
// existing assets.
func Provision(ctx context.Context, opts ...Option) error {
	o := defaultOptions()
	applyOptions(&o, opts)

	if o.oasfURL == "" {
		return errors.New("OASF URL is required (use WithOASFURL)")
	}

	modelDir, labelsFile, manifestFile := assetPaths(o.assetDir)

	emb := o.embedder
	if emb == nil {
		te, err := newTransformerEmbedder(modelDir, o.modelName)
		if err != nil {
			return fmt.Errorf("provision model: %w", err)
		}

		emb = te
	}

	sc, err := schema.New(o.oasfURL, schema.WithCache(true))
	if err != nil {
		return fmt.Errorf("schema client: %w", err)
	}

	versions, err := sc.GetAvailableSchemaVersions(ctx)
	if err != nil {
		return fmt.Errorf("list versions: %w", err)
	}

	skillVecs, skillDigest, err := embedKind(ctx, sc, emb, versions, KindSkill)
	if err != nil {
		return err
	}

	domainVecs, domainDigest, err := embedKind(ctx, sc, emb, versions, KindDomain)
	if err != nil {
		return err
	}

	if err := writeLabelVectors(labelsFile, skillVecs, domainVecs); err != nil {
		return err
	}

	return writeManifest(manifestFile, manifest{
		FormatVersion:    manifestFormatVersion,
		ModelName:        o.modelName,
		ModelID:          emb.ID(),
		OASFURL:          o.oasfURL,
		TaxonomyVersions: versions,
		SkillDigest:      skillDigest,
		DomainDigest:     domainDigest,
	})
}

// mergeFetchedClasses fetches the given kind for each version and merges classes
// that share a hierarchical name (versions are processed in the given order, so
// pass them ascending — the newest processed last wins for the class data; all
// versions the class appears in are recorded). The result is in a deterministic
// id/name order for reproducible embedding and tie-breaks.
func mergeFetchedClasses(ctx context.Context, sc *schema.Schema, versions []string, kind Kind) ([]indexedClass, error) {
	byName := make(map[string]*indexedClass)

	for _, v := range versions {
		classes, err := fetchClasses(ctx, sc, v, kind)
		if err != nil {
			return nil, err
		}

		for _, c := range classes {
			ic, ok := byName[c.Name]
			if !ok {
				ic = &indexedClass{}
				byName[c.Name] = ic
			}

			ic.Class = c
			ic.versions = append(ic.versions, v)
		}
	}

	list := make([]indexedClass, 0, len(byName))
	for _, ic := range byName {
		list = append(list, *ic)
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].ID != list[j].ID {
			return list[i].ID < list[j].ID
		}

		return list[i].Name < list[j].Name
	})

	return list, nil
}

// attachLexical fills in the lexical fields (lexTokens, captionLower) for each
// class in place.
func attachLexical(list []indexedClass) {
	for i := range list {
		list[i].lexTokens = tokenSet(humanize(list[i].Name) + " " + list[i].Caption)
		list[i].captionLower = strings.ToLower(list[i].Caption)
	}
}

// embedKind fetches+merges the given kind across versions, embeds the merged
// label texts once, and returns the vectors plus the catalog digest. Used by
// Provision.
func embedKind(ctx context.Context, sc *schema.Schema, emb Embedder, versions []string, kind Kind) ([][]float32, string, error) {
	list, err := mergeFetchedClasses(ctx, sc, versions, kind)
	if err != nil {
		return nil, "", err
	}

	texts := make([]string, len(list))
	for i := range list {
		texts[i] = labelText(list[i].Class)
	}

	vecs, err := emb.Embed(ctx, texts, RoleDocument)
	if err != nil {
		return nil, "", fmt.Errorf("embed catalog: %w", err)
	}

	return vecs, catalogDigest(texts), nil
}

// indexFromCache fetches+merges the given kind across versions and attaches the
// cached vectors (in the same deterministic order Provision embedded them),
// building the lexical fields. It validates the count and content digest of the
// fetched classes against the stored manifest so that a changed taxonomy is
// detected immediately rather than silently corrupting ranking. Used by New.
func indexFromCache(ctx context.Context, sc *schema.Schema, versions []string, kind Kind, vecs [][]float32, m manifest) ([]indexedClass, error) {
	list, err := mergeFetchedClasses(ctx, sc, versions, kind)
	if err != nil {
		return nil, err
	}

	if len(vecs) != len(list) {
		return nil, fmt.Errorf("cached %d vectors for %d %s classes; re-run Provision", len(vecs), len(list), kind)
	}

	texts := make([]string, len(list))
	for i := range list {
		texts[i] = labelText(list[i].Class)
	}

	if err := checkLabelDigest(kind, texts, m); err != nil {
		return nil, err
	}

	for i := range list {
		list[i].vec = vecs[i]
	}

	attachLexical(list)

	return list, nil
}

// newFromClasses builds an Extractor directly from in-memory taxonomy fixtures,
// embedding the label texts with the supplied embedder. It bypasses the asset
// directory and the network, so tests can exercise the scoring/indexing logic
// against a small fixture. versionsAscending must be the versions the fixtures
// span, ascending. skills and domains are the pre-merged indexedClass sets
// (with their versions populated).
func newFromClasses(versionsAscending []string, skills, domains []indexedClass, emb Embedder, opts ...Option) (*Extractor, error) {
	o := defaultOptions()
	applyOptions(&o, opts)

	r := newExtractor(o)

	r.versions = append([]string(nil), versionsAscending...)
	r.embedder = emb

	var err error
	if r.skills, err = embedFixture(skills, emb); err != nil {
		return nil, err
	}

	if r.domains, err = embedFixture(domains, emb); err != nil {
		return nil, err
	}

	return r, nil
}

// embedFixture embeds the label texts of the given fixture classes and attaches
// the vectors and lexical fields, mirroring what indexFromCache does with cached
// vectors.
func embedFixture(list []indexedClass, emb Embedder) ([]indexedClass, error) {
	out := append([]indexedClass(nil), list...)

	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}

		return out[i].Name < out[j].Name
	})

	texts := make([]string, len(out))
	for i := range out {
		texts[i] = labelText(out[i].Class)
	}

	vecs, err := emb.Embed(context.Background(), texts, RoleDocument)
	if err != nil {
		return nil, fmt.Errorf("embed fixture: %w", err)
	}

	for i := range out {
		out[i].vec = vecs[i]
	}

	attachLexical(out)

	return out, nil
}

// VersionScope selects, version-agnostically, which OASF versions a query
// considers. Callers pick All or Latest without naming concrete versions.
type VersionScope int

const (
	// ScopeAll considers every supported OASF version. Use it for search, so a
	// record published against any version remains discoverable. This is the
	// default.
	ScopeAll VersionScope = iota
	// ScopeLatest considers only the newest supported OASF version. Use it when
	// enriching a record on import, where labels must belong to one version.
	ScopeLatest
)

// queryConfig holds the effective per-call scoring parameters.
type queryConfig struct {
	tiers         int
	tierRatio     float64
	minScore      float64
	minResults    int                 // global floor per kind
	minPerVersion int                 // floor per in-scope version (multi-version scopes)
	scope         VersionScope        // used when explicit is empty
	explicit      map[string]struct{} // explicit version pin; overrides scope
}

// QueryOption overrides parameters for a single Extract call.
type QueryOption func(*queryConfig)

// All searches every supported OASF version (the default). Intended for
// searching a node so records of any version are found.
func All() QueryOption {
	return func(c *queryConfig) { c.scope = ScopeAll }
}

// Latest searches only the newest supported OASF version. Intended for
// enriching a record on import.
func Latest() QueryOption {
	return func(c *queryConfig) { c.scope = ScopeLatest }
}

// MinResultsPerVersion guarantees at least n results per kind for each in-scope
// OASF version (default 1). It only affects multi-version scopes; for a single
// version, MinResults applies. This is what keeps version-exclusive classes -
// e.g. a skill that exists only in 0.7.0 - discoverable under All.
func MinResultsPerVersion(n int) QueryOption {
	return func(c *queryConfig) {
		if n >= 0 {
			c.minPerVersion = n
		}
	}
}

// Tiers sets how many score tiers to return per kind (default 1). Results are
// grouped into tiers separated by a relative drop in score (see WithTierRatio):
// Tiers(1) returns only the closest group, Tiers(2) the closest two, and so on.
func Tiers(n int) QueryOption {
	return func(c *queryConfig) {
		if n >= 1 {
			c.tiers = n
		}
	}
}

// MinScore drops results scoring below s (subject to MinResults).
func MinScore(s float64) QueryOption {
	return func(c *queryConfig) {
		if s >= 0 {
			c.minScore = s
		}
	}
}

// MinResults guarantees at least n results per kind even if they score below
// MinScore. Default 1, so every query yields at least one skill and one domain.
func MinResults(n int) QueryOption {
	return func(c *queryConfig) {
		if n >= 0 {
			c.minResults = n
		}
	}
}

// Versions pins the search to an explicit set of OASF schema versions,
// overriding the All/Latest scope. Use it to enrich a record whose exact
// version is known. Passing an unsupported version makes Extract return an
// error.
func Versions(vs ...string) QueryOption {
	return func(c *queryConfig) {
		for _, v := range vs {
			if c.explicit == nil {
				c.explicit = make(map[string]struct{})
			}

			c.explicit[v] = struct{}{}
		}
	}
}

// Extract returns the OASF skills and domains most relevant to text. text may
// be a short search query or an entire SKILL.md; long inputs are automatically
// chunked to fit the embedder's context window.
//
// By default (ScopeAll) it searches every supported OASF version and guarantees
// per-version coverage, so records of any version are discoverable - use this
// for searching a node. Pass Latest() (or Versions("x")) to scope to a single
// version - use this when enriching a record on import.
func (r *Extractor) Extract(ctx context.Context, text string, opts ...QueryOption) (Result, error) {
	cfg := queryConfig{
		tiers:         r.tiers,
		tierRatio:     r.tierRatio,
		minScore:      r.minScore,
		minResults:    1,
		minPerVersion: 1,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	versions, err := r.resolveVersions(cfg)
	if err != nil {
		return Result{}, err
	}

	want := make(map[string]struct{}, len(versions))
	for _, v := range versions {
		want[v] = struct{}{}
	}

	// Semantic side: embed each chunk of the (possibly long) input.
	chunks := chunkText(text, r.embedder.MaxTokens(), r.overlap)

	chunkVecs, err := r.embedder.Embed(ctx, chunks, RoleQuery)
	if err != nil {
		return Result{}, fmt.Errorf("embed query: %w", err)
	}

	// Lexical side: derive query tokens once from the whole input.
	queryLower := strings.ToLower(text)
	queryTokens := tokenSet(text)

	// Modules match on literal mention, so they use the raw (unfiltered) tokens:
	// stopwords like "skill" must still be able to name a module.
	moduleTokens := make(map[string]struct{})
	for _, t := range tokenize(text) {
		moduleTokens[t] = struct{}{}
	}

	skills := r.score(r.skills, KindSkill, chunkVecs, queryLower, queryTokens, want)
	domains := r.score(r.domains, KindDomain, chunkVecs, queryLower, queryTokens, want)
	modules := scoreModules(queryLower, moduleTokens, want)

	return Result{
		Skills:   selectCovered(skills, versions, cfg),
		Domains:  selectCovered(domains, versions, cfg),
		Modules:  modules,
		Keywords: extractKeywords(text),
	}, nil
}

// resolveVersions turns the scope/explicit configuration into a concrete,
// supported-order list of OASF versions to search.
func (r *Extractor) resolveVersions(cfg queryConfig) ([]string, error) {
	if len(cfg.explicit) > 0 {
		out := make([]string, 0, len(cfg.explicit))

		for _, v := range r.versions { // preserve ascending order
			if _, ok := cfg.explicit[v]; ok {
				out = append(out, v)
			}
		}

		// Surface any pinned version that is not supported.
		if len(out) != len(cfg.explicit) {
			for v := range cfg.explicit {
				if !r.IsSupported(v) {
					return nil, fmt.Errorf("unsupported OASF version %q (supported: %s)",
						v, strings.Join(r.versions, ", "))
				}
			}
		}

		return out, nil
	}

	if cfg.scope == ScopeLatest {
		return []string{r.LatestVersion()}, nil
	}

	return r.SupportedVersions(), nil
}

// score combines the semantic and lexical signals for every class in scope and
// returns the list sorted by descending Score. want restricts scoring to
// classes present in at least one of the requested versions (nil => all).
func (r *Extractor) score(
	classes []indexedClass,
	kind Kind,
	chunkVecs [][]float32,
	queryLower string,
	queryTokens map[string]struct{},
	want map[string]struct{},
) []ScoredClass {
	ws, wl := r.wSemantic, r.wLexical
	if kind == KindDomain {
		ws, wl = r.wSemanticDomain, r.wLexicalDomain
	}

	wSum := ws + wl

	out := make([]ScoredClass, 0, len(classes))

	for i := range classes {
		ic := &classes[i]

		if !versionInScope(ic.versions, want) {
			continue
		}

		// Semantic score: mean of the top-N per-chunk cosines (top-N-mean
		// pooling) so a single tangential chunk cannot dominate a long input.
		sims := make([]float64, len(chunkVecs))
		for j, cv := range chunkVecs {
			sims[j] = dot(cv, ic.vec)
		}

		sem := poolScores(sims)
		if sem < 0 {
			sem = 0
		}

		lex := lexicalScore(queryLower, queryTokens, ic)
		combined := (ws*sem + wl*lex) / wSum

		out = append(out, ScoredClass{
			Class:    ic.Class,
			Kind:     kind,
			Versions: ic.versions,
			Score:    combined,
			Semantic: sem,
			Lexical:  lex,
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		// Deterministic tie-break by id.
		return out[i].ID < out[j].ID
	})

	return out
}

// versionInScope reports whether a class belonging to `have` versions should be
// included given the requested `want` set (nil/empty => include everything).
func versionInScope(have []string, want map[string]struct{}) bool {
	if len(want) == 0 {
		return true
	}

	for _, v := range have {
		if _, ok := want[v]; ok {
			return true
		}
	}

	return false
}

// selectCovered ranks results by score (the first cfg.tiers score tiers, with
// the MinResults floor and MinScore threshold) and then, for multi-version
// scopes, tops up the list so that every in-scope version has at least
// MinResultsPerVersion entries. The per-version top-up may push the result past
// the selected tiers on purpose: coverage of all versions takes precedence, so
// version-exclusive classes stay discoverable. The returned slice is sorted by
// descending score.
func selectCovered(sorted []ScoredClass, versions []string, cfg queryConfig) []ScoredClass {
	out := selectByTiers(sorted, cfg)

	if cfg.minPerVersion > 0 && len(versions) > 1 {
		out = ensureVersionCoverage(out, sorted, versions, cfg.minPerVersion)

		sort.SliceStable(out, func(i, j int) bool {
			if out[i].Score != out[j].Score {
				return out[i].Score > out[j].Score
			}

			return out[i].ID < out[j].ID
		})
	}

	return out
}

// ensureVersionCoverage adds the best-scoring not-yet-selected classes until
// every version has at least minPerVersion entries whose Versions contain it.
func ensureVersionCoverage(out, sorted []ScoredClass, versions []string, minPerVersion int) []ScoredClass {
	selected := make(map[string]struct{}, len(out))
	covered := make(map[string]int, len(versions))

	for _, sc := range out {
		selected[sc.Name] = struct{}{}

		for _, v := range sc.Versions {
			covered[v]++
		}
	}

	for _, v := range versions {
		for covered[v] < minPerVersion {
			best := -1

			for i := range sorted {
				if _, taken := selected[sorted[i].Name]; taken {
					continue
				}

				if containsString(sorted[i].Versions, v) {
					best = i

					break // sorted is descending by score, so the first is best
				}
			}

			if best < 0 {
				break // no more classes cover this version
			}

			sc := sorted[best]
			out = append(out, sc)
			selected[sc.Name] = struct{}{}

			for _, vv := range sc.Versions {
				covered[vv]++
			}
		}
	}

	return out
}

func containsString(ss []string, target string) bool {
	return slices.Contains(ss, target)
}

// --- Package-level convenience -------------------------------------------------

var (
	defaultOnce sync.Once
	defaultRec  *Extractor
	errDefault  error
)

// Extract is a convenience wrapper that lazily builds (and caches) a default
// Extractor from previously provisioned assets. The OASF endpoint is read from
// the OASF_URL environment variable, and assets must have been created by a
// prior Provision at the default asset dir. For custom endpoints, embedders, or
// weights, construct an Extractor with New and reuse it.
func Extract(ctx context.Context, text string, opts ...QueryOption) (Result, error) {
	defaultOnce.Do(func() {
		url := os.Getenv("OASF_URL")
		if url == "" {
			errDefault = errors.New("OASF_URL environment variable is required for the package-level Extract; use New with WithOASFURL instead")

			return
		}

		defaultRec, errDefault = New(WithOASFURL(url))
	})

	if errDefault != nil {
		return Result{}, errDefault
	}

	return defaultRec.Extract(ctx, text, opts...)
}
