// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"log/slog"
	"testing"
)

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		want   slog.Level
		wantOK bool
	}{
		{"unset keeps the default", "", slog.LevelInfo, false},
		{"unparseable keeps the default", "verbose", slog.LevelInfo, false},
		{"debug", "debug", slog.LevelDebug, true},
		{"case insensitive", "DEBUG", slog.LevelDebug, true},
		{"surrounding space", "  debug  ", slog.LevelDebug, true},
		{"info", "info", slog.LevelInfo, true},
		{"warn", "warn", slog.LevelWarn, true},
		{"error", "error", slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := logLevel(tt.in)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("logLevel(%q) = (%v, %v), want (%v, %v)", tt.in, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
