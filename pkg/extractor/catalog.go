// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package extractor

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"github.com/agntcy/oasf-sdk/pkg/schema"
)

// SupportedVersions returns the OASF schema versions this Extractor was
// provisioned against, in ascending order.
func (r *Extractor) SupportedVersions() []string {
	out := make([]string, len(r.versions))
	copy(out, r.versions)

	return out
}

// LatestVersion returns the newest OASF schema version this Extractor covers.
func (r *Extractor) LatestVersion() string {
	if len(r.versions) == 0 {
		return ""
	}

	return r.versions[len(r.versions)-1]
}

// IsSupported reports whether this Extractor covers the given OASF version.
func (r *Extractor) IsSupported(version string) bool {
	return slices.Contains(r.versions, version)
}

// flattenTaxonomy walks the nested OASF taxonomy tree and returns the flat list
// of concrete classes, dropping category grouping nodes, deprecated classes, and
// the abstract id-0 base class. TaxonomyItem.Name is already the full
// hierarchical path (e.g. "nlp/nlu/sentiment"), so no path joining is needed.
// The result is sorted by ID for deterministic embedding and tie-breaks.
func flattenTaxonomy(t schema.Taxonomy) []Class {
	var out []Class

	var walk func(items map[string]schema.TaxonomyItem)

	walk = func(items map[string]schema.TaxonomyItem) {
		for _, it := range items {
			if !it.Category && !it.Deprecated && it.ID != 0 {
				out = append(out, Class{
					ID:          uint64(it.ID), //nolint:gosec // TaxonomyItem.ID is always a positive OASF class id that fits in uint64.
					Name:        it.Name,
					Caption:     it.Caption,
					Description: it.Description,
				})
			}

			if len(it.Classes) > 0 {
				walk(it.Classes)
			}
		}
	}

	walk(map[string]schema.TaxonomyItem(t))

	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })

	return out
}

// kindEndpointKnown maps a Kind to the human label used in error messages.
func kindEndpointKnown(kind Kind) string {
	if kind == KindDomain {
		return "domains"
	}

	return "skills"
}

// fetchClasses fetches the taxonomy for one version and kind from the OASF
// endpoint and flattens it to the extractor's Class list.
func fetchClasses(ctx context.Context, sc *schema.Schema, version string, kind Kind) ([]Class, error) {
	var (
		tax schema.Taxonomy
		err error
	)

	opt := schema.WithSchemaVersion(version)
	if kind == KindDomain {
		tax, err = sc.GetSchemaDomains(ctx, opt)
	} else {
		tax, err = sc.GetSchemaSkills(ctx, opt)
	}

	if err != nil {
		return nil, fmt.Errorf("fetch %s for %s: %w", kindEndpointKnown(kind), version, err)
	}

	return flattenTaxonomy(tax), nil
}
