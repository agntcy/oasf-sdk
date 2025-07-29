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

	translationv1grpc "buf.build/gen/go/agntcy/oasf-sdk/grpc/go/translation/v1/translationv1grpc"
	"github.com/agntcy/oasf-sdk/translation/config"
	controllerv1 "github.com/agntcy/oasf-sdk/translation/controller/v1"
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

	if err := server.start(); err != nil {
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

	translationv1grpc.RegisterTranslationServiceServer(server.grpcServer, controllerv1.NewRoutingController())

	reflection.Register(server.grpcServer)

	return server, nil
}

func (s Server) close() {
	s.grpcServer.GracefulStop()
}

func (s Server) start() error {
	listen, err := net.Listen("tcp", s.cfg.ListenAddress)
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
