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

const AgentSkillsModuleName = "agentskills"

// RecordToSkillMarkdown renders a spec-compliant SKILL.md from a record containing
// the agentskills module. The body is read from the skill_body field in the module data.
//
// The SKILL.md frontmatter contains exactly the fields defined by the Agent Skills
// specification: name, description, license, compatibility, metadata, and allowed-tools.
// There is no top-level version field in the spec; version belongs inside metadata.
func RecordToSkillMarkdown(record *structpb.Struct) (string, error) {
	if record == nil {
		return "", errors.New("record is nil")
	}

	found, moduleData := recordutil.GetModuleData(record, AgentSkillsModuleName)
	if !found || moduleData == nil {
		return "", errors.New("agentskills module not found in record")
	}

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
	compatibility := getString(manifestMap, "compatibility")
	allowedTools := getStringSlice(manifestMap, "allowed_tools")
	metadata := getStringMap(manifestMap, "frontmatter_metadata")

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

	// Read body from skill_body in module data.
	body := strings.TrimSpace(moduleData.GetFields()["skill_body"].GetStringValue())
	if body != "" {
		lines = append(lines, "", body)
	} else {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n") + "\n", nil
}

// SkillMarkdownToRecord converts a SKILL.md file content into an OASF-compliant record.
// The input must be wrapped as {"skillMarkdown": "<content>"}.
// Generates records using the specified schema version (via WithVersion option) or the default schema version.
//
// The spec-defined frontmatter fields are: name, description, license, compatibility,
// metadata, and allowed-tools. There is no top-level version field; version should live
// inside metadata if needed.
func SkillMarkdownToRecord(skillData *structpb.Struct, opts ...TranslatorOption) (*structpb.Struct, error) {
	mdVal, ok := skillData.GetFields()["skillMarkdown"]
	if !ok {
		return nil, errors.New("missing 'skillMarkdown' in input data")
	}

	content := mdVal.GetStringValue()
	if content == "" {
		return nil, errors.New("'skillMarkdown' is empty")
	}

	name, description, license, compatibility, allowedTools, metadata, body, err := parseSkillMarkdownContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md content: %w", err)
	}

	// Derive record-level version from metadata["version"] if present, else use default.
	recordVersion := defaultVersion
	if v, ok := metadata["version"]; ok && v != "" {
		recordVersion = v
	}

	// Derive authors from metadata["author"] if present.
	authors := []*structpb.Value{}
	if author, ok := metadata["author"]; ok && author != "" {
		authors = []*structpb.Value{
			{Kind: &structpb.Value_StringValue{StringValue: author}},
		}
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Build skill_manifest struct fields using only spec-defined frontmatter keys.
	manifestFields := map[string]*structpb.Value{
		"name":        {Kind: &structpb.Value_StringValue{StringValue: name}},
		"description": {Kind: &structpb.Value_StringValue{StringValue: description}},
	}

	if license != "" {
		manifestFields["license"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: license}}
	}

	if compatibility != "" {
		manifestFields["compatibility"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: compatibility}}
	}

	if len(allowedTools) > 0 {
		toolValues := make([]*structpb.Value, 0, len(allowedTools))
		for _, tool := range allowedTools {
			toolValues = append(toolValues, &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: tool}})
		}

		manifestFields["allowed_tools"] = &structpb.Value{
			Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: toolValues}},
		}
	}

	if len(metadata) > 0 {
		metaFields := make(map[string]*structpb.Value, len(metadata))
		for k, v := range metadata {
			metaFields[k] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: v}}
		}

		manifestFields["frontmatter_metadata"] = &structpb.Value{
			Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: metaFields}},
		}
	}

	// Build module data.
	moduleDataFields := map[string]*structpb.Value{
		"skill_file": {Kind: &structpb.Value_StringValue{StringValue: "SKILL.md"}},
		"skill_manifest": {
			Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: manifestFields}},
		},
	}

	if body != "" {
		moduleDataFields["skill_body"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: body}}
	}

	agentSkillsModule := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name": {Kind: &structpb.Value_StringValue{StringValue: AgentSkillsModuleName}},
			"data": {Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: moduleDataFields}}},
		},
	}

	// Determine target schema version.
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

	record := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name":           {Kind: &structpb.Value_StringValue{StringValue: name}},
			"schema_version": {Kind: &structpb.Value_StringValue{StringValue: targetVersion}},
			"version":        {Kind: &structpb.Value_StringValue{StringValue: recordVersion}},
			"description":    {Kind: &structpb.Value_StringValue{StringValue: description}},
			"authors": {
				Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: authors}},
			},
			"created_at": {Kind: &structpb.Value_StringValue{StringValue: createdAt}},
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
							{Kind: &structpb.Value_StructValue{StructValue: agentSkillsModule}},
						},
					},
				},
			},
		},
	}

	return record, nil
}

// parseSkillMarkdownContent parses a SKILL.md and returns its spec-defined components.
// Spec fields: name, description, license, compatibility, allowed-tools, metadata.
// There is no top-level version field in the spec.
func parseSkillMarkdownContent(content string) (name, description, license, compatibility string, allowedTools []string, metadata map[string]string, body string, err error) {
	sections := strings.SplitN(content, "---", 3)
	if len(sections) < 3 {
		return "", "", "", "", nil, nil, "", errors.New("invalid SKILL.md: missing frontmatter delimiters")
	}

	frontmatter := strings.TrimSpace(sections[1])
	body = strings.TrimSpace(sections[2])
	metadata = map[string]string{}

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
				metadata[k] = v
			}

			continue
		}

		k, v := splitYAMLKeyValue(line)
		switch k {
		case "name":
			name = v
		case "description":
			description = v
		case "license":
			license = v
		case "compatibility":
			compatibility = v
		case "allowed-tools":
			allowedTools = strings.Fields(v)
		}
	}

	if name == "" || description == "" {
		return "", "", "", "", nil, nil, "", errors.New("SKILL.md must include name and description in frontmatter")
	}

	return name, description, license, compatibility, allowedTools, metadata, body, nil
}

// splitYAMLKeyValue splits a YAML "key: value" line and unquotes quoted values.
func splitYAMLKeyValue(line string) (string, string) {
	parts := strings.SplitN(line, ":", 2)
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
