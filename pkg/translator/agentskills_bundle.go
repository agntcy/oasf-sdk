// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package translator

import (
	"archive/tar"
	"bytes"
	"cmp"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	recordutil "github.com/agntcy/oasf-sdk/pkg/record"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	skillFile                   = "SKILL.md"
	agentSkillsBundleMediaType  = "application/agent-skills+gzip"
	maxSkillArchiveFiles        = 1000
	maxSkillArchiveUncompressed = 50 * 1024 * 1024 // 50 MiB
)

type archiveEntry struct {
	path string
	typ  string
	hash string
}

// SkillBundleToRecord converts SKILL.md content and a skill archive into an OASF record.
// The input must include:
//   - skillMarkdown: SKILL.md content (required)
//   - skillArchive: base64-encoded .gzip bytes (required)
//
// The archive is stored in module.artifact with media type application/agent-skills+gzip.
// The frontmatter is stored in module.data.skill_manifest.
// Non-entrypoint files are indexed in module.data.artifacts when present in the archive.
func SkillBundleToRecord(skillData *structpb.Struct, opts ...TranslatorOption) (*structpb.Struct, error) {
	content, err := extractSkillMarkdown(skillData)
	if err != nil {
		return nil, err
	}

	archive, err := extractSkillArchive(skillData)
	if err != nil {
		return nil, err
	}

	entries, err := getArchiveEntries(archive)
	if err != nil {
		return nil, fmt.Errorf("invalid skill archive: %w", err)
	}

	return skillToRecord(content, archive, agentSkillsBundleMediaType, entries, opts...)
}

// RecordToSkillBundle returns the .gzip bytes from a application/agent-skills+gzip record.
func RecordToSkillBundle(record *structpb.Struct) ([]byte, error) {
	if record == nil {
		return nil, errors.New("record is nil")
	}

	found, moduleStruct := recordutil.GetModule(record, AgentSkillsModuleName)
	if !found || moduleStruct == nil {
		return nil, fmt.Errorf("module %q not found", AgentSkillsModuleName)
	}

	mediaType := moduleStruct.GetFields()["artifact"].GetStructValue().GetFields()["media_type"].GetStringValue()
	if mediaType != agentSkillsBundleMediaType {
		return nil, fmt.Errorf("not a skill bundle: %q", mediaType)
	}

	raw := artifactDataBytes(moduleStruct)
	if len(raw) == 0 {
		return nil, errors.New("bundle artifact data is missing or invalid")
	}

	return raw, nil
}

func extractSkillMarkdown(skillData *structpb.Struct) (string, error) {
	if skillData == nil {
		return "", errors.New("input data is nil")
	}

	mdVal, ok := skillData.GetFields()["skillMarkdown"]
	if !ok {
		return "", errors.New("missing 'skillMarkdown' in input data")
	}

	content := mdVal.GetStringValue()
	if content == "" {
		return "", errors.New("'skillMarkdown' is empty")
	}

	return content, nil
}

func extractSkillArchive(skillData *structpb.Struct) ([]byte, error) {
	if skillData == nil {
		return nil, errors.New("input data is nil")
	}

	archiveVal, ok := skillData.GetFields()["skillArchive"]
	if !ok {
		return nil, errors.New("missing 'skillArchive' in input data")
	}

	encoded := archiveVal.GetStringValue()
	if encoded == "" {
		return nil, errors.New("'skillArchive' is empty")
	}

	archive, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode 'skillArchive' base64: %w", err)
	}

	if len(archive) == 0 {
		return nil, errors.New("'skillArchive' is empty")
	}

	return archive, nil
}

func getArchiveEntries(archive []byte) ([]archiveEntry, error) {
	reader, closeReader, err := newTarGzReader(archive)
	if err != nil {
		return nil, err
	}

	defer func() { _ = closeReader() }()

	builder := &archiveIndexBuilder{}

	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}

		if err := builder.addEntry(reader, header); err != nil {
			return nil, err
		}
	}

	if !builder.foundSkillFile {
		return nil, fmt.Errorf("archive must contain %q", skillFile)
	}

	return builder.entries, nil
}

