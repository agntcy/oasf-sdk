// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"testing"

	"github.com/agntcy/oasf-sdk/server/config"
)

// TestExtractorOptions checks how ExtractorConfig maps to extractor options by
// count: each set string/number adds one option, and a weight PAIR is applied
// only when BOTH of the pair are set (a lone weight is ignored so it can't
// force the other to 0).
func TestExtractorOptions(t *testing.T) {
	cases := []struct {
		name string
		ex   config.ExtractorConfig
		want int
	}{
		{"url only", config.ExtractorConfig{OASFURL: "x"}, 1},
		{"url + model", config.ExtractorConfig{OASFURL: "x", ModelName: "m"}, 2},
		{"url + asset dir", config.ExtractorConfig{OASFURL: "x", AssetDir: "/d"}, 2},
		{"both skill weights", config.ExtractorConfig{OASFURL: "x", SkillSemanticWeight: 0.6, SkillLexicalWeight: 0.4}, 2},
		{"lone skill weight ignored", config.ExtractorConfig{OASFURL: "x", SkillSemanticWeight: 0.7}, 1},
		{"both domain weights", config.ExtractorConfig{OASFURL: "x", DomainSemanticWeight: 0.8, DomainLexicalWeight: 0.2}, 2},
		{"lone domain weight ignored", config.ExtractorConfig{OASFURL: "x", DomainLexicalWeight: 0.2}, 1},
		{"tiers + ratio + min score", config.ExtractorConfig{OASFURL: "x", Tiers: 1, TierRatio: 0.9, MinScore: 0.1}, 4},
	}

	for _, tc := range cases {
		cfg := &config.Config{Extractor: tc.ex}
		if got := len(extractorOptions(cfg)); got != tc.want {
			t.Errorf("%s: len(extractorOptions) = %d, want %d", tc.name, got, tc.want)
		}
	}
}

// TestNewServerWithoutExtractor verifies the server builds and registers the
// stateless controllers when no OASF URL is configured — i.e. the extractor is
// skipped, so no provisioning or model load is attempted.
func TestNewServerWithoutExtractor(t *testing.T) {
	srv, err := NewServer(context.Background(), &config.Config{ListenAddress: config.DefaultListenAddress})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	if srv == nil || srv.grpcServer == nil {
		t.Fatal("expected a server with an initialized gRPC server")
	}
}
