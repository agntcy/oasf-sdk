// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/structpb"
)

const bundleSkillMarkdown = `---
name: bundled-skill
description: Skill with supporting files.
metadata:
  author: example-org
  version: "1.0"
---

# Bundled Skill
`

func getSkillArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o600,
			Size: int64(len(content)),
		}

		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header for %q: %v", name, err)
		}

		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("write tar payload for %q: %v", name, err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	return buf.Bytes()
}

func archiveToStruct(t *testing.T, archive []byte) *structpb.Struct {
	t.Helper()

	input, err := structpb.NewStruct(map[string]any{
		"skillMarkdown": bundleSkillMarkdown,
		"skillArchive":  base64.StdEncoding.EncodeToString(archive),
	})
	if err != nil {
		t.Fatalf("Failed to build input struct: %v", err)
	}

	return input
}

func TestSkillBundleToRecord(t *testing.T) {
	archive := getSkillArchive(t, map[string]string{
		"SKILL.md":            bundleSkillMarkdown,
		"references/extra.md": "# Extra reference\n",
		"scripts/run.sh":      "#!/bin/sh\n",
	})

	input := archiveToStruct(t, archive)

	record, err := SkillBundleToRecord(input)
	if err != nil {
		t.Fatalf("SkillBundleToRecord() error: %v", err)
	}

	fields := record.GetFields()
	if fields["name"].GetStringValue() != "bundled-skill" {
		t.Errorf("Expected record name bundled-skill, got %q", fields["name"].GetStringValue())
	}

	found, module := findAgentSkillsModule(record)
	if !found || module == nil {
		t.Fatalf("Expected agentskills module in record")
	}

	moduleData := module.GetFields()["data"].GetStructValue()
	artifact := module.GetFields()["artifact"].GetStructValue()

	if artifact.GetFields()["media_type"].GetStringValue() != agentSkillsBundleMediaType {
		t.Errorf("Expected bundle media type %q, got %q", agentSkillsBundleMediaType, artifact.GetFields()["media_type"].GetStringValue())
	}

	raw := artifactDataBytes(module)
	if !bytes.Equal(raw, archive) {
		t.Fatalf("Expected artifact payload to match input archive")
	}

	artifacts := moduleData.GetFields()["artifacts"].GetListValue().GetValues()
	if len(artifacts) != 2 {
		t.Fatalf("Expected 2 indexed artifacts, got %d", len(artifacts))
	}

	assertArtifactEntry(t, artifacts[0], "references/extra.md", "reference")
	assertArtifactEntry(t, artifacts[1], "scripts/run.sh", "script")
}

func TestSkillBundleToRecordMissingArchive(t *testing.T) {
	input, err := structpb.NewStruct(map[string]any{"skillMarkdown": bundleSkillMarkdown})
	if err != nil {
		t.Fatalf("Failed to build input struct: %v", err)
	}

	_, err = SkillBundleToRecord(input)
	if err == nil || !strings.Contains(err.Error(), "skillArchive") {
		t.Fatalf("Expected missing skillArchive error, got %v", err)
	}
}

func TestSkillBundleToRecordMissingSkillFile(t *testing.T) {
	archive := getSkillArchive(t, map[string]string{
		"references/extra.md": "# Extra reference\n",
	})

	input := archiveToStruct(t, archive)

	_, err := SkillBundleToRecord(input)
	if err == nil || !strings.Contains(err.Error(), "SKILL.md") {
		t.Fatalf("Expected missing SKILL.md error, got %v", err)
	}
}

func TestSkillBundleToRecordRejectsPathTraversal(t *testing.T) {
	var buf bytes.Buffer

	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for _, name := range []string{"../SKILL.md", bundleSkillMarkdown} {
		hdr := &tar.Header{Name: name, Mode: 0o600, Size: int64(len(bundleSkillMarkdown))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("write tar header: %v", err)
		}

		if _, err := tw.Write([]byte(bundleSkillMarkdown)); err != nil {
			t.Fatalf("write tar payload: %v", err)
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}

	if err := gzw.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}

	input := archiveToStruct(t, buf.Bytes())

	_, err := SkillBundleToRecord(input)
	if err == nil || !strings.Contains(err.Error(), "path traversal") {
		t.Fatalf("Expected path traversal error, got %v", err)
	}
}

