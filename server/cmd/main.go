// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"

	"github.com/agntcy/oasf-sdk/server"
	"github.com/agntcy/oasf-sdk/server/config"
	"github.com/spf13/cobra"
)

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

func main() {
	cobra.CheckErr(rootCmd.Execute())
}
