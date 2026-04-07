// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	schemav1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/schema/v1/schemav1grpc"
	schemav1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/schema/v1"
	"github.com/agntcy/oasf-sdk/pkg/decoder"
	"github.com/agntcy/oasf-sdk/pkg/schema"
)

// schemaURLPathParts is the number of slash-separated parts after trimming the leading slash:
// ["schema", "<version>", "<type>", "<name>"].
const schemaURLPathParts = 4

type schemaCtrl struct {
	schemav1grpc.UnimplementedSchemaServiceServer
}

func New() schemav1grpc.SchemaServiceServer {
	return &schemaCtrl{}
}

func schemaOptionsFromVersion(version string) []schema.SchemaOption {
	if version == "" {
		return nil
	}

	return []schema.SchemaOption{schema.WithSchemaVersion(version)}
}

func newSchemaClient(schemaURL string) (*schema.Schema, error) {
	client, err := schema.New(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema client: %w", err)
	}

	return client, nil
}

// taxonomyToProto converts a schema.Taxonomy map to the proto map type.
func taxonomyToProto(src schema.Taxonomy) map[string]*schemav1.TaxonomyItem {
	if src == nil {
		return nil
	}

	dst := make(map[string]*schemav1.TaxonomyItem, len(src))
	for k, v := range src {
		dst[k] = taxonomyItemToProto(v)
	}

	return dst
}

func taxonomyItemToProto(src schema.TaxonomyItem) *schemav1.TaxonomyItem {
	item := &schemav1.TaxonomyItem{
		Id:          int32(src.ID), //nolint:gosec // taxonomy IDs are small positive integers defined by the OASF schema
		Name:        src.Name,
		Caption:     src.Caption,
		Description: src.Description,
		Category:    src.Category,
		Deprecated:  src.Deprecated,
	}

	if len(src.Classes) > 0 {
		item.Classes = make(map[string]*schemav1.TaxonomyItem, len(src.Classes))
		for k, v := range src.Classes {
			item.Classes[k] = taxonomyItemToProto(v)
		}
	}

	return item
}

func (s *schemaCtrl) GetDefaultSchemaVersion(ctx context.Context, req *schemav1.GetDefaultSchemaVersionRequest) (*schemav1.GetDefaultSchemaVersionResponse, error) {
	slog.Info("Received GetDefaultSchemaVersion request")

	client, err := newSchemaClient(req.GetSchemaUrl())
	if err != nil {
		return nil, err
	}

	version, err := client.GetDefaultSchemaVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default schema version: %w", err)
	}

	return &schemav1.GetDefaultSchemaVersionResponse{Version: version}, nil
}

func (s *schemaCtrl) GetAvailableSchemaVersions(ctx context.Context, req *schemav1.GetAvailableSchemaVersionsRequest) (*schemav1.GetAvailableSchemaVersionsResponse, error) {
	slog.Info("Received GetAvailableSchemaVersions request")

	client, err := newSchemaClient(req.GetSchemaUrl())
	if err != nil {
		return nil, err
	}

	versions, err := client.GetAvailableSchemaVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available schema versions: %w", err)
	}

	return &schemav1.GetAvailableSchemaVersionsResponse{Versions: versions}, nil
}

func (s *schemaCtrl) GetRecordJSONSchema(ctx context.Context, req *schemav1.GetRecordJSONSchemaRequest) (*schemav1.GetRecordJSONSchemaResponse, error) {
	slog.Info("Received GetRecordJSONSchema request")

	client, err := newSchemaClient(req.GetSchemaUrl())
	if err != nil {
		return nil, err
	}

	schemaContent, err := client.GetRecordJSONSchema(ctx, schemaOptionsFromVersion(req.GetSchemaVersion())...)
	if err != nil {
		return nil, fmt.Errorf("failed to get record JSON schema: %w", err)
	}

	schemaStruct, err := decoder.JsonToProto(schemaContent)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema JSON to proto struct: %w", err)
	}

	return &schemav1.GetRecordJSONSchemaResponse{Schema: schemaStruct}, nil
}

