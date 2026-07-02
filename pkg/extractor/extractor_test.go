// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"hash/fnv"
	"os"
	"slices"
	"strings"
	"testing"
)

// fixtureVersions are the OASF versions the in-memory test fixture spans. They
// match the versions used by the curated module catalog so module version
// assertions hold.
var fixtureVersions = []string{"0.7.0", "0.8.0", "1.0.0"}

// ic is a tiny helper to build a fixture indexedClass.
func ic(id uint64, name, caption, desc string, versions ...string) indexedClass {
	return indexedClass{
		Class:    Class{ID: id, Name: name, Caption: caption, Description: desc},
		versions: versions,
	}
}

// fixtureSkills is a small representative skill catalog spanning all three
// fixture versions. text_completion appears in all three; a couple of classes
// are version-exclusive so per-version coverage and version filtering are
// genuinely exercised.
func fixtureSkills() []indexedClass {
	all := fixtureVersions

	return []indexedClass{
		ic(10101, "natural_language_processing/natural_language_generation/text_completion",
			"Text Completion", "Complete and generate natural language text.", all...),
		ic(10102, "natural_language_processing/information_retrieval_synthesis/text_summarization",
			"Text Summarization", "Summarize long documents into concise text.", all...),
		ic(10103, "natural_language_processing/natural_language_understanding/sentiment_analysis",
			"Sentiment Analysis", "Detect sentiment and mood in text.", "0.8.0", "1.0.0"),
		ic(20201, "computer_vision/image_classification",
			"Image Classification", "Classify images into categories.", all...),
		ic(20202, "computer_vision/object_detection",
			"Object Detection", "Detect and localize objects in images.", "1.0.0"),
		ic(30301, "data_science/data_engineering_pipelines",
			"Data Engineering Pipelines", "Build data engineering pipelines for ingestion.", "0.7.0"),
		ic(30302, "data_science/anomaly_detection",
			"Anomaly Detection", "Detect anomalies and fraud in data.", all...),
	}
}

// fixtureDomains is a small representative domain catalog spanning all three
// fixture versions.
func fixtureDomains() []indexedClass {
	all := fixtureVersions

	return []indexedClass{
		ic(40401, "retail/retail_analytics",
			"Retail Analytics", "Analytics for online retail and commerce.", all...),
		ic(40402, "finance/fraud_detection",
			"Fraud Detection", "Detect fraud in financial and retail transactions.", all...),
		ic(40403, "healthcare/clinical_operations",
			"Clinical Operations", "Healthcare clinical operations domain.", "0.7.0", "0.8.0"),
		ic(40404, "manufacturing/supply_chain",
			"Supply Chain", "Manufacturing supply chain domain.", "1.0.0"),
	}
}

// fakeEmbedder is a fast, deterministic token-hashing embedder used to keep the
// logic tests instant. It avoids loading the real (slow) transformer model;
// the transformer's behavior is covered by TestDefaultTransformer (gated).
type fakeEmbedder struct{ dim int }

func (f *fakeEmbedder) Embed(_ context.Context, texts []string, _ EmbedRole) ([][]float32, error) {
	out := make([][]float32, len(texts))

	for i, t := range texts {
		vec := make([]float32, f.dim)

		for _, tok := range tokenize(t) {
			h := fnv.New32a()
			_, _ = h.Write([]byte(tok))
			vec[h.Sum32()%uint32(f.dim)] += 1 //nolint:gosec // test fake embedder; dim is a small constant
		}

		l2normalize(vec)
		out[i] = vec
	}

	return out, nil
}

func (f *fakeEmbedder) Dim() int       { return f.dim }
func (f *fakeEmbedder) MaxTokens() int { return 256 }
func (f *fakeEmbedder) ID() string     { return "fake" }

// newTestExtractor builds an extractor from the in-memory fixture with the fast
// fake embedder, so tests exercise the scoring/indexing logic without loading
// the real model, hitting the network, or reading provisioned assets.
func newTestExtractor(t *testing.T, opts ...Option) *Extractor {
	t.Helper()

	r, err := newFromClasses(fixtureVersions, fixtureSkills(), fixtureDomains(),
		&fakeEmbedder{dim: 256}, opts...)
	if err != nil {
		t.Fatal(err)
	}

	return r
}

func TestSupportedVersions(t *testing.T) {
	r := newTestExtractor(t)

	if got := r.SupportedVersions(); len(got) != 3 {
		t.Fatalf("expected 3 supported versions, got %v", got)
	}

	for _, v := range fixtureVersions {
		if !r.IsSupported(v) {
			t.Errorf("version %q should be supported", v)
		}
	}

	if r.IsSupported("9.9.9") {
		t.Error("version 9.9.9 should not be supported")
	}

	if got := r.LatestVersion(); got != "1.0.0" {
		t.Errorf("LatestVersion = %q, want 1.0.0", got)
	}
}

