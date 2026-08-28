// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/agntcy/oasf-sdk/server"
	"github.com/agntcy/oasf-sdk/server/config"
	"github.com/spf13/cobra"
)

// envLogLevel names the environment variable overriding the minimum slog level
// ("debug", "info", "warn", "error"; case-insensitive). It is read directly
// rather than through config.LoadConfig because it has to take effect before
// the config is loaded, so that config failures are logged at the chosen level
// too.
const envLogLevel = config.DefaultEnvPrefix + "_LOG_LEVEL"

var rootCmd = &cobra.Command{
	Use:   "server",
	Short: "OASF SDK Server",
	Long:  "A server for handling OASF SDK requests.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return server.Run(cmd.Context(), cfg)
	},
}

// logLevel parses s into a slog level, reporting whether it was usable. An
// empty or unrecognized value reports false so the caller leaves the default in
// place rather than failing to start over a logging setting.
//
// The four names are matched explicitly rather than deferring to
// slog.Level.UnmarshalText, which also accepts offset forms such as "INFO+1".
// Those are outside the documented contract, so they take the same fallback as
// any other unrecognized value.
func logLevel(s string) (slog.Level, bool) {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return slog.LevelDebug, true
	case "INFO":
		return slog.LevelInfo, true
	case "WARN":
		return slog.LevelWarn, true
	case "ERROR":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}

func main() {
	// slog's top-level functions log through the log package until SetDefault is
	// called, which nothing here does. SetLogLoggerLevel therefore raises the
	// minimum level for those calls — exposing slog.Debug — without changing the
	// output format.
	if l, ok := logLevel(os.Getenv(envLogLevel)); ok {
		slog.SetLogLoggerLevel(l)
	}

	cobra.CheckErr(rootCmd.Execute())
}
