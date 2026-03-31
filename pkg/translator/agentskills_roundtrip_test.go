// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"errors"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	recordutil "github.com/agntcy/oasf-sdk/pkg/record"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSkillMarkdownRoundTrip(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "agentskills", "SKILL.md"))
	if err != nil {
		t.Fatalf("Failed to read SKILL.md: %v", err)
	}

	originalManifest, originalBody, err := parseSkillMarkdown(string(content))
	if err != nil {
		t.Fatalf("Failed to parse SKILL.md: %v", err)
	}

	recordStruct, err := structpb.NewStruct(map[string]any{
		"schema_version": "1.0.0",
		"modules": []any{
			map[string]any{
				"name": "agentskills",
				"data": map[string]any{
					"skill_file":     "SKILL.md",
					"skill_manifest": manifestToStruct(originalManifest),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to build record struct: %v", err)
	}

	found, moduleData := recordutil.GetModuleData(recordStruct, "agentskills")
	if !found || moduleData == nil {
		t.Fatalf("Expected agentskills module data in record")
	}

	manifest := moduleData.GetFields()["skill_manifest"].GetStructValue()
	if manifest == nil {
		t.Fatalf("Expected skill_manifest in module data")
	}

	rebuiltMarkdown, err := BuildSkillMarkdown(manifest, WithBody(originalBody))
	if err != nil {
		t.Fatalf("BuildSkillMarkdown() error: %v", err)
	}

	rebuiltManifest, rebuiltBody, err := parseSkillMarkdown(rebuiltMarkdown)
	if err != nil {
		t.Fatalf("Failed to parse rebuilt SKILL.md: %v", err)
	}

	if !reflect.DeepEqual(normalizeManifest(originalManifest), normalizeManifest(rebuiltManifest)) {
		t.Fatalf("Manifest mismatch after roundtrip")
	}

	if strings.TrimSpace(originalBody) != strings.TrimSpace(rebuiltBody) {
		t.Fatalf("Body mismatch after roundtrip")
	}
}

type skillManifest struct {
	Name          string
	Description   string
	License       string
	Compatibility string
	Version       string
	AllowedTools  []string
	Metadata      map[string]string
}

func parseSkillMarkdown(content string) (skillManifest, string, error) {
	sections := strings.Split(content, "---")
	if len(sections) < 3 {
		return skillManifest{}, "", ErrInvalidFrontmatter
	}

	frontmatter := strings.TrimSpace(sections[1])
	body := strings.TrimSpace(strings.Join(sections[2:], "---"))

	manifest := skillManifest{Metadata: map[string]string{}}

	lines := strings.Split(frontmatter, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			if line == "metadata:" {
				for i+1 < len(lines) {
					next := lines[i+1]
					if !strings.HasPrefix(next, "  ") {
						break
					}

					i++
					key, value := splitKeyValue(strings.TrimSpace(next))
					manifest.Metadata[key] = value
				}
			} else {
				key, value := splitKeyValue(line)
				switch key {
				case "name":
					manifest.Name = value
				case "description":
					manifest.Description = value
				case "license":
					manifest.License = value
				case "compatibility":
					manifest.Compatibility = value
				case "version":
					manifest.Version = value
				case "allowed-tools":
					manifest.AllowedTools = strings.Fields(value)
				}
			}
		}
	}

	return manifest, body, nil
}

var ErrInvalidFrontmatter = errors.New("invalid frontmatter")

func splitKeyValue(line string) (string, string) {
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

func manifestToStruct(manifest skillManifest) map[string]any {
	data := map[string]any{
		"name":        manifest.Name,
		"description": manifest.Description,
	}

	if manifest.License != "" {
		data["license"] = manifest.License
	}

	if manifest.Compatibility != "" {
		data["compatibility"] = manifest.Compatibility
	}

	if manifest.Version != "" {
		data["version"] = manifest.Version
	}

	if len(manifest.AllowedTools) > 0 {
		allowed := make([]any, 0, len(manifest.AllowedTools))
		for _, tool := range manifest.AllowedTools {
			allowed = append(allowed, tool)
		}

		data["allowed_tools"] = allowed
	}

	if len(manifest.Metadata) > 0 {
		metadata := map[string]any{}
		for key, value := range manifest.Metadata {
			metadata[key] = value
		}

		data["frontmatter_metadata"] = metadata
	}

	return data
}

func normalizeManifest(manifest skillManifest) skillManifest {
	copyManifest := manifest
	copyManifest.Metadata = map[string]string{}
	maps.Copy(copyManifest.Metadata, manifest.Metadata)

	sort.Strings(copyManifest.AllowedTools)

	return copyManifest
}