// parseSchemaURL decomposes a full schema URL into its base, version, type and name.
// Expected format: <schemaBase>/schema/<version>/<type>/<name>.
func parseSchemaURL(rawURL string) (string, string, schema.EntityType, string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", "", fmt.Errorf("invalid url %q: %w", rawURL, err)
	}

	parts := strings.SplitN(strings.TrimPrefix(parsed.Path, "/"), "/", schemaURLPathParts)
	if len(parts) != schemaURLPathParts || parts[0] != "schema" {
		return "", "", "", "", fmt.Errorf("url path %q must follow /schema/<version>/<type>/<name>", parsed.Path)
	}

	return parsed.Scheme + "://" + parsed.Host, parts[1], schema.EntityType(parts[2]), parts[3], nil
}

func (s *schemaCtrl) GetJSONSchema(ctx context.Context, req *schemav1.GetJSONSchemaRequest) (*schemav1.GetJSONSchemaResponse, error) {
	slog.Info("Received GetJSONSchema request")

	schemaBase, version, schemaType, name, err := parseSchemaURL(req.GetUrl())
	if err != nil {
		return nil, err
	}

	client, err := newSchemaClient(schemaBase)
	if err != nil {
		return nil, err
	}

	schemaContent, err := client.GetJSONSchema(ctx, schemaType, name, schema.WithSchemaVersion(version))
	if err != nil {
		return nil, fmt.Errorf("failed to get JSON schema: %w", err)
	}

	schemaStruct, err := decoder.JsonToProto(schemaContent)
	if err != nil {
		return nil, fmt.Errorf("failed to convert schema JSON to proto struct: %w", err)
	}

	return &schemav1.GetJSONSchemaResponse{Schema: schemaStruct}, nil
}

func (s *schemaCtrl) GetSchemaSkills(ctx context.Context, req *schemav1.GetSchemaSkillsRequest) (*schemav1.GetSchemaSkillsResponse, error) {
	slog.Info("Received GetSchemaSkills request")

	client, err := newSchemaClient(req.GetSchemaUrl())
	if err != nil {
		return nil, err
	}

	taxonomy, err := client.GetSchemaSkills(ctx, schemaOptionsFromVersion(req.GetSchemaVersion())...)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema skills: %w", err)
	}

	return &schemav1.GetSchemaSkillsResponse{Items: taxonomyToProto(taxonomy)}, nil
}

func (s *schemaCtrl) GetSchemaDomains(ctx context.Context, req *schemav1.GetSchemaDomainsRequest) (*schemav1.GetSchemaDomainsResponse, error) {
	slog.Info("Received GetSchemaDomains request")

	client, err := newSchemaClient(req.GetSchemaUrl())
	if err != nil {
		return nil, err
	}

	taxonomy, err := client.GetSchemaDomains(ctx, schemaOptionsFromVersion(req.GetSchemaVersion())...)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema domains: %w", err)
	}

	return &schemav1.GetSchemaDomainsResponse{Items: taxonomyToProto(taxonomy)}, nil
}

func (s *schemaCtrl) GetSchemaModules(ctx context.Context, req *schemav1.GetSchemaModulesRequest) (*schemav1.GetSchemaModulesResponse, error) {
	slog.Info("Received GetSchemaModules request")

	client, err := newSchemaClient(req.GetSchemaUrl())
	if err != nil {
		return nil, err
	}

	taxonomy, err := client.GetSchemaModules(ctx, schemaOptionsFromVersion(req.GetSchemaVersion())...)
	if err != nil {
		return nil, fmt.Errorf("failed to get schema modules: %w", err)
	}

	return &schemav1.GetSchemaModulesResponse{Items: taxonomyToProto(taxonomy)}, nil
}
