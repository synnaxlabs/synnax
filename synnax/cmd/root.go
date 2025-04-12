// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var rootCmd = &cobra.Command{
	Use:   "synnax",
	Short: "The telemetry engine for operating large scale hardware systems with ease.",
	Long: `Synnax is a distributed telemetry engine designed to acquire and store data
from, issue commands to, and process telemetry generated by hardware systems. It scales
horizontally, and can be deployed on edge devices (data acquisition) in highly dynamic
environments with intermittent network connectivity, or in cloud environments (data
processing) for high performance analysis.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If the user doesn't specify a subcommand, print the help message.
		if viper.GetBool(versionFlag) {
			versionCmd.Run(cmd, args)
			return nil
		}
		return cmd.Help()
	},
}

// Execute is the entrypoint for the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	configureRootFlags()
	bindFlags(rootCmd)
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	cfgFile := viper.GetString(configFlag)
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".synnax")
	}
	viper.SetEnvPrefix("synnax")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	if err := viper.ReadInConfig(); err != nil {
		zap.S().Error("failed to read config", zap.Error(err))
	}
}
