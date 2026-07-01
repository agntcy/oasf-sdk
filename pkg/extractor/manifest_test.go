// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"path/filepath"
	"testing"
)

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	_, _, mf := assetPaths(dir)

	want := manifest{
		FormatVersion:    manifestFormatVersion,
		ModelName:        "all-MiniLM-L6-v2",
		ModelID:          "cybertron:sentence-transformers/all-MiniLM-L6-v2",
		OASFURL:          "http://localhost:8080",
		TaxonomyVersions: []string{"1.0.0"},
		SkillDigest:      "abc",
		DomainDigest:     "def",
	}
	if err := writeManifest(mf, want); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}

	got, err := readManifest(mf)
	if err != nil {
		t.Fatalf("readManifest: %v", err)
	}

	if got.ModelID != want.ModelID || got.ModelName != want.ModelName ||
		got.SkillDigest != want.SkillDigest ||
		len(got.TaxonomyVersions) != 1 || got.TaxonomyVersions[0] != "1.0.0" {
		t.Fatalf("round trip mismatch: %+v", got)
	}
}

func TestAssetPaths(t *testing.T) {
	m, l, mf := assetPaths("/root")
	if m != filepath.Join("/root", "models") ||
		l != filepath.Join("/root", "index.bin") ||
		mf != filepath.Join("/root", "manifest.json") {
		t.Fatalf("unexpected paths: %s %s %s", m, l, mf)
	}
}

func TestIndexRoundTrip(t *testing.T) {
	dir := t.TempDir()
	_, indexPath, _ := assetPaths(dir)

	skills := []persistedClass{
		{Class: Class{ID: 101, Name: "a/b"}, Versions: []string{"1.0.0"}, Vec: []float32{0.1, 0.2}},
	}
	domains := []persistedClass{
		{Class: Class{ID: 201, Name: "c/d"}, Versions: []string{"0.8.0", "1.0.0"}, Vec: []float32{0.5}},
	}

	if err := writeIndex(indexPath, skills, domains); err != nil {
		t.Fatalf("writeIndex: %v", err)
	}

	gotSkills, gotDomains, err := readIndex(indexPath)
	if err != nil {
		t.Fatalf("readIndex: %v", err)
	}

	if len(gotSkills) != 1 || gotSkills[0].ID != 101 || gotSkills[0].Vec[1] != 0.2 ||
		len(gotDomains) != 1 || gotDomains[0].Vec[0] != 0.5 || len(gotDomains[0].Versions) != 2 {
		t.Fatalf("round trip mismatch: skills=%+v domains=%+v", gotSkills, gotDomains)
	}
}

func TestManifestCurrent(t *testing.T) {
	dir := t.TempDir()
	_, indexPath, mf := assetPaths(dir)

	o := options{modelName: "all-MiniLM-L6-v2", oasfURL: "http://oasf", assetDir: dir}
	versions := []string{"1.0.0"}
	sd, dd := "skill-digest", "domain-digest"

	// Nothing on disk yet.
	if manifestCurrent(mf, indexPath, o, versions, sd, dd, 2, 1) {
		t.Fatal("expected not current when assets are absent")
	}

	if err := writeManifest(mf, manifest{
		FormatVersion:    manifestFormatVersion,
		ModelName:        o.modelName,
		ModelID:          "cybertron:sentence-transformers/all-MiniLM-L6-v2",
		OASFURL:          o.oasfURL,
		TaxonomyVersions: versions,
		SkillDigest:      sd,
		DomainDigest:     dd,
	}); err != nil {
		t.Fatal(err)
	}

	if err := writeIndex(indexPath,
		[]persistedClass{{Class: Class{ID: 1}}, {Class: Class{ID: 2}}},
		[]persistedClass{{Class: Class{ID: 3}}}); err != nil {
		t.Fatal(err)
	}

	if !manifestCurrent(mf, indexPath, o, versions, sd, dd, 2, 1) {
		t.Fatal("expected current for a matching configuration")
	}

	// Each mismatch must invalidate the cache.
	if manifestCurrent(mf, indexPath, o, versions, "CHANGED", dd, 2, 1) {
		t.Error("skill digest change should invalidate")
	}

	if manifestCurrent(mf, indexPath, o, []string{"1.0.0", "1.1.0"}, sd, dd, 2, 1) {
		t.Error("taxonomy version change should invalidate")
	}

	if manifestCurrent(mf, indexPath, o, versions, sd, dd, 3, 1) {
		t.Error("skill count change should invalidate")
	}

	other := o
	other.modelName = "other-model"

	if manifestCurrent(mf, indexPath, other, versions, sd, dd, 2, 1) {
		t.Error("model change should invalidate")
	}
}

func TestNewErrorsOnOASFURLMismatch(t *testing.T) {
	dir := t.TempDir()
	_, _, mf := assetPaths(dir)

	if err := writeManifest(mf, manifest{
		FormatVersion: manifestFormatVersion,
		OASFURL:       "http://provisioned",
	}); err != nil {
		t.Fatal(err)
	}

	// New reads the manifest and rejects a different OASF URL before touching the
	// index or the model, so this needs no assets or network.
	if _, err := New(WithOASFURL("http://different"), WithAssetDir(dir)); err == nil {
		t.Fatal("expected error when the requested OASF URL differs from the provisioned one")
	}
}

func TestCatalogDigestStable(t *testing.T) {
	a := catalogDigest([]string{"x", "y"})
	if a == "" {
		t.Fatal("empty digest")
	}

	if a != catalogDigest([]string{"x", "y"}) {
		t.Fatal("digest not stable")
	}

	if a == catalogDigest([]string{"x", "z"}) {
		t.Fatal("digest should change with content")
	}
}

func TestCheckLabelDigest(t *testing.T) {
	texts := []string{"alpha. Caption A. Description A.", "beta. Caption B. Description B."}
	digest := catalogDigest(texts)

	// Matching digests: skill kind
	m := manifest{
		OASFURL:      "http://example.com",
		SkillDigest:  digest,
		DomainDigest: "unrelated",
	}
	if err := checkLabelDigest(KindSkill, texts, m); err != nil {
		t.Errorf("checkLabelDigest(KindSkill, matching): want nil, got %v", err)
	}

	// Matching digests: domain kind
	m2 := manifest{
		OASFURL:      "http://example.com",
		SkillDigest:  "unrelated",
		DomainDigest: digest,
	}
	if err := checkLabelDigest(KindDomain, texts, m2); err != nil {
		t.Errorf("checkLabelDigest(KindDomain, matching): want nil, got %v", err)
	}

	// Mismatched digest: skill kind — must return a non-nil error.
	changedTexts := []string{"alpha. Caption A. Description CHANGED.", "beta. Caption B. Description B."}
	if err := checkLabelDigest(KindSkill, changedTexts, m); err == nil {
		t.Error("checkLabelDigest(KindSkill, changed texts): want non-nil error, got nil")
	}

	// Mismatched digest: domain kind — must return a non-nil error.
	if err := checkLabelDigest(KindDomain, changedTexts, m2); err == nil {
		t.Error("checkLabelDigest(KindDomain, changed texts): want non-nil error, got nil")
	}

	// Unknown kind: no stored digest, must return nil regardless of texts.
	if err := checkLabelDigest(KindModule, changedTexts, m); err != nil {
		t.Errorf("checkLabelDigest(KindModule, unknown kind): want nil, got %v", err)
	}
}