func TestNewMergesAllVersions(t *testing.T) {
	r := newTestExtractor(t)

	if len(r.skills) == 0 || len(r.domains) == 0 {
		t.Fatalf("empty catalog (skills=%d domains=%d)", len(r.skills), len(r.domains))
	}

	// The fixture has 7 skills and 4 domains.
	if len(r.skills) != len(fixtureSkills()) {
		t.Errorf("expected %d merged skills, got %d", len(fixtureSkills()), len(r.skills))
	}

	for _, c := range append(append([]indexedClass{}, r.skills...), r.domains...) {
		if c.ID == 0 {
			t.Errorf("base class (id 0) leaked into catalog: %q", c.Name)
		}
	}

	for _, c := range r.skills {
		if c.Name == "natural_language_processing/natural_language_generation/text_completion" {
			if len(c.versions) != 3 {
				t.Errorf("text_completion should appear in all 3 versions, got %v", c.versions)
			}
		}
	}
}

func TestUnsupportedVersionOption(t *testing.T) {
	r := newTestExtractor(t)

	if _, err := r.Extract(context.Background(), "anything", Versions("9.9.9")); err == nil {
		t.Fatal("expected error for unsupported version in Versions option")
	}
}

func TestVerbatimCaptionRanksFirst(t *testing.T) {
	r := newTestExtractor(t)

	res, err := r.Extract(context.Background(), "Text Summarization")
	if err != nil {
		t.Fatal(err)
	}

	if got := res.Skills[0].Name; !strings.Contains(got, "summarization") {
		t.Errorf("top skill for 'Text Summarization' = %q, want a summarization skill", got)
	}

	res, err = r.Extract(context.Background(), "Retail Analytics")
	if err != nil {
		t.Fatal(err)
	}

	if got := res.Domains[0].Name; !strings.Contains(got, "retail_analytics") {
		t.Errorf("top domain for 'Retail Analytics' = %q, want retail_analytics", got)
	}
}

func TestVersionsFilter(t *testing.T) {
	r := newTestExtractor(t)

	// Default: a class shared across versions reports all of them.
	all, err := r.Extract(context.Background(), "Text Summarization", Tiers(1))
	if err != nil {
		t.Fatal(err)
	}

	if got := all.Skills[0].Versions; len(got) != 3 {
		t.Errorf("default search should report all versions for a shared skill, got %v", got)
	}

	// Restricting to a single version: every result must belong to it.
	only07, err := r.Extract(context.Background(), "data engineering pipelines",
		Versions("0.7.0"), Tiers(5))
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range append(append([]ScoredClass{}, only07.Skills...), only07.Domains...) {
		if !slices.Contains(s.Versions, "0.7.0") {
			t.Errorf("Versions(0.7.0) returned %q present only in %v", s.Name, s.Versions)
		}
	}
}

// Use case 1: searching a node. Under the default All scope, every supported
// version must be represented by at least one skill and one domain.
func TestSearchScopeCoversEveryVersion(t *testing.T) {
	r := newTestExtractor(t)

	res, err := r.Extract(context.Background(), "fraud detection for online retail", Tiers(2))
	if err != nil {
		t.Fatal(err)
	}

	for _, v := range r.SupportedVersions() {
		if !kindCoversVersion(res.Skills, v) {
			t.Errorf("no skill covers version %s: %v", v, withVersions(res.Skills))
		}

		if !kindCoversVersion(res.Domains, v) {
			t.Errorf("no domain covers version %s: %v", v, withVersions(res.Domains))
		}
	}
}

// Use case 2: enriching a record on import. Latest scope must return only
// classes that belong to the latest version.
func TestEnrichScopeLatestOnly(t *testing.T) {
	r := newTestExtractor(t)

	latest := r.LatestVersion()

	res, err := r.Extract(context.Background(), "Text Summarization", Latest(), Tiers(5))
	if err != nil {
		t.Fatal(err)
	}

	all := append(append([]ScoredClass{}, res.Skills...), res.Domains...)
	if len(all) == 0 {
		t.Fatal("expected results under Latest scope")
	}

	for _, sc := range all {
		if !slices.Contains(sc.Versions, latest) {
			t.Errorf("Latest scope returned %q not present in %s (versions=%v)", sc.Name, latest, sc.Versions)
		}
	}
}