func TestRecordToSkillMarkdownRejectsBundleArtifact(t *testing.T) {
	archive := getSkillArchive(t, map[string]string{"SKILL.md": bundleSkillMarkdown})

	input := archiveToStruct(t, archive)

	record, err := SkillBundleToRecord(input)
	if err != nil {
		t.Fatalf("SkillBundleToRecord() error: %v", err)
	}

	_, err = RecordToSkillMarkdown(record)
	if err == nil || !strings.Contains(err.Error(), "bundle archive") {
		t.Fatalf("Expected bundle extraction error, got %v", err)
	}
}

func TestRecordToSkillBundle(t *testing.T) {
	archive := getSkillArchive(t, map[string]string{
		"SKILL.md":            bundleSkillMarkdown,
		"references/extra.md": "# Extra reference\n",
	})

	input := archiveToStruct(t, archive)

	record, err := SkillBundleToRecord(input)
	if err != nil {
		t.Fatalf("SkillBundleToRecord() error: %v", err)
	}

	got, err := RecordToSkillBundle(record)
	if err != nil {
		t.Fatalf("RecordToSkillBundle() error: %v", err)
	}

	if !bytes.Equal(got, archive) {
		t.Fatalf("Expected bundle bytes to match original archive")
	}
}

func TestSkillBundleToRecordArtifactDigestMatchesPayload(t *testing.T) {
	archive := getSkillArchive(t, map[string]string{"SKILL.md": bundleSkillMarkdown})

	input := archiveToStruct(t, archive)

	record, err := SkillBundleToRecord(input)
	if err != nil {
		t.Fatalf("SkillBundleToRecord() error: %v", err)
	}

	_, module := findAgentSkillsModule(record)
	artifact := module.GetFields()["artifact"].GetStructValue()

	encoded := artifact.GetFields()["data"].GetStringValue()

	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode artifact data: %v", err)
	}

	if !bytes.Equal(decoded, archive) {
		t.Fatalf("Expected decoded artifact to match archive bytes")
	}
}

func TestCleanPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		wantError string
	}{
		{name: "entrypoint", input: "SKILL.md", want: "SKILL.md"},
		{name: "normalized entrypoint", input: "./SKILL.md", want: "SKILL.md"},
		{name: "nested file", input: "references/extra.md", want: "references/extra.md"},
		{name: "windows separators", input: `scripts\run.sh`, want: "scripts/run.sh"},
		{name: "reject escaped traversal", input: "references/../../etc/passwd", wantError: "path traversal"},
		{name: "reject parent prefix", input: "../SKILL.md", wantError: "path traversal"},
		{name: "reject parent only", input: "..", wantError: "path traversal"},
		{name: "reject current directory", input: ".", wantError: "invalid archive path"},
		{name: "reject empty string", input: "", wantError: "invalid archive path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := cleanPath(tt.input)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("sanitizeArchivePath(%q) expected error containing %q, got nil", tt.input, tt.wantError)
				}

				if !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("sanitizeArchivePath(%q) error = %q, want substring %q", tt.input, err.Error(), tt.wantError)
				}

				return
			}

			if err != nil {
				t.Fatalf("sanitizeArchivePath(%q) unexpected error: %v", tt.input, err)
			}

			if got != tt.want {
				t.Errorf("sanitizeArchivePath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func assertArtifactEntry(t *testing.T, value *structpb.Value, wantPath, wantType string) {
	t.Helper()

	entry := value.GetStructValue()
	if entry == nil {
		t.Fatalf("Expected artifact entry struct")
	}

	fields := entry.GetFields()
	if fields["path"].GetStringValue() != wantPath {
		t.Errorf("Expected artifact path %q, got %q", wantPath, fields["path"].GetStringValue())
	}

	if fields["type"].GetStringValue() != wantType {
		t.Errorf("Expected artifact type %q, got %q", wantType, fields["type"].GetStringValue())
	}

	if !strings.HasPrefix(fields["artifact_hash"].GetStringValue(), "sha256:") {
		t.Errorf("Expected sha256 artifact_hash, got %q", fields["artifact_hash"].GetStringValue())
	}
}