func newTarGzReader(archive []byte) (*tar.Reader, func() error, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, nil, fmt.Errorf("read gzip archive: %w", err)
	}

	return tar.NewReader(gzr), gzr.Close, nil
}

type archiveIndexBuilder struct {
	entries           []archiveEntry
	foundSkillFile    bool
	uncompressedBytes int64
	fileCount         int
}

func (b *archiveIndexBuilder) addEntry(reader *tar.Reader, header *tar.Header) error {
	if header.Typeflag != tar.TypeReg {
		return nil
	}

	entryPath, err := cleanPath(header.Name)
	if err != nil {
		return err
	}

	if entryPath == skillFile {
		b.foundSkillFile = true
	}

	if err := b.checkLimits(header.Size); err != nil {
		return fmt.Errorf("invalid size for %q: %w", entryPath, err)
	}

	if entryPath == skillFile {
		if _, err := io.CopyN(io.Discard, reader, header.Size); err != nil {
			return fmt.Errorf("read %q: %w", entryPath, err)
		}

		return nil
	}

	entry, err := readArchiveEntry(reader, header, entryPath)
	if err != nil {
		return err
	}

	b.entries = append(b.entries, entry)

	return nil
}

func (b *archiveIndexBuilder) checkLimits(size int64) error {
	if size < 0 {
		return errors.New("negative file size")
	}

	b.uncompressedBytes += size
	if b.uncompressedBytes > maxSkillArchiveUncompressed {
		return fmt.Errorf("archive exceeds %d byte uncompressed limit", maxSkillArchiveUncompressed)
	}

	b.fileCount++
	if b.fileCount > maxSkillArchiveFiles {
		return fmt.Errorf("archive exceeds %d file limit", maxSkillArchiveFiles)
	}

	return nil
}

func readArchiveEntry(reader *tar.Reader, header *tar.Header, entryPath string) (archiveEntry, error) {
	limitReader := io.LimitReader(reader, header.Size)

	payload, err := io.ReadAll(limitReader)
	if err != nil {
		return archiveEntry{}, fmt.Errorf("read %q: %w", entryPath, err)
	}

	if int64(len(payload)) != header.Size {
		return archiveEntry{}, fmt.Errorf("truncated payload for %q", entryPath)
	}

	sum := sha256.Sum256(payload)

	return archiveEntry{
		path: entryPath,
		typ:  classifySkillArtifactPath(entryPath),
		hash: fmt.Sprintf("sha256:%x", sum),
	}, nil
}

func cleanPath(name string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	clean = strings.TrimPrefix(clean, "./")

	if clean == "" || clean == "." {
		return "", fmt.Errorf("invalid archive path %q", name)
	}

	if strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("archive path traversal is not allowed: %q", name)
	}

	return clean, nil
}

func classifySkillArtifactPath(entryPath string) string {
	switch {
	case strings.HasPrefix(entryPath, "scripts/"):
		return "script"
	case strings.HasPrefix(entryPath, "references/"):
		return "reference"
	case strings.HasPrefix(entryPath, "assets/"):
		return "asset"
	case strings.HasPrefix(entryPath, "templates/"), strings.HasPrefix(entryPath, "template/"):
		return "template"
	case strings.HasPrefix(entryPath, "workflows/"):
		return "workflow"
	default:
		return "other"
	}
}

func buildArtifactsListValue(entries []archiveEntry) *structpb.Value {
	sorted := slices.Clone(entries)
	slices.SortFunc(sorted, func(a, b archiveEntry) int {
		return cmp.Compare(a.path, b.path)
	})

	values := make([]*structpb.Value, 0, len(sorted))

	for _, entry := range sorted {
		fields := map[string]*structpb.Value{
			"path": {Kind: &structpb.Value_StringValue{StringValue: entry.path}},
			"type": {Kind: &structpb.Value_StringValue{StringValue: entry.typ}},
		}

		if entry.hash != "" {
			fields["artifact_hash"] = &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: entry.hash}}
		}

		values = append(values, &structpb.Value{
			Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{Fields: fields}},
		})
	}

	return &structpb.Value{
		Kind: &structpb.Value_ListValue{ListValue: &structpb.ListValue{Values: values}},
	}
}
