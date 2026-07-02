// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/decoding/v1/decodingv1grpc"
	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/extractor/v1/extractorv1grpc"
	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/schema/v1/schemav1grpc"
	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/translation/v1/translationv1grpc"
	"buf.build/gen/go/agntcy/oasf-sdk/grpc/go/agntcy/oasfsdk/validation/v1/validationv1grpc"
	"github.com/agntcy/oasf-sdk/pkg/extractor"
	"github.com/agntcy/oasf-sdk/server/config"
	decodingcontrollerv1 "github.com/agntcy/oasf-sdk/server/controller/decoding/v1"
	extractorcontrollerv1 "github.com/agntcy/oasf-sdk/server/controller/extractor/v1"
	schemacontrollerv1 "github.com/agntcy/oasf-sdk/server/controller/schema/v1"
	translationcontrollerv1 "github.com/agntcy/oasf-sdk/server/controller/translation/v1"
	validationcontrollerv1 "github.com/agntcy/oasf-sdk/server/controller/validation/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	cfg        *config.Config
	grpcServer *grpc.Server
}

func Run(ctx context.Context, cfg *config.Config) error {
	server, err := NewServer(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	if err := server.start(ctx); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer server.close()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-ctx.Done():
		return fmt.Errorf("stopping server due to context cancellation: %w", ctx.Err())
	case sig := <-sigCh:
		return fmt.Errorf("stopping server due to signal: %v", sig)
	}
}

func NewServer(ctx context.Context, cfg *config.Config) (*Server, error) {
	slog.Info("Creating new server", "config", cfg)

	server := &Server{
		cfg:        cfg,
		grpcServer: grpc.NewServer(),
	}

	validationController, err := validationcontrollerv1.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create validation controller: %w", err)
	}

	decodingv1grpc.RegisterDecodingServiceServer(server.grpcServer, decodingcontrollerv1.New())
	schemav1grpc.RegisterSchemaServiceServer(server.grpcServer, schemacontrollerv1.New())
	translationv1grpc.RegisterTranslationServiceServer(server.grpcServer, translationcontrollerv1.New())
	validationv1grpc.RegisterValidationServiceServer(server.grpcServer, validationController)

	// The extractor controller is registered only when an OASF endpoint is
	// configured (it cannot run without one). On startup it provisions its assets
	// and loads the model, so the server is self-sufficient.
	if cfg.Extractor.OASFURL != "" {
		extractorController, err := extractorcontrollerv1.New(ctx, extractorOptions(cfg)...)
		if err != nil {
			return nil, fmt.Errorf("failed to create extractor controller: %w", err)
		}

		extractorv1grpc.RegisterExtractorServiceServer(server.grpcServer, extractorController)
	}

	reflection.Register(server.grpcServer)

	return server, nil
}

// extractorOptions builds the pkg/extractor options from config. Fields that are
// unset (zero-valued) fall back to the library defaults.
func extractorOptions(cfg *config.Config) []extractor.Option {
	ex := cfg.Extractor
	opts := []extractor.Option{extractor.WithOASFURL(ex.OASFURL)}

	if ex.ModelName != "" {
		opts = append(opts, extractor.WithModelName(ex.ModelName))
	}

	if ex.AssetDir != "" {
		opts = append(opts, extractor.WithAssetDir(ex.AssetDir))
	}

	if ex.SkillSemanticWeight > 0 || ex.SkillLexicalWeight > 0 {
		opts = append(opts, extractor.WithWeights(ex.SkillSemanticWeight, ex.SkillLexicalWeight))
	}

	if ex.DomainSemanticWeight > 0 || ex.DomainLexicalWeight > 0 {
		opts = append(opts, extractor.WithDomainWeights(ex.DomainSemanticWeight, ex.DomainLexicalWeight))
	}

	if ex.Tiers > 0 {
		opts = append(opts, extractor.WithDefaultTiers(ex.Tiers))
	}

	if ex.TierRatio > 0 {
		opts = append(opts, extractor.WithTierRatio(ex.TierRatio))
	}

	if ex.MinScore > 0 {
		opts = append(opts, extractor.WithDefaultMinScore(ex.MinScore))
	}

	return opts
}

func (s Server) close() {
	s.grpcServer.GracefulStop()
}

func (s Server) start(ctx context.Context) error {
	lc := &net.ListenConfig{}

	listen, err := lc.Listen(ctx, "tcp", s.cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.cfg.ListenAddress, err)
	}

	go func() {
		slog.Info("Starting server", "address", s.cfg.ListenAddress)

		if err := s.grpcServer.Serve(listen); err != nil {
			slog.Error("Server stopped unexpectedly", "error", err)
		}
	}()

	return nil
}
