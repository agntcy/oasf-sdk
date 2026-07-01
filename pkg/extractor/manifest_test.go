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
		l != filepath.Join("/root", "label_vectors.bin") ||
		mf != filepath.Join("/root", "manifest.json") {
		t.Fatalf("unexpected paths: %s %s %s", m, l, mf)
	}
}

func TestLabelVectorsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	_, labels, _ := assetPaths(dir)

	skills := [][]float32{{0.1, 0.2}, {0.3, 0.4}}
	domains := [][]float32{{0.5, 0.6}}

	if err := writeLabelVectors(labels, skills, domains); err != nil {
		t.Fatalf("writeLabelVectors: %v", err)
	}

	gotSkills, gotDomains, err := readLabelVectors(labels)
	if err != nil {
		t.Fatalf("readLabelVectors: %v", err)
	}

	if len(gotSkills) != 2 || gotSkills[1][1] != 0.4 || len(gotDomains) != 1 || gotDomains[0][0] != 0.5 {
		t.Fatalf("round trip mismatch: skills=%v domains=%v", gotSkills, gotDomains)
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
