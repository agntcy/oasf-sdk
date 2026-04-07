// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"testing"

	schemav1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/schema/v1"
	"github.com/agntcy/oasf-sdk/pkg/schema"
)

func TestParseSchemaURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		rawURL          string
		wantBase        string
		wantVersion     string
		wantEntityType  schema.EntityType
		wantSchemaName  string
		wantErrContains string
	}{
		{
			name:           "valid objects URL",
			rawURL:         "https://schema.oasf.outshift.com/schema/1.0.0/objects/record",
			wantBase:       "https://schema.oasf.outshift.com",
			wantVersion:    "1.0.0",
			wantEntityType: schema.EntityTypeObjects,
			wantSchemaName: "record",
		},
		{
			name:           "valid modules URL",
			rawURL:         "https://schema.oasf.outshift.com/schema/0.8.0/modules/agentskills",
			wantBase:       "https://schema.oasf.outshift.com",
			wantVersion:    "0.8.0",
			wantEntityType: schema.EntityTypeModules,
			wantSchemaName: "agentskills",
		},
		{
			name:           "valid skills URL",
			rawURL:         "https://schema.oasf.outshift.com/schema/0.7.0/skills/nlp",
			wantBase:       "https://schema.oasf.outshift.com",
			wantVersion:    "0.7.0",
			wantEntityType: schema.EntityTypeSkills,
			wantSchemaName: "nlp",
		},
		{
			name:            "missing name segment",
			rawURL:          "https://schema.oasf.outshift.com/schema/1.0.0/objects",
			wantErrContains: "must follow /schema/<version>/<type>/<name>",
		},
		{
			name:            "path does not start with schema",
			rawURL:          "https://schema.oasf.outshift.com/api/1.0.0/objects/record",
			wantErrContains: "must follow /schema/<version>/<type>/<name>",
		},
		{
			name:            "empty URL",
			rawURL:          "",
			wantErrContains: "must follow /schema/<version>/<type>/<name>",
		},
		{
			name:            "invalid URL",
			rawURL:          "://not a url",
			wantErrContains: "invalid url",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base, version, entityType, name, err := parseSchemaURL(tc.rawURL)

			if tc.wantErrContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrContains)
				}

				if !containsStr(err.Error(), tc.wantErrContains) {
					t.Fatalf("expected error containing %q, got %q", tc.wantErrContains, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if base != tc.wantBase {
				t.Errorf("base: got %q, want %q", base, tc.wantBase)
			}

			if version != tc.wantVersion {
				t.Errorf("version: got %q, want %q", version, tc.wantVersion)
			}

			if entityType != tc.wantEntityType {
				t.Errorf("entityType: got %q, want %q", entityType, tc.wantEntityType)
			}

			if name != tc.wantSchemaName {
				t.Errorf("name: got %q, want %q", name, tc.wantSchemaName)
			}
		})
	}
}

func TestTaxonomyItemToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  schema.TaxonomyItem
		want *schemav1.TaxonomyItem
	}{
		{
			name: "leaf node",
			src: schema.TaxonomyItem{
				ID:          42,
				Name:        "nlp",
				Caption:     "Natural Language Processing",
				Description: "NLP skills",
				Category:    true,
				Deprecated:  false,
			},
			want: &schemav1.TaxonomyItem{
				Id:          42,
				Name:        "nlp",
				Caption:     "Natural Language Processing",
				Description: "NLP skills",
				Category:    true,
				Deprecated:  false,
				Classes:     nil,
			},
		},
		{
			name: "node with children",
			src: schema.TaxonomyItem{
				ID:      1,
				Name:    "ai",
				Caption: "Artificial Intelligence",
				Classes: map[string]schema.TaxonomyItem{
					"nlp": {
						ID:      2,
						Name:    "nlp",
						Caption: "NLP",
					},
				},
			},
			want: &schemav1.TaxonomyItem{
				Id:      1,
				Name:    "ai",
				Caption: "Artificial Intelligence",
				Classes: map[string]*schemav1.TaxonomyItem{
					"nlp": {
						Id:      2,
						Name:    "nlp",
						Caption: "NLP",
					},
				},
			},
		},
		{
			name: "deprecated item",
			src: schema.TaxonomyItem{
				ID:         99,
				Name:       "old_skill",
				Deprecated: true,
			},
			want: &schemav1.TaxonomyItem{
				Id:         99,
				Name:       "old_skill",
				Deprecated: true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assertTaxonomyItemEqual(t, tc.want, taxonomyItemToProto(tc.src))
		})
	}
}

