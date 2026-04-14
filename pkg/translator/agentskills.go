// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"errors"
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"
	"time"

	recordutil "github.com/agntcy/oasf-sdk/pkg/record"
	"google.golang.org/protobuf/types/known/structpb"
)

const agentSkillsMediaType = "application/agent-skills+md"

const (
	AgentSkillsModuleName = "core/language_model/agentskills"
	agentSkillsModuleID   = 10302
)

const (
	frontmatterParts    = 3
	yamlKeyValueParts   = 2
	frontmatterMinParts = 3
)

// RecordToSkillMarkdown returns the full SKILL.md content for a record containing
// the agentskills module (name: "core/language_model/agentskills").
//
// If the module has an artifact with base64-encoded data, the original SKILL.md
// (including its body) is decoded and returned directly. Otherwise the SKILL.md
// is reconstructed from the stored manifest (frontmatter only, no body).
func RecordToSkillMarkdown(record *structpb.Struct) (string, error) {
	if record == nil {
		return "", errors.New("record is nil")
	}

	found, moduleStruct := recordutil.GetModule(record, AgentSkillsModuleName)
	if !found || moduleStruct == nil {
		return "", errors.New("agentskills module not found in record")
	}

	// Prefer returning the full original SKILL.md from the artifact when available.
	if raw := artifactDataBytes(moduleStruct); raw != nil {
		return string(raw), nil
	}

	moduleData := moduleStruct.GetFields()["data"].GetStructValue()

	return buildSkillMarkdownFromManifest(moduleData)
}

// buildSkillMarkdownFromManifest reconstructs a SKILL.md frontmatter string from the
// skill_manifest stored in the agentskills module data. Used as fallback when no artifact is present.
func buildSkillMarkdownFromManifest(moduleData *structpb.Struct) (string, error) {
	manifestField := moduleData.GetFields()["skill_manifest"]
	if manifestField == nil {
		return "", errors.New("skill_manifest is missing")
	}

	manifest := manifestField.GetStructValue()
	if manifest == nil {
		return "", errors.New("skill_manifest is not a struct")
	}

	manifestMap := manifest.AsMap()
	name := getString(manifestMap, "name")
	description := getString(manifestMap, "description")

	if name == "" || description == "" {
		return "", errors.New("manifest must include name and description")
	}

	license := getString(manifestMap, "license")
	version := getString(manifestMap, "version")
	compatibilityItems := getStringSlice(manifestMap, "compatibility")
	compatibility := strings.Join(compatibilityItems, ", ")
	allowedTools := getStringSlice(manifestMap, "allowed_tools")
	metadata := getStringMap(manifestMap, "frontmatter_metadata")

	// Ensure version appears in metadata (not as a top-level frontmatter key per spec).
	if version != "" {
		if _, ok := metadata["version"]; !ok {
			metadata["version"] = version
		}
	}

	return renderSkillMarkdownFrontmatter(name, description, license, compatibility, allowedTools, metadata), nil
}

