// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.ListenAddress != DefaultListenAddress {
		t.Errorf("ListenAddress = %q, want %q", cfg.ListenAddress, DefaultListenAddress)
	}

	if cfg.Extractor.OASFURL != "" {
		t.Errorf("Extractor.OASFURL = %q, want empty (extractor disabled)", cfg.Extractor.OASFURL)
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("OASF_SDK_LISTEN_ADDRESS", "127.0.0.1:9999")
	t.Setenv("OASF_SDK_EXTRACTOR_OASF_URL", "https://schema.oasf.outshift.com")
	t.Setenv("OASF_SDK_EXTRACTOR_MODEL_NAME", "sentence-transformers/all-MiniLM-L12-v2")
	t.Setenv("OASF_SDK_EXTRACTOR_ASSET_DIR", "/tmp/assets")
	t.Setenv("OASF_SDK_EXTRACTOR_SKILL_SEMANTIC_WEIGHT", "0.7")
	t.Setenv("OASF_SDK_EXTRACTOR_TIERS", "2")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if cfg.ListenAddress != "127.0.0.1:9999" {
		t.Errorf("ListenAddress = %q, want 127.0.0.1:9999", cfg.ListenAddress)
	}

	ex := cfg.Extractor
	if ex.OASFURL != "https://schema.oasf.outshift.com" {
		t.Errorf("OASFURL = %q", ex.OASFURL)
	}

	if ex.ModelName != "sentence-transformers/all-MiniLM-L12-v2" {
		t.Errorf("ModelName = %q", ex.ModelName)
	}

	if ex.AssetDir != "/tmp/assets" {
		t.Errorf("AssetDir = %q", ex.AssetDir)
	}

	if ex.SkillSemanticWeight != 0.7 {
		t.Errorf("SkillSemanticWeight = %v, want 0.7", ex.SkillSemanticWeight)
	}

	if ex.Tiers != 2 {
		t.Errorf("Tiers = %d, want 2", ex.Tiers)
	}
}