// assertTaxonomyItemEqual compares two proto TaxonomyItems field by field,
// including a shallow check of their Classes children.
func assertTaxonomyItemEqual(t *testing.T, want, got *schemav1.TaxonomyItem) {
	t.Helper()

	if got.GetId() != want.GetId() {
		t.Errorf("Id: got %d, want %d", got.GetId(), want.GetId())
	}

	if got.GetName() != want.GetName() {
		t.Errorf("Name: got %q, want %q", got.GetName(), want.GetName())
	}

	if got.GetCaption() != want.GetCaption() {
		t.Errorf("Caption: got %q, want %q", got.GetCaption(), want.GetCaption())
	}

	if got.GetDescription() != want.GetDescription() {
		t.Errorf("Description: got %q, want %q", got.GetDescription(), want.GetDescription())
	}

	if got.GetCategory() != want.GetCategory() {
		t.Errorf("Category: got %v, want %v", got.GetCategory(), want.GetCategory())
	}

	if got.GetDeprecated() != want.GetDeprecated() {
		t.Errorf("Deprecated: got %v, want %v", got.GetDeprecated(), want.GetDeprecated())
	}

	assertClassesEqual(t, want.GetClasses(), got.GetClasses())
}

// assertClassesEqual checks that two Classes maps have matching keys and scalar fields.
func assertClassesEqual(t *testing.T, want, got map[string]*schemav1.TaxonomyItem) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("Classes length: got %d, want %d", len(got), len(want))
	}

	for key, wantChild := range want {
		gotChild, ok := got[key]
		if !ok {
			t.Errorf("missing child key %q in Classes", key)

			continue
		}

		if gotChild.GetId() != wantChild.GetId() {
			t.Errorf("Classes[%q].Id: got %d, want %d", key, gotChild.GetId(), wantChild.GetId())
		}

		if gotChild.GetName() != wantChild.GetName() {
			t.Errorf("Classes[%q].Name: got %q, want %q", key, gotChild.GetName(), wantChild.GetName())
		}

		if gotChild.GetCaption() != wantChild.GetCaption() {
			t.Errorf("Classes[%q].Caption: got %q, want %q", key, gotChild.GetCaption(), wantChild.GetCaption())
		}
	}
}

func TestTaxonomyToProto(t *testing.T) {
	t.Parallel()

	t.Run("nil input returns nil", func(t *testing.T) {
		t.Parallel()

		if got := taxonomyToProto(nil); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("non-nil input maps all entries", func(t *testing.T) {
		t.Parallel()

		src := schema.Taxonomy{
			"skill_a": {ID: 1, Name: "skill_a"},
			"skill_b": {ID: 2, Name: "skill_b"},
		}

		got := taxonomyToProto(src)

		if len(got) != len(src) {
			t.Fatalf("length: got %d, want %d", len(got), len(src))
		}

		for key, item := range src {
			protoItem, ok := got[key]
			if !ok {
				t.Errorf("missing key %q", key)

				continue
			}

			if int(protoItem.GetId()) != item.ID {
				t.Errorf("[%q] Id: got %d, want %d", key, protoItem.GetId(), item.ID)
			}

			if protoItem.GetName() != item.Name {
				t.Errorf("[%q] Name: got %q, want %q", key, protoItem.GetName(), item.Name)
			}
		}
	})
}

// containsStr reports whether substr is within s.
func containsStr(s, substr string) bool {
	for i := range len(s) - len(substr) + 1 {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
