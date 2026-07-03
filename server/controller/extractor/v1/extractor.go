// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"context"
	"fmt"
	"log/slog"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/extractor/v1/extractorv1grpc"
	extractorv1 "buf.build/gen/go/agntcy/oasf-sdk/protocolbuffers/go/agntcy/oasfsdk/extractor/v1"
	"github.com/agntcy/oasf-sdk/pkg/extractor"
)

// extractorEngine is the subset of *extractor.Extractor the controller depends
// on. Declaring it as an interface lets the Extract handler be tested with a
// fake, without provisioning a real model.
type extractorEngine interface {
	Extract(ctx context.Context, text string, opts ...extractor.QueryOption) (extractor.Result, error)
}

// extractorCtrl serves the ExtractorService. Unlike the stateless controllers,
// it holds a warm extractor engine (model + provisioned index) built once at
// startup and reused for every request.
type extractorCtrl struct {
	engine extractorEngine
}

// New builds the extractor controller from the given options (which must include
// WithOASFURL). It provisions the assets (downloading the model and embedding the
// taxonomy if they are missing or stale — idempotent and a fast no-op when
// already current) and then loads the warm engine, so the server is
// self-sufficient at startup.
func New(ctx context.Context, opts ...extractor.Option) (extractorv1grpc.ExtractorServiceServer, error) {
	if err := extractor.Provision(ctx, opts...); err != nil {
		return nil, fmt.Errorf("failed to provision extractor assets: %w", err)
	}

	engine, err := extractor.New(opts...) //nolint:contextcheck // extractor.New is a synchronous constructor with no ctx parameter
	if err != nil {
		return nil, fmt.Errorf("failed to build extractor: %w", err)
	}

	return &extractorCtrl{engine: engine}, nil
}

func (c *extractorCtrl) Extract(ctx context.Context, req *extractorv1.ExtractRequest) (*extractorv1.ExtractResponse, error) {
	slog.Debug("Received Extract request", "text_len", len(req.GetText()), "scope", req.GetScope())

	res, err := c.engine.Extract(ctx, req.GetText(), queryOptions(req)...)
	if err != nil {
		return nil, fmt.Errorf("failed to extract: %w", err)
	}

	return &extractorv1.ExtractResponse{
		Skills:   toScoredClasses(res.Skills),
		Domains:  toScoredClasses(res.Domains),
		Modules:  toScoredClasses(res.Modules),
		Keywords: toKeywords(res.Keywords),
	}, nil
}

// queryOptions maps the request's filtering fields onto extractor query options.
// An explicit version list overrides the scope; zero-valued numeric fields fall
// back to the extractor defaults.
func queryOptions(req *extractorv1.ExtractRequest) []extractor.QueryOption {
	var opts []extractor.QueryOption

	switch versions := req.GetVersions(); {
	case len(versions) > 0:
		opts = append(opts, extractor.Versions(versions...))
	case req.GetScope() == extractorv1.VersionScope_VERSION_SCOPE_LATEST:
		opts = append(opts, extractor.Latest())
	default:
		opts = append(opts, extractor.All())
	}

	if t := req.GetTiers(); t > 0 {
		opts = append(opts, extractor.Tiers(int(t)))
	}

	if s := req.GetMinScore(); s > 0 {
		opts = append(opts, extractor.MinScore(s))
	}

	if n := req.GetMinResults(); n > 0 {
		opts = append(opts, extractor.MinResults(int(n)))
	}

	return opts
}

func toScoredClasses(in []extractor.ScoredClass) []*extractorv1.ScoredClass {
	out := make([]*extractorv1.ScoredClass, len(in))
	for i := range in {
		sc := in[i]
		out[i] = &extractorv1.ScoredClass{
			Id:          sc.ID,
			Name:        sc.Name,
			Caption:     sc.Caption,
			Description: sc.Description,
			Kind:        kindToClassType(sc.Kind),
			Versions:    sc.Versions,
			Score:       sc.Score,
			Semantic:    sc.Semantic,
			Lexical:     sc.Lexical,
			//nolint:gosec // Tier is a small positive score-group index.
			Tier: uint32(sc.Tier),
		}
	}

	return out
}

func toKeywords(in []extractor.Keyword) []*extractorv1.Keyword {
	out := make([]*extractorv1.Keyword, len(in))
	for i := range in {
		out[i] = &extractorv1.Keyword{Text: in[i].Text, Score: in[i].Score}
	}

	return out
}

func kindToClassType(k extractor.Kind) extractorv1.ClassType {
	switch k {
	case extractor.KindSkill:
		return extractorv1.ClassType_CLASS_TYPE_SKILL
	case extractor.KindDomain:
		return extractorv1.ClassType_CLASS_TYPE_DOMAIN
	case extractor.KindModule:
		return extractorv1.ClassType_CLASS_TYPE_MODULE
	default:
		return extractorv1.ClassType_CLASS_TYPE_UNSPECIFIED
	}
}
