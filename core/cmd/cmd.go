// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/cert"
	"github.com/synnaxlabs/synnax/cmd/start"
	"github.com/synnaxlabs/synnax/cmd/update"
	"github.com/synnaxlabs/synnax/cmd/version"
	"github.com/synnaxlabs/synnax/pkg/release"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

var Cmd = &cobra.Command{
	Use:   "synnax",
	Short: "The telemetry engine for operating large scale hardware systems with ease",
	Long:  "Synnax is a telemetry engine that acquires and stores data from hardware systems, sends commands to them, and processes their telemetry. It runs on edge devices for data acquisition, including on networks with intermittent connectivity, or in the cloud for high-performance analysis.",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		if viper.GetBool(flagVersion) {
			return version.FPrint(cmd.OutOrStdout())
		}
		return cmd.Help()
	},
}

func init() {
	addFlags(Cmd)
	Cmd.AddCommand(
		version.Cmd,
		cert.Cmd,
		start.Cmd,
		update.NewCmd(release.DefaultConfig),
	)
	Cmd.AddCommand(serviceCmds()...)
	Cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return errors.Mark(err, validate.ErrValidation)
	})
	lo.Must0(viper.BindPFlags(Cmd.PersistentFlags()))
	lo.Must0(viper.BindPFlags(Cmd.Flags()))
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	cfgFile := viper.GetString(flagConfig)
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
