// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"
)

// AI Catalog media types advertised by records carrying a given OASF
// integration module.
const (
	A2ACatalogMediaType         = "application/a2a-agent-card+json"
	MCPCatalogMediaType         = "application/mcp-server+json"
	AgentSkillsCatalogMediaType = "application/ai-skill+md"

	// CatalogContainerMediaType is the media type used for a container
	// entry: a record that bundles more than one known module.
	CatalogContainerMediaType = "application/ai-catalog+json"

	// DefaultCatalogURNHost is the authority segment used in entry URNs
	// ("urn:ai:<host>:cid:<cid>[:<suffix>]") when no host is supplied via
	// WithCatalogHost.
	DefaultCatalogURNHost = "org.agntcy"

	// DefaultCatalogSpecVersion is the AI Catalog spec version embedded in
	// the nested catalog produced for multi-module records.
	DefaultCatalogSpecVersion = "1.0"
)

// catalogModuleProjection captures the per-module projection rules: the
// media type a module advertises, the URN suffix used to disambiguate the
// module's entry inside a multi-module container, and a short human label
// used to synthesise nested display names.
type catalogModuleProjection struct {
	MediaType string
	URNSuffix string
	Label     string
}

// catalogModules maps OASF integration module names onto their AI Catalog
// projection rules. A record is projectable only if it carries at least
// one of these modules.
var catalogModules = map[string]catalogModuleProjection{
	MCPModuleName:         {MediaType: MCPCatalogMediaType, URNSuffix: "mcp", Label: "MCP"},
	A2AModuleName:         {MediaType: A2ACatalogMediaType, URNSuffix: "a2a", Label: "A2A"},
	AgentSkillsModuleName: {MediaType: AgentSkillsCatalogMediaType, URNSuffix: "agentskill", Label: "Skill"},
}

// CatalogOption configures RecordToCatalog.
type CatalogOption func(*catalogOptions)

type catalogOptions struct {
	host        string
	cid         string
	specVersion string
}

// WithCatalogHost sets the authority segment of the entry URN. Defaults to
// DefaultCatalogURNHost.
func WithCatalogHost(host string) CatalogOption {
	return func(o *catalogOptions) {
		if h := strings.TrimSpace(host); h != "" {
			o.host = h
		}
	}
}

// WithCatalogCID sets the content identifier used in the entry URN. The
// raw OASF record does not carry its own CID, so callers MUST supply one;
// RecordToCatalog returns an error when no CID is provided.
func WithCatalogCID(cid string) CatalogOption {
	return func(o *catalogOptions) {
		o.cid = cid
	}
}

// WithCatalogSpecVersion overrides the AI Catalog spec version embedded in
// the nested catalog of a multi-module container.
func WithCatalogSpecVersion(version string) CatalogOption {
	return func(o *catalogOptions) {
		if v := strings.TrimSpace(version); v != "" {
			o.specVersion = v
		}
	}
}

// catalogModule is a known module extracted from a record: its name and
// the structured `data` object carried alongside it.
type catalogModule struct {
	name string
	data *structpb.Struct
}