func TestGuaranteeAtLeastOneEach(t *testing.T) {
	r := newTestExtractor(t)

	res, err := r.Extract(context.Background(), "zzzz qqqq wwww")
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Skills) < 1 || len(res.Domains) < 1 {
		t.Fatalf("MinResults guarantee violated: skills=%d domains=%d", len(res.Skills), len(res.Domains))
	}
}

func TestResultsFirstTierUnderLatest(t *testing.T) {
	r := newTestExtractor(t)

	// Under a single-version scope there is no per-version top-up, so Tiers(1)
	// returns exactly the closest tier.
	res, err := r.Extract(context.Background(),
		"image classification and object detection", Latest(), Tiers(1))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Skills) == 0 {
		t.Fatal("expected at least one skill")
	}

	for _, sc := range res.Skills {
		if sc.Tier != 1 {
			t.Errorf("Tiers(1) returned a non-first-tier skill %q (tier %d)", sc.Name, sc.Tier)
		}
	}

	for i := 1; i < len(res.Skills); i++ {
		if res.Skills[i-1].Score < res.Skills[i].Score {
			t.Errorf("skills not sorted by descending score at index %d", i)
		}
	}

	for _, sc := range append(append([]ScoredClass{}, res.Skills...), res.Domains...) {
		if sc.Score < 0 || sc.Score > 1 {
			t.Errorf("score out of [0,1] for %q: %v", sc.Name, sc.Score)
		}
	}
}

func TestChunkText(t *testing.T) {
	if got := chunkText("short input", 256, 32); len(got) != 1 {
		t.Errorf("short input should yield 1 chunk, got %d", len(got))
	}

	long := strings.Repeat("filler word ", 400) // 800 tokens
	chunks := chunkText(long, 256, 32)

	if len(chunks) < 2 {
		t.Fatalf("long input should be split into multiple chunks, got %d", len(chunks))
	}

	for _, c := range chunks {
		if n := len(strings.Fields(c)); n > 256 {
			t.Errorf("chunk exceeds maxTokens: %d > 256", n)
		}
	}
}

func TestLongSkillMdInput(t *testing.T) {
	r := newTestExtractor(t)

	skillMD := "# My Agent\n\n" +
		strings.Repeat("This section describes operational details. ", 60) +
		"\n\n## Capabilities\n\nThe agent performs Text Summarization of long documents.\n\n" +
		strings.Repeat("Additional unrelated boilerplate content here. ", 60)

	res, err := r.Extract(context.Background(), skillMD, Tiers(5))
	if err != nil {
		t.Fatal(err)
	}

	found := false

	for _, s := range res.Skills {
		if strings.Contains(s.Name, "summarization") {
			found = true

			break
		}
	}

	if !found {
		t.Errorf("expected a summarization skill among results for the SKILL.md, got %v", classNames(res.Skills))
	}
}

func TestNewErrorsWhenNotProvisioned(t *testing.T) {
	_, err := New(WithOASFURL("http://localhost:1"), WithAssetDir(t.TempDir()))
	if err == nil {
		t.Fatal("expected error when assets are not provisioned")
	}
}

func TestNewErrorsWithoutOASFURL(t *testing.T) {
	_, err := New(WithAssetDir(t.TempDir()))
	if err == nil {
		t.Fatal("expected error when OASF URL is empty")
	}
}

func TestProvisionErrorsWithoutOASFURL(t *testing.T) {
	if err := Provision(context.Background(), WithAssetDir(t.TempDir())); err == nil {
		t.Fatal("expected error when OASF URL is empty")
	}
}

// TestDefaultTransformer exercises the real default embedder (the provisioned
// sentence-transformer) end-to-end via the package-level Extract. It is gated
// behind OASF_TRANSFORMER_TEST=1 because it requires previously provisioned
// assets (run Provision first) and the OASF_URL environment variable, and
// loading the model + embedding the query is slow.
//
//	OASF_URL=... OASF_TRANSFORMER_TEST=1 go test ./...
func TestDefaultTransformer(t *testing.T) {
	if os.Getenv("OASF_TRANSFORMER_TEST") != "1" {
		t.Skip("set OASF_TRANSFORMER_TEST=1 to run the provisioned transformer model test")
	}

	// A true synonym query the lexical path cannot resolve ("reviews source
	// code for bugs" shares no words with "quality assurance"): the model can.
	res, err := Extract(context.Background(), "an agent that reviews source code for bugs")
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Skills) == 0 || len(res.Domains) == 0 {
		t.Fatal("expected at least one skill and one domain")
	}

	t.Logf("top skill:  %s (%.3f)", res.Skills[0].Name, res.Skills[0].Score)
	t.Logf("top domain: %s (%.3f)", res.Domains[0].Name, res.Domains[0].Score)
}

