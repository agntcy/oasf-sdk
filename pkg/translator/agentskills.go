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

	"google.golang.org/protobuf/types/known/structpb"
)

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

// BuildSkillMarkdown renders SKILL.md content from a manifest.
// The manifest is expected to match the agentskills_manifest schema.
func BuildSkillMarkdown(manifest *structpb.Struct, opts ...MarkdownOption) (string, error) {
	if manifest == nil {
		return "", errors.New("manifest is nil")
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
