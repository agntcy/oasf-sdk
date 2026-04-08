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

// MarkdownOption configures the markdown output.
type MarkdownOption func(*markdownOptions)

type markdownOptions struct {
	body string
}

// WithBody sets the markdown body appended after frontmatter.
func WithBody(body string) MarkdownOption {
	return func(opts *markdownOptions) {
		opts.body = body
	}
}

// RecordToSkillMarkdown renders SKILL.md content from a record containing the agentskills module.
// The record must have an agentskills module with a skill_manifest field.
func RecordToSkillMarkdown(record *structpb.Struct, opts ...MarkdownOption) (string, error) {
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
		return "", errors.New("skill_manifest is missing")
	}

	options := &markdownOptions{}
	for _, opt := range opts {
		opt(options)
	}

	manifestMap := manifest.AsMap()
	name := getString(manifestMap, "name")
	description := getString(manifestMap, "description")

	if name == "" || description == "" {
		return "", errors.New("manifest must include name and description")
	}

	license := getString(manifestMap, "license")
	compatibility := getString(manifestMap, "compatibility")
	version := getString(manifestMap, "version")
	allowedTools := getStringSlice(manifestMap, "allowed_tools")
	frontmatterMetadata := getStringMap(manifestMap, "frontmatter_metadata")

	if version != "" {
		if _, ok := frontmatterMetadata["version"]; !ok {
			frontmatterMetadata["version"] = version
		}
	}

	lines := []string{"---"}
	lines = append(lines, "name: "+yamlScalar(name))
	lines = append(lines, "description: "+yamlScalar(description))

	if license != "" {
		lines = append(lines, "license: "+yamlScalar(license))
	}

	if compatibility != "" {
		lines = append(lines, "compatibility: "+yamlScalar(compatibility))
	}

	if version != "" {
		lines = append(lines, "version: "+yamlScalar(version))
	}

	if len(allowedTools) > 0 {
		lines = append(lines, "allowed-tools: "+strings.Join(allowedTools, " "))
	}

	if len(frontmatterMetadata) > 0 {
		lines = append(lines, "metadata:")

		keys := make([]string, 0, len(frontmatterMetadata))
		for key := range frontmatterMetadata {
			keys = append(keys, key)
		}

		sort.Strings(keys)

		for _, key := range keys {
			value := frontmatterMetadata[key]
			lines = append(lines, "  "+key+": "+yamlScalar(value))
		}
	}

	lines = append(lines, "---")

	body := strings.TrimSpace(options.body)
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
func SkillMarkdownToRecord(skillData *structpb.Struct, opts ...TranslatorOption) (*structpb.Struct, error) {
	mdVal, ok := skillData.GetFields()["skillMarkdown"]
	if !ok {
		return nil, errors.New("missing 'skillMarkdown' in input data")
	}

	content := mdVal.GetStringValue()
	if content == "" {
		return nil, errors.New("'skillMarkdown' is empty")
	}

	name, description, version, license, compatibility, allowedTools, metadata, body, err := parseSkillMarkdownContent(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md content: %w", err)
	}

	recordName := name
	if recordName == "" {
		recordName = "generated-skill-agent"
	}

	recordDescription := description
	if recordDescription == "" {
		recordDescription = "Agent generated from SKILL.md"
	}

	recordVersion := version
	if recordVersion == "" {
		recordVersion = defaultVersion
	}

	createdAt := time.Now().UTC().Format(time.RFC3339)

	// Build skill_manifest struct fields.
	manifestFields := map[string]*structpb.Value{
		"name": {Kind: &structpb.Value_StringValue{StringValue: name}},
		"description": {Kind: &structpb.Value_StringValue{StringValue: description}},
	}

	if license != "" {
		manifestFields["license"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: license}}
	}

	if compatibility != "" {
		manifestFields["compatibility"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: compatibility}}
	}

	if version != "" {
		manifestFields["version"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: version}}
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

	moduleData := &structpb.Struct{Fields: moduleDataFields}

	agentSkillsModule := &structpb.Struct{
		Fields: map[string]*structpb.Value{
			"name": {Kind: &structpb.Value_StringValue{StringValue: AgentSkillsModuleName}},
			"data": {Kind: &structpb.Value_StructValue{StructValue: moduleData}},
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
			"name":           {Kind: &structpb.Value_StringValue{StringValue: recordName}},
			"schema_version": {Kind: &structpb.Value_StringValue{StringValue: targetVersion}},
			"version":        {Kind: &structpb.Value_StringValue{StringValue: recordVersion}},
			"description":    {Kind: &structpb.Value_StringValue{StringValue: recordDescription}},
			"authors": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{Values: []*structpb.Value{}},
				},
			},
			"created_at": {Kind: &structpb.Value_StringValue{StringValue: createdAt}},
			"skills": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{Values: []*structpb.Value{}},
				},
			},
			"domains": {
				Kind: &structpb.Value_ListValue{
					ListValue: &structpb.ListValue{Values: []*structpb.Value{}},
				},
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

// parseSkillMarkdownContent parses a SKILL.md file and returns its components.
func parseSkillMarkdownContent(content string) (name, description, version, license, compatibility string, allowedTools []string, metadata map[string]string, body string, err error) {
	sections := strings.SplitN(content, "---", 3)
	if len(sections) < 3 {
		return "", "", "", "", "", nil, nil, "", errors.New("invalid SKILL.md: missing frontmatter delimiters")
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
		case "version":
			version = v
		case "allowed-tools":
			allowedTools = strings.Fields(v)
		}
	}

	if name == "" || description == "" {
		return "", "", "", "", "", nil, nil, "", errors.New("SKILL.md must include name and description in frontmatter")
	}

	return name, description, version, license, compatibility, allowedTools, metadata, body, nil
}

// splitYAMLKeyValue splits a YAML "key: value" line.
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
			str, ok := item.(string)
			if ok {
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
