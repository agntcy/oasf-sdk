// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const manifestFormatVersion = 1

// File permission constants used when writing assets to disk.
const (
	permAssetDir  = 0o755 // directories are traversable by owner+group+other
	permAssetFile = 0o600 // files are readable/writable by owner only
)

// manifest records what Provision produced so New can trust the cached assets
// without re-deriving or re-embedding the catalog on every start.
type manifest struct {
	FormatVersion    int      `json:"format_version"`
	ModelName        string   `json:"model_name"`
	ModelID          string   `json:"model_id"`
	OASFURL          string   `json:"oasf_url"`
	TaxonomyVersions []string `json:"taxonomy_versions"`
	SkillDigest      string   `json:"skill_digest"`
	DomainDigest     string   `json:"domain_digest"`
}

// assetPaths returns the model dir, label-vectors file, and manifest file under
// the asset directory.
func assetPaths(dir string) (string, string, string) {
	return filepath.Join(dir, "models"),
		filepath.Join(dir, "label_vectors.bin"),
		filepath.Join(dir, "manifest.json")
}

func writeManifest(path string, m manifest) error {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), permAssetDir); err != nil {
		return fmt.Errorf("create manifest dir: %w", err)
	}

	if err := os.WriteFile(path, b, permAssetFile); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

func readManifest(path string) (manifest, error) {
	var m manifest

	b, err := os.ReadFile(path)
	if err != nil {
		return m, fmt.Errorf("failed to read manifest: %w", err)
	}

	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("parse manifest: %w", err)
	}

	return m, nil
}

// labelText builds the single string embedded for one class. Provision and the
// load-time digest check MUST build label text via this helper: catalogDigest
// is computed over these strings, so any divergence would cause checkLabelDigest
// to detect stale cached vectors.
func labelText(c Class) string {
	return humanize(c.Name) + ". " + c.Caption + ". " + c.Description
}

// checkLabelDigest verifies that the digest of texts matches the stored digest
// for kind in m. It returns nil when the digests match and an error describing
// the staleness when they differ. It is a pure function with no I/O, so it is
// easily unit-tested.
func checkLabelDigest(kind Kind, texts []string, m manifest) error {
	got := catalogDigest(texts)

	var stored string

	switch kind {
	case KindSkill:
		stored = m.SkillDigest
	case KindDomain:
		stored = m.DomainDigest
	case KindModule:
		// Modules are curated in-process; there is no stored digest to check.
		return nil
	default:
		// Unknown kinds have no stored digest; skip the check.
		return nil
	}

	if got != stored {
		return fmt.Errorf(
			"cached %s vectors are stale (taxonomy at %s changed since provisioning); re-run Provision",
			kind, m.OASFURL,
		)
	}

	return nil
}

// catalogDigest hashes the ordered label texts so the manifest is invalidated
// whenever the catalog (or the text built from it) changes.
func catalogDigest(texts []string) string {
	h := sha256.New()
	for _, t := range texts {
		h.Write([]byte(t))
		h.Write([]byte{0})
	}

	return hex.EncodeToString(h.Sum(nil)[:16])
}

// labelVectorFile is the gob-encoded on-disk form of the cached label vectors.
type labelVectorFile struct {
	Skills  [][]float32
	Domains [][]float32
}

// writeLabelVectors gob-encodes the skill and domain label vectors to path.
func writeLabelVectors(path string, skills, domains [][]float32) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(labelVectorFile{Skills: skills, Domains: domains}); err != nil {
		return fmt.Errorf("encode label vectors: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), permAssetDir); err != nil {
		return fmt.Errorf("create label vectors dir: %w", err)
	}

	if err := os.WriteFile(path, buf.Bytes(), permAssetFile); err != nil {
		return fmt.Errorf("write label vectors: %w", err)
	}

	return nil
}

// readLabelVectors decodes the gob-encoded label vectors written by
// writeLabelVectors.
func readLabelVectors(path string) ([][]float32, [][]float32, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read label vectors: %w", err)
	}

	var f labelVectorFile
	if err := gob.NewDecoder(bytes.NewReader(b)).Decode(&f); err != nil {
		return nil, nil, fmt.Errorf("decode label vectors: %w", err)
	}

	return f.Skills, f.Domains, nil
}