func TestShrink(t *testing.T) {
	// Multi-word text drops trailing words and always gets strictly shorter.
	long := strings.Repeat("word ", 100)

	prev := strings.TrimSpace(long)
	for i := 0; i < 50 && prev != ""; i++ {
		next := shrink(prev)
		if next != "" && len(next) >= len(prev) {
			t.Fatalf("shrink did not reduce length: %d -> %d", len(prev), len(next))
		}

		prev = next
	}

	if prev != "" {
		t.Errorf("shrink should eventually reach empty, stuck at %q", prev)
	}

	// A single oversized token (no spaces) is halved by runes, then emptied.
	if got := shrink("aaaaaa"); got != "aaa" {
		t.Errorf("shrink single token = %q, want %q", got, "aaa")
	}

	if got := shrink("a"); got != "" {
		t.Errorf("shrink single char = %q, want empty", got)
	}
}

func TestPerKindWeights(t *testing.T) {
	// Domains default to a higher semantic weight than skills.
	r := newTestExtractor(t)
	if r.wSemanticDomain <= r.wSemantic {
		t.Errorf("domain semantic weight (%v) should exceed skill semantic weight (%v)",
			r.wSemanticDomain, r.wSemantic)
	}
	// WithDomainWeights overrides only the domain blend, leaving skills intact.
	r2 := newTestExtractor(t, WithWeights(0.6, 0.4), WithDomainWeights(1, 0))
	if r2.wSemantic != 0.6 || r2.wLexical != 0.4 {
		t.Errorf("skill weights changed unexpectedly: %v/%v", r2.wSemantic, r2.wLexical)
	}

	if r2.wSemanticDomain != 1 || r2.wLexicalDomain != 0 {
		t.Errorf("domain weights not applied: %v/%v", r2.wSemanticDomain, r2.wLexicalDomain)
	}
	// Invalid domain weights are ignored (negative / zero-sum).
	r3 := newTestExtractor(t, WithDomainWeights(-1, 0))
	if r3.wSemanticDomain != 0.8 || r3.wLexicalDomain != 0.2 {
		t.Errorf("invalid domain weights should be ignored, got %v/%v", r3.wSemanticDomain, r3.wLexicalDomain)
	}
}

func TestTierOptions(t *testing.T) {
	// Defaults: one tier, ratio 0.97.
	r := newTestExtractor(t)
	if r.tiers != 1 {
		t.Errorf("default tiers = %d, want 1", r.tiers)
	}

	if r.tierRatio != 0.97 {
		t.Errorf("default tierRatio = %v, want 0.97", r.tierRatio)
	}

	// Overrides apply.
	r2 := newTestExtractor(t, WithDefaultTiers(3), WithTierRatio(0.9))
	if r2.tiers != 3 || r2.tierRatio != 0.9 {
		t.Errorf("overrides not applied: tiers=%d ratio=%v", r2.tiers, r2.tierRatio)
	}

	// Invalid values are ignored (tiers < 1, ratio out of (0,1]).
	r3 := newTestExtractor(t, WithDefaultTiers(0), WithTierRatio(1.5))
	if r3.tiers != 1 || r3.tierRatio != 0.97 {
		t.Errorf("invalid options should be ignored, got tiers=%d ratio=%v", r3.tiers, r3.tierRatio)
	}
}

func TestModulesLexicalGate(t *testing.T) {
	r := newTestExtractor(t)

	// A literal module mention returns that module (modules are pure lexical,
	// so the fake embedder is irrelevant here).
	res, err := r.Extract(context.Background(), "show me MCP support", Latest())
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Modules) == 0 {
		t.Fatal("expected a module match for 'MCP'")
	}

	foundMCP := false

	for _, m := range res.Modules {
		if m.Kind != KindModule {
			t.Errorf("module result has kind %q", m.Kind)
		}

		if m.Semantic != 0 {
			t.Errorf("module %q has non-zero Semantic %v", m.Name, m.Semantic)
		}

		if strings.Contains(m.Name, "mcp") {
			foundMCP = true
		}
	}

	if !foundMCP {
		t.Errorf("expected an mcp module, got %v", classNames(res.Modules))
	}

	// Sorted by descending score.
	for i := 1; i < len(res.Modules); i++ {
		if res.Modules[i-1].Score < res.Modules[i].Score {
			t.Errorf("modules not sorted by descending score at index %d", i)
		}
	}
}

