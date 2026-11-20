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
	"io/fs"
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
)

// NewCmd returns the root synnax command with all of its subcommands. It binds the
// root flags to viper.
func NewCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "synnax",
		Short: "The telemetry engine for operating large scale hardware systems with ease",
		Long:  "Synnax is a telemetry engine that acquires and stores data from hardware systems, sends commands to them, and processes their telemetry. It runs on edge devices for data acquisition, including on networks with intermittent connectivity, or in the cloud for high-performance analysis.",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return initConfig(cmd.Flags().Changed(flagConfig))
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			if viper.GetBool(flagVersion) {
				return version.FPrint(cmd.OutOrStdout())
			}
			return cmd.Help()
		},
	}
	addFlags(root)
	root.AddCommand(
		version.NewCmd(),
		cert.NewCmd(),
		start.NewCmd(),
		update.NewCmd(release.DefaultConfig),
	)
	root.AddCommand(serviceCmds()...)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return errors.Mark(err, validate.ErrValidation)
	})
	lo.Must0(viper.BindPFlags(root.PersistentFlags()))
	lo.Must0(viper.BindPFlags(root.Flags()))
	return root
}

// initConfig reads the config file into viper. A missing file is an error only when
// the operator passed its path, since the default path rarely exists.
func initConfig(pathPassed bool) error {
	cfgFile := viper.GetString(flagConfig)
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".synnax")
	}
	viper.SetEnvPrefix("synnax")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	var notFound viper.ConfigFileNotFoundError
	err := viper.ReadInConfig()
	if !pathPassed && (errors.Is(err, fs.ErrNotExist) || errors.As(err, &notFound)) {
		return nil
	}
	if err != nil {
		return errors.Mark(
			errors.Wrapf(err, "failed to read config file %s", viper.ConfigFileUsed()),
			validate.ErrValidation,
		)
	}
	return nil
}
