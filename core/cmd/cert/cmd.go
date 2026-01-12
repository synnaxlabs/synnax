// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cert

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Generate self-signed certificates for a Synnax Core",
	Long: `Generate self-signed certificates for a Synnax Core.
See each sub-command's help for details on how to use them.`,
	Args: cobra.NoArgs,
}

var caCmd = &cobra.Command{
	Use:   "ca",
	Short: "Generate a self-signed CA certificate",
	Long:  "Generate a self-signed CA certificate.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cmd.SilenceUsage = true
		ins := instrumentation.Configure()
		defer instrumentation.Cleanup(cmd.Context(), ins)
		factory, err := cert.NewFactory(BuildCertFactoryConfig(ins))
		if err != nil {
			return err
		}
		return factory.CreateCAPair()
	},
}

var nodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Generate a self-signed node certificate",
	Long:  "Generate a self-signed node certificate.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, hosts []string) error {
		cmd.SilenceUsage = true
		ins := instrumentation.Configure()
		defer instrumentation.Cleanup(cmd.Context(), ins)
		addresses := make([]address.Address, len(hosts))
		for i, host := range hosts {
			addresses[i] = address.Address(host)
		}
		cfg := BuildCertFactoryConfig(ins)
		cfg.Hosts = addresses
		factory, err := cert.NewFactory(cfg)
		if err != nil {
			return err
		}
		return factory.CreateNodePair()
	},
}

// AddCommand adds the cert subcommand to the given parent command.
func AddCommand(cmd *cobra.Command) error {
	cmd.AddCommand(certCmd)
	BindFlags(caCmd)
	if err := viper.BindPFlags(caCmd.Flags()); err != nil {
		return err
	}
	if err := viper.BindPFlags(caCmd.PersistentFlags()); err != nil {
		return err
	}
	certCmd.AddCommand(caCmd)
	BindFlags(nodeCmd)
	certCmd.AddCommand(nodeCmd)
	return nil
}
