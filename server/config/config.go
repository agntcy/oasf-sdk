// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	DefaultEnvPrefix     = "OASF_SDK"
	DefaultListenAddress = "0.0.0.0:31234"
)

type Config struct {
	ListenAddress string          `json:"listen_address,omitempty" mapstructure:"listen_address"`
	Extractor     ExtractorConfig `json:"extractor"                mapstructure:"extractor"`
}

// ExtractorConfig configures the extractor controller. The controller is
// registered only when OASFURL is set — it is the extractor's on/off switch —
// and it provisions its assets at startup. The remaining fields are optional and
// fall back to the pkg/extractor defaults.
type ExtractorConfig struct {
	// OASFURL is the OASF schema endpoint to provision/serve the taxonomy from.
	// Empty disables the extractor controller.
	OASFURL string `json:"oasf_url,omitempty" mapstructure:"oasf_url"`
	// ModelName is the embedding model to provision (default all-MiniLM-L6-v2;
	// any cybertron-convertible BERT model).
	ModelName string `json:"model_name,omitempty" mapstructure:"model_name"`
	// AssetDir is the local directory for provisioned assets (default
	// ~/.agntcy/oasf-sdk/extractor/).
	AssetDir string `json:"asset_dir,omitempty" mapstructure:"asset_dir"`

	// Scoring overrides; a zero value keeps the library default. The semantic and
	// lexical weights are a normalized pair — set BOTH of a pair to override it;
	// setting only one is ignored (keeps the default pair).
	SkillSemanticWeight  float64 `json:"skill_semantic_weight,omitempty"  mapstructure:"skill_semantic_weight"`
	SkillLexicalWeight   float64 `json:"skill_lexical_weight,omitempty"   mapstructure:"skill_lexical_weight"`
	DomainSemanticWeight float64 `json:"domain_semantic_weight,omitempty" mapstructure:"domain_semantic_weight"`
	DomainLexicalWeight  float64 `json:"domain_lexical_weight,omitempty"  mapstructure:"domain_lexical_weight"`
	Tiers                int     `json:"tiers,omitempty"                  mapstructure:"tiers"`
	TierRatio            float64 `json:"tier_ratio,omitempty"             mapstructure:"tier_ratio"`
	MinScore             float64 `json:"min_score,omitempty"              mapstructure:"min_score"`
}

func LoadConfig() (*Config, error) {
	v := viper.NewWithOptions(
		viper.KeyDelimiter("."),
		viper.EnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_")),
	)

	v.SetEnvPrefix(DefaultEnvPrefix)
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	_ = v.BindEnv("listen_address")
	v.SetDefault("listen_address", DefaultListenAddress)

	for _, key := range []string{
		"extractor.oasf_url",
		"extractor.model_name",
		"extractor.asset_dir",
		"extractor.skill_semantic_weight",
		"extractor.skill_lexical_weight",
		"extractor.domain_semantic_weight",
		"extractor.domain_lexical_weight",
		"extractor.tiers",
		"extractor.tier_ratio",
		"extractor.min_score",
	} {
		_ = v.BindEnv(key)
	}

	decodeHooks := mapstructure.ComposeDecodeHookFunc(
		mapstructure.TextUnmarshallerHookFunc(),
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)

	config := &Config{}
	if err := v.Unmarshal(config, viper.DecodeHook(decodeHooks)); err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return config, nil
}