// renderSkillMarkdownFrontmatter renders the YAML frontmatter block for a SKILL.md file.
func renderSkillMarkdownFrontmatter(name, description, license, compatibility string, allowedTools []string, metadata map[string]string) string {
	lines := []string{"---"}
	lines = append(lines, "name: "+yamlScalar(name))
	lines = append(lines, "description: "+yamlScalar(description))

	if license != "" {
		lines = append(lines, "license: "+yamlScalar(license))
	}

	if compatibility != "" {
		lines = append(lines, "compatibility: "+yamlScalar(compatibility))
	}

	if len(allowedTools) > 0 {
		lines = append(lines, "allowed-tools: "+strings.Join(allowedTools, " "))
	}

	if len(metadata) > 0 {
		lines = append(lines, "metadata:")

		keys := make([]string, 0, len(metadata))
		for key := range metadata {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		for _, key := range keys {
			lines = append(lines, "  "+key+": "+yamlScalar(metadata[key]))
		}
	}

	lines = append(lines, "---")
	lines = append(lines, "")

	return strings.Join(lines, "\n") + "\n"
}

// SkillMarkdownToRecord converts a SKILL.md file content into an OASF-compliant record.
// The input must be wrapped as {"skillMarkdown": "<content>"}.
// Generates records using the specified schema version (via WithVersion option) or the default schema version.
func SkillMarkdownToRecord(skillData *structpb.Struct, opts ...TranslatorOption) (*structpb.Struct, error) {
	mdVal, ok := skillData.GetFields()["skillMarkdown"]
	if !ok {
		return nil, errors.New("missing 'skillMarkdown' in input data")
	}

	content := mdVal.GetStringValue()
	if content == "" {
		return nil, errors.New("'skillMarkdown' is empty")
	}

	parsed, err := parseSkillMarkdownContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md content: %w", err)
	}

	// version is not a spec frontmatter field; source it from metadata["version"].
	version := parsed.metadata["version"]

	recordVersion := defaultVersion
	if version != "" {
		recordVersion = version
	}

	authors := buildAuthors(parsed.metadata)
	manifestFields := buildManifestFields(parsed, version)
	moduleDataFields := buildModuleDataFields(manifestFields)
	artifactStruct := buildArtifactDescriptor([]byte(content), agentSkillsMediaType)
	agentSkillsModule := buildAgentSkillsModule(moduleDataFields, artifactStruct)

	options := &translatorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	targetVersion := DefaultSchemaVersion

	if options.version != "" {
		if err := validateMajorVersion(options.version); err != nil {
			return nil, err
		}

		targetVersion = options.version
	}

	return buildRecord(parsed.name, parsed.description, targetVersion, recordVersion, authors, agentSkillsModule), nil
}

func buildAuthors(metadata map[string]string) []*structpb.Value {
	if author, ok := metadata["author"]; ok && author != "" {
		return []*structpb.Value{
			{Kind: &structpb.Value_StringValue{StringValue: author}},
		}
	}

	return []*structpb.Value{}
}

func buildManifestFields(parsed skillMarkdownFields, version string) map[string]*structpb.Value {
	fields := map[string]*structpb.Value{
		"name":        {Kind: &structpb.Value_StringValue{StringValue: parsed.name}},
		"description": {Kind: &structpb.Value_StringValue{StringValue: parsed.description}},
	}

	if parsed.license != "" {
		fields["license"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: parsed.license}}
	}

	if version != "" {
		fields["version"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: version}}
	}

	if parsed.compatibility != "" {
		fields["compatibility"] = &structpb.Value{
			Kind: &structpb.Value_ListValue{
				ListValue: &structpb.ListValue{
					Values: []*structpb.Value{
						{Kind: &structpb.Value_StringValue{StringValue: parsed.compatibility}},
					},
				},
			},
		}
	}

	if len(parsed.allowedTools) > 0 {
		toolValues := make([]*structpb.Value, 0, len(parsed.allowedTools))
		for _, tool := range parsed.allowedTools {
			toolValues = append(toolValues, &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: tool}})
		}

		fields["allowed_tools"] = &structpb.Value{
			Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: toolValues}},
		}
	}

	if len(parsed.metadata) > 0 {
		metaFields := make(map[string]*structpb.Value, len(parsed.metadata))
		for k, v := range parsed.metadata {
			metaFields[k] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: v}}
		}

		fields["frontmatter_metadata"] = &structpb.Value{
			Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: metaFields}},
		}
	}

	return fields
}

func buildModuleDataFields(manifestFields map[string]*structpb.Value) map[string]*structpb.Value {
	return map[string]*structpb.Value{
		"skill_file": {Kind: &structpb.Value_StringValue{StringValue: "SKILL.md"}},
		"skill_manifest": {
			Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: manifestFields}},
		},
	}
}

func buildAgentSkillsModule(moduleDataFields map[string]*structpb.Value, artifactStruct *structpb.Struct) *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name":     {Kind: &structpb.Value_StringValue{StringValue: AgentSkillsModuleName}},
			"id":       {Kind: &structpb.Value_NumberValue{NumberValue: agentSkillsModuleID}},
			"data":     {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: moduleDataFields}}},
			"artifact": {Kind: &structpb.Value_StructValue{StructValue: artifactStruct}},
		},
	}
}