func TestModulesCuratedSet(t *testing.T) {
	r := newTestExtractor(t)

	// Only mcp, a2a, and agentskills are curated; each of these spellings hits.
	for _, q := range []string{"mcp", "a2a", "agentskills", "agent skills", "agent skill"} {
		res, err := r.Extract(context.Background(), q)
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Modules) == 0 {
			t.Errorf("expected a curated module match for %q", q)
		}
	}

	// Words for modules NOT in the curated list must never match.
	for _, q := range []string{"observability", "evaluation", "agentspec", "acp", "core integration of data"} {
		res, err := r.Extract(context.Background(), q)
		if err != nil {
			t.Fatal(err)
		}

		if len(res.Modules) != 0 {
			t.Errorf("query %q should match no curated module, got %v", q, classNames(res.Modules))
		}
	}
}

func TestModulesReportVersions(t *testing.T) {
	r := newTestExtractor(t)

	// All scope: mcp surfaces both renamed identities, each with its versions.
	res, err := r.Extract(context.Background(), "mcp")
	if err != nil {
		t.Fatal(err)
	}

	got := map[string][]string{}
	for _, m := range res.Modules {
		got[m.Name] = m.Versions
	}

	if !slices.Equal(got["integration/mcp"], []string{"0.8.0", "1.0.0"}) {
		t.Errorf("integration/mcp versions = %v, want [0.8.0 1.0.0]", got["integration/mcp"])
	}

	if !slices.Equal(got["runtime/mcp"], []string{"0.7.0"}) {
		t.Errorf("runtime/mcp versions = %v, want [0.7.0]", got["runtime/mcp"])
	}

	// Latest scope: only the latest identity, scoped to the latest version.
	res, err = r.Extract(context.Background(), "mcp", Latest())
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Modules) != 1 {
		t.Fatalf("Latest mcp should yield one identity, got %v", classNames(res.Modules))
	}

	if res.Modules[0].Name != "integration/mcp" || !slices.Equal(res.Modules[0].Versions, []string{"1.0.0"}) {
		t.Errorf("Latest mcp = %q %v, want integration/mcp [1.0.0]", res.Modules[0].Name, res.Modules[0].Versions)
	}
}

func TestModuleSkillMatches(t *testing.T) {
	r := newTestExtractor(t)

	// "skill" is a stopword for catalog matching, but must still name the
	// Language Model Agent Skills module (the record-type meaning of "skill").
	res, err := r.Extract(context.Background(), "skill for mcp server development", Latest())
	if err != nil {
		t.Fatal(err)
	}

	var hasAgentskills, hasMCP bool

	for _, m := range res.Modules {
		switch m.Name {
		case "core/language_model/agentskills":
			hasAgentskills = true
		case "integration/mcp":
			hasMCP = true
		}
	}

	if !hasAgentskills {
		t.Errorf(`"skill ..." should surface the agentskills module, got %v`, classNames(res.Modules))
	}

	if !hasMCP {
		t.Errorf(`"... mcp ..." should still surface the mcp module, got %v`, classNames(res.Modules))
	}
}

func TestModulesEmptyWhenUnmentioned(t *testing.T) {
	r := newTestExtractor(t)

	// No module name/caption appears in this query -> empty (no MinResults floor).
	res, err := r.Extract(context.Background(), "summarize legal documents")
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Modules) != 0 {
		t.Errorf("expected no modules for an unmentioned query, got %v", classNames(res.Modules))
	}
	// Skills/domains still honor their at-least-one guarantee.
	if len(res.Skills) == 0 || len(res.Domains) == 0 {
		t.Errorf("skills/domains floor violated: skills=%d domains=%d", len(res.Skills), len(res.Domains))
	}
}

func TestModulesIgnoreTiersAndMinScore(t *testing.T) {
	r := newTestExtractor(t)

	// Tiers(1) and a near-maximal MinScore must not drop matched modules.
	res, err := r.Extract(context.Background(), "show me MCP support",
		Latest(), Tiers(1), MinScore(0.99))
	if err != nil {
		t.Fatal(err)
	}

	if len(res.Modules) == 0 {
		t.Fatal("modules must ignore Tiers/MinScore; expected an mcp match")
	}
}

func classNames(scs []ScoredClass) []string {
	out := make([]string, len(scs))
	for i, sc := range scs {
		out[i] = sc.Name
	}

	return out
}

func kindCoversVersion(scs []ScoredClass, version string) bool {
	for _, sc := range scs {
		if slices.Contains(sc.Versions, version) {
			return true
		}
	}

	return false
}

func withVersions(scs []ScoredClass) []string {
	out := make([]string, len(scs))
	for i, sc := range scs {
		out[i] = sc.Name + strings.Join(sc.Versions, "|")
	}

	return out
}