// RecordToCatalog projects an OASF record onto its AI Catalog entry
// representation, returned as a structpb.Struct. The result depends
// on how many known integration modules the record carries.
//
//   - 0 known modules → error; a catalog entry MUST point at an artifact
//     and there is nothing to project.
//   - 1 known module  → a LEAF entry whose media_type matches the module
//     (e.g. "application/a2a-agent-card+json") and whose `data` is the
//     module's structured data.
//   - 2+ known modules → a CONTAINER entry with media_type
//     "application/ai-catalog+json" whose `data` embeds a nested AI Catalog
//     with one entry per known module.
//
// The projection is deliberately pure: trust/identity metadata (e.g.
// signature-derived TrustManifests) and any publisher/host data beyond the
// URN host are layered on by the caller and intentionally not produced
// here.
func RecordToCatalog(record *structpb.Struct, opts ...CatalogOption) (*structpb.Struct, error) {
	if record == nil {
		return nil, errors.New("record is nil")
	}

	options := &catalogOptions{
		host:        DefaultCatalogURNHost,
		specVersion: DefaultCatalogSpecVersion,
	}
	for _, opt := range opts {
		opt(options)
	}

	cid := strings.TrimSpace(options.cid)
	if cid == "" {
		return nil, errors.New("catalog CID is required (set WithCatalogCID)")
	}

	modules := knownCatalogModules(record)
	if len(modules) == 0 {
		return nil, errors.New("record has no known catalog modules")
	}

	name := recordStringField(record, "name")
	version := recordStringField(record, "version")
	description := strings.TrimSpace(recordStringField(record, "description"))
	updatedAt := recordStringField(record, "created_at")
	tags := catalogTags(record)

	baseURN := catalogURN(options.host, cid, "")

	// Single known module — leaf entry on the parent URN.
	if len(modules) == 1 {
		entry := moduleToCatalogEntry(modules[0], baseURN)
		entry.Fields["display_name"] = catalogStringValue(firstNonEmptyString(name, baseURN))
		setCatalogTags(entry, tags)
		setOptionalString(entry, "version", version)
		setOptionalString(entry, "description", description)
		setOptionalString(entry, "updated_at", updatedAt)

		return entry, nil
	}

	// 2+ known modules — container whose data is a nested AI Catalog.
	nested := make([]*structpb.Value, 0, len(modules))

	for _, m := range modules {
		moduleURN := catalogURN(options.host, cid, catalogModules[m.name].URNSuffix)

		entry := moduleToCatalogEntry(m, moduleURN)
		entry.Fields["display_name"] = catalogStringValue(moduleDisplayName(m, name))

		nested = append(nested, structpb.NewStructValue(entry))
	}

	nestedCatalog := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"spec_version": catalogStringValue(options.specVersion),
			"entries": {
				Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: nested}},
			},
		},
	}

	container := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"identifier":   catalogStringValue(baseURN),
			"display_name": catalogStringValue(firstNonEmptyString(name, baseURN)),
			"media_type":   catalogStringValue(CatalogContainerMediaType),
			"data":         structpb.NewStructValue(nestedCatalog),
		},
	}
	setCatalogTags(container, tags)
	setOptionalString(container, "version", version)
	setOptionalString(container, "description", description)
	setOptionalString(container, "updated_at", updatedAt)

	return container, nil
}

// moduleToCatalogEntry builds a leaf catalog entry for a single module.
//
// A catalog entry requires exactly one of `url` or `data`. We always carry
// the module's structured data inline via `data` in OASF.
func moduleToCatalogEntry(m catalogModule, identifier string) *structpb.Struct {
	proj := catalogModules[m.name]

	data := m.data
	if data == nil {
		data = &structpb.Struct{Fields: map[string]*structpb.Value{}}
	}

	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"identifier": catalogStringValue(identifier),
			"media_type": catalogStringValue(proj.MediaType),
			"data":       structpb.NewStructValue(data),
		},
	}
}

