// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	extractorv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/extractor/v1"
	"github.com/agntcy/oasf-sdk/pkg/extractor"
)

func TestKindToClassType(t *testing.T) {
	cases := map[extractor.Kind]extractorv1.ClassType{
		extractor.KindSkill:  extractorv1.ClassType_CLASS_TYPE_SKILL,
		extractor.KindDomain: extractorv1.ClassType_CLASS_TYPE_DOMAIN,
		extractor.KindModule: extractorv1.ClassType_CLASS_TYPE_MODULE,
		extractor.Kind("?"):  extractorv1.ClassType_CLASS_TYPE_UNSPECIFIED,
	}

	for k, want := range cases {
		if got := kindToClassType(k); got != want {
			t.Errorf("kindToClassType(%q) = %v, want %v", k, got, want)
		}
	}
}

func TestToScoredClasses(t *testing.T) {
	in := []extractor.ScoredClass{{
		Class:    extractor.Class{ID: 101, Name: "a/b", Caption: "A/B", Description: "desc"},
		Kind:     extractor.KindSkill,
		Versions: []string{"1.0.0"},
		Score:    0.9,
		Semantic: 0.8,
		Lexical:  0.4,
		Tier:     1,
	}}

	out := toScoredClasses(in)
	if len(out) != 1 {
		t.Fatalf("len = %d, want 1", len(out))
	}

	sc := out[0]
	if sc.GetId() != 101 || sc.GetName() != "a/b" || sc.GetCaption() != "A/B" ||
		sc.GetDescription() != "desc" || sc.GetKind() != extractorv1.ClassType_CLASS_TYPE_SKILL ||
		len(sc.GetVersions()) != 1 || sc.GetVersions()[0] != "1.0.0" ||
		sc.GetScore() != 0.9 || sc.GetSemantic() != 0.8 || sc.GetLexical() != 0.4 ||
		sc.GetTier() != 1 {
		t.Fatalf("unexpected mapping: %+v", sc)
	}
}

func TestToKeywords(t *testing.T) {
	out := toKeywords([]extractor.Keyword{{Text: "fraud", Score: 3}})
	if len(out) != 1 || out[0].GetText() != "fraud" || out[0].GetScore() != 3 {
		t.Fatalf("unexpected mapping: %+v", out)
	}
}

// TestQueryOptions asserts the request→option rules by option count: exactly one
// version-scoping option is always produced (an explicit version list overrides
// the scope rather than adding a second), and zero-valued numeric fields add no
// option (they fall back to the extractor defaults). The options themselves are
// opaque funcs, so this checks the selection/gating rather than their effect.
func TestQueryOptions(t *testing.T) {
	cases := []struct {
		name string
		req  *extractorv1.ExtractRequest
		want int
	}{
		{"default scope only", &extractorv1.ExtractRequest{}, 1},
		{"latest scope", &extractorv1.ExtractRequest{Scope: extractorv1.VersionScope_VERSION_SCOPE_LATEST}, 1},
		{"explicit versions override scope", &extractorv1.ExtractRequest{Versions: []string{"1.0.0"}, Scope: extractorv1.VersionScope_VERSION_SCOPE_LATEST}, 1},
		{"zero numeric fields ignored", &extractorv1.ExtractRequest{Tiers: 0, MinScore: 0, MinResults: 0}, 1},
		{"tiers only", &extractorv1.ExtractRequest{Tiers: 2}, 2},
		{"all knobs set", &extractorv1.ExtractRequest{Versions: []string{"1.0.0"}, Tiers: 1, MinScore: 0.1, MinResults: 3}, 4},
	}

	for _, tc := range cases {
		if got := len(queryOptions(tc.req)); got != tc.want {
			t.Errorf("%s: len(queryOptions) = %d, want %d", tc.name, got, tc.want)
		}
	}
}
