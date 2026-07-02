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