// knownCatalogModules returns the record's modules that have a catalog
// projection rule, sorted by name for deterministic output.
func knownCatalogModules(record *structpb.Struct) []catalogModule {
	modulesVal, ok := record.GetFields()["modules"]
	if !ok {
		return nil
	}

	list := modulesVal.GetListValue()
	if list == nil {
		return nil
	}

	out := make([]catalogModule, 0, len(list.GetValues()))

	for _, v := range list.GetValues() {
		moduleStruct := v.GetStructValue()
		if moduleStruct == nil {
			continue
		}

		nameField := moduleStruct.GetFields()["name"]
		if nameField == nil {
			continue
		}

		name := nameField.GetStringValue()
		if _, known := catalogModules[name]; !known {
			continue
		}

		out = append(out, catalogModule{
			name: name,
			data: moduleStruct.GetFields()["data"].GetStructValue(),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })

	return out
}

// catalogTags assembles the OASF-taxonomy + annotation tag list shared by
// leaf and container entries: one tag per skill and domain
// ("oasf:v<schema>:skills:<name>" / "oasf:v<schema>:domains:<name>"),
// followed by record annotations ("key" or "key=value").
func catalogTags(record *structpb.Struct) []string {
	schemaVer := strings.TrimPrefix(recordStringField(record, "schema_version"), "v")
	if schemaVer == "" {
		schemaVer = "1"
	}

	skills := taxonomyNames(record, "skills")
	domains := taxonomyNames(record, "domains")
	annotations := sortedAnnotations(record)

	out := make([]string, 0, len(skills)+len(domains)+len(annotations))

	for _, s := range skills {
		out = append(out, fmt.Sprintf("oasf:v%s:skills:%s", schemaVer, s))
	}

	for _, d := range domains {
		out = append(out, fmt.Sprintf("oasf:v%s:domains:%s", schemaVer, d))
	}

	out = append(out, annotations...)

	return out
}

// taxonomyNames extracts the "name" of each entry in a record's skills or
// domains list, preserving record order.
func taxonomyNames(record *structpb.Struct, field string) []string {
	val, ok := record.GetFields()[field]
	if !ok {
		return nil
	}

	list := val.GetListValue()
	if list == nil {
		return nil
	}

	out := make([]string, 0, len(list.GetValues()))

	for _, v := range list.GetValues() {
		entry := v.GetStructValue()
		if entry == nil {
			continue
		}

		if name := entry.GetFields()["name"].GetStringValue(); name != "" {
			out = append(out, name)
		}
	}

	return out
}

// sortedAnnotations renders a record's annotations map into a
// deterministic (key-sorted) list of "key" or "key=value" tags.
func sortedAnnotations(record *structpb.Struct) []string {
	val, ok := record.GetFields()["annotations"]
	if !ok {
		return nil
	}

	annotations := val.GetStructValue()
	if annotations == nil || len(annotations.GetFields()) == 0 {
		return nil
	}

	keys := make([]string, 0, len(annotations.GetFields()))
	for key := range annotations.GetFields() {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	out := make([]string, 0, len(keys))

	for _, key := range keys {
		value := annotations.GetFields()[key].GetStringValue()
		if value == "" {
			out = append(out, key)
		} else {
			out = append(out, fmt.Sprintf("%s=%s", key, value))
		}
	}

	return out
}

// moduleDisplayName synthesises the nested entry display name as
// "<record name> (<label>)", falling back gracefully when either piece is
// missing.
func moduleDisplayName(m catalogModule, recordName string) string {
	label := catalogModules[m.name].Label

	switch {
	case recordName != "" && label != "":
		return fmt.Sprintf("%s (%s)", recordName, label)
	case recordName != "":
		return recordName
	case label != "":
		return label
	default:
		return m.name
	}
}

// catalogURN builds "urn:ai:<host>:cid:<cid>" with an optional ":<suffix>"
// used to disambiguate nested entries inside a container.
func catalogURN(host, cid, suffix string) string {
	base := fmt.Sprintf("urn:ai:%s:cid:%s", host, cid)
	if suffix == "" {
		return base
	}

	return base + ":" + suffix
}

// recordStringField returns a top-level string field of the record, or "".
func recordStringField(record *structpb.Struct, field string) string {
	return record.GetFields()[field].GetStringValue()
}

// catalogStringValue wraps a string in a structpb.Value.
func catalogStringValue(s string) *structpb.Value {
	return &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: s}}
}

// setOptionalString sets a string field on an entry only when non-empty.
func setOptionalString(entry *structpb.Struct, field, value string) {
	if value != "" {
		entry.Fields[field] = catalogStringValue(value)
	}
}

// setCatalogTags sets the "tags" field on an entry only when non-empty.
func setCatalogTags(entry *structpb.Struct, tags []string) {
	if len(tags) == 0 {
		return
	}

	values := make([]*structpb.Value, 0, len(tags))
	for _, t := range tags {
		values = append(values, catalogStringValue(t))
	}

	entry.Fields["tags"] = &structpb.Value{
		Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: values}},
	}
}

// firstNonEmptyString returns the first non-empty string, or "".
func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
