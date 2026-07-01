// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"reflect"
	"testing"

	"github.com/agntcy/oasf-sdk/pkg/schema"
)

func TestFetchClassesSelectsKind(t *testing.T) {
	if kindEndpointKnown(KindSkill) != "skills" || kindEndpointKnown(KindDomain) != "domains" {
		t.Fatalf("kind mapping wrong")
	}
}

func TestFlattenTaxonomy(t *testing.T) {
	tax := schema.Taxonomy{
		"nlp": {
			ID: 1, Name: "nlp", Caption: "NLP", Category: true,
			Classes: map[string]schema.TaxonomyItem{
				"nlu": {
					ID: 101, Name: "nlp/nlu", Caption: "NLU", Description: "understand",
					Classes: map[string]schema.TaxonomyItem{
						"sentiment": {ID: 10101, Name: "nlp/nlu/sentiment", Caption: "Sentiment", Description: "mood"},
					},
				},
				"old": {ID: 102, Name: "nlp/old", Caption: "Old", Deprecated: true},
			},
		},
		"base": {ID: 0, Name: "base_skill", Caption: "Skill", Category: false},
	}

	got := flattenTaxonomy(tax)

	want := []Class{
		{ID: 101, Name: "nlp/nlu", Caption: "NLU", Description: "understand"},
		{ID: 10101, Name: "nlp/nlu/sentiment", Caption: "Sentiment", Description: "mood"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flattenTaxonomy() =\n%#v\nwant\n%#v", got, want)
	}
}