func buildRecord(name, description, schemaVersion, version string, authors []*structpb.Value, module *structpb.Struct) *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name":           {Kind: &structpb.Value_StringValue{StringValue: name}},
			"schema_version": {Kind: &structpb.Value_StringValue{StringValue: schemaVersion}},
			"version":        {Kind: &structpb.Value_StringValue{StringValue: version}},
			"description":    {Kind: &structpb.Value_StringValue{StringValue: description}},
			"authors": {
				Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: authors}},
			},
			"created_at": {Kind: &structpb.Value_StringValue{StringValue: time.Now().UTC().Format(time.RFC3339)}},
			"skills": {
				Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: []*structpb.Value{}}},
			},
			"domains": {
				Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: []*structpb.Value{}}},
			},
			"modules": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{
						Values: []*structpb.Value{
							{Kind: &structpb.Value_StructValue{StructValue: module}},
						},
					},
				},
			},
		},
	}
}

// skillMarkdownFields holds the parsed frontmatter fields from a SKILL.md file.
type skillMarkdownFields struct {
	name          string
	description   string
	license       string
	compatibility string
	allowedTools  []string
	metadata      map[string]string
}

// parseSkillMarkdownContent parses a SKILL.md and returns its spec-defined frontmatter fields.
// Spec frontmatter fields: name, description, license, compatibility, allowed-tools, metadata.
// There is no top-level version field in the spec; version lives inside metadata.
func parseSkillMarkdownContent(content string) (skillMarkdownFields, error) {
	sections := strings.SplitN(content, "---", frontmatterParts)
	if len(sections) < frontmatterMinParts {
		return skillMarkdownFields{}, errors.New("invalid SKILL.md: missing frontmatter delimiters")
	}

	frontmatter := strings.TrimSpace(sections[1])

	result := skillMarkdownFields{
		metadata: map[string]string{},
	}

	lines := strings.Split(frontmatter, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		if line == "metadata:" {
			for i+1 < len(lines) {
				next := lines[i+1]
				if !strings.HasPrefix(next, "  ") {
					break
				}

				i++
				k, v := splitYAMLKeyValue(strings.TrimSpace(next))
				result.metadata[k] = v
			}

			continue
		}

		k, v := splitYAMLKeyValue(line)
		switch k {
		case "name":
			result.name = v
		case "description":
			result.description = v
		case "license":
			result.license = v
		case "compatibility":
			result.compatibility = v
		case "allowed-tools":
			result.allowedTools = strings.Fields(v)
		}
	}

	if result.name == "" || result.description == "" {
		return skillMarkdownFields{}, errors.New("SKILL.md must include name and description in frontmatter")
	}

	return result, nil
}

// splitYAMLKeyValue splits a YAML "key: value" line and unquotes quoted values.
func splitYAMLKeyValue(line string) (string, string) {
	parts := strings.SplitN(line, ":", yamlKeyValueParts)
	key := strings.TrimSpace(parts[0])
	value := ""

	if len(parts) > 1 {
		value = strings.TrimSpace(parts[1])
	}

	if strings.HasPrefix(value, "\"") {
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
	}

	return key, value
}

func getString(data map[string]any, key string) string {
	if value, ok := data[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}

	return ""
}

func getStringSlice(data map[string]any, key string) []string {
	value, ok := data[key]
	if !ok {
		return nil
	}

	switch typed := value.(type) {
	case []string:
		return append([]string{}, typed...)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}

		return result
	default:
		return nil
	}
}

func getStringMap(data map[string]any, key string) map[string]string {
	value, ok := data[key]
	if !ok {
		return map[string]string{}
	}

	metadata := map[string]string{}
	if rawMap, ok := value.(map[string]any); ok {
		maps.Copy(metadata, toStringMap(rawMap))

		return metadata
	}

	if rawMap, ok := value.(map[string]string); ok {
		maps.Copy(metadata, rawMap)
	}

	return metadata
}

func toStringMap(rawMap map[string]any) map[string]string {
	metadata := make(map[string]string, len(rawMap))
	for k, v := range rawMap {
		metadata[k] = fmt.Sprint(v)
	}

	return metadata
}

func yamlScalar(value string) string {
	if value == "" {
		return "\"\""
	}

	if strings.ContainsAny(value, ":#\n") || strings.HasPrefix(value, " ") || strings.HasSuffix(value, " ") {
		return strconv.Quote(value)
	}

	return value
}
