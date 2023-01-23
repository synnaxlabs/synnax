// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"go.uber.org/zap"
)

var certCmd = &cobra.Command{
	Use:   "cert",
	Short: "Generate self-signed certificates for securing a Synnax cluster.",
	Args:  cobra.NoArgs,
}

var certCA = &cobra.Command{
	Use:   "ca",
	Short: "Generate a self-signed CA certificate.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		l, err := configureLogging(viper.GetBool("verbose"))
		if err != nil {
			return err
		}
		factory, err := cert.NewFactory(buildCertFactoryConfig(l))
		if err != nil {
			return err
		}
		return factory.CreateCAPair()
	},
}

var certNode = &cobra.Command{
	Use:   "node",
	Short: "Generate a self-signed node certificate.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, hosts []string) error {
		l, err := configureLogging(viper.GetBool("verbose"))
		if err != nil {
			return err
		}
		// convert hosts to addresses
		addresses := make([]address.Address, len(hosts))
		for i, host := range hosts {
			addresses[i] = address.Address(host)
		}
		cfg := buildCertFactoryConfig(l)
		cfg.Hosts = addresses
		factory, err := cert.NewFactory(cfg)
		if err != nil {
			return err
		}
		return factory.CreateNodePair()
	},
}

func init() {
	rootCmd.AddCommand(certCmd)

	certCmd.AddCommand(certCA)
	certCmd.AddCommand(certNode)
}

func buildCertLoaderConfig(logger *zap.Logger) cert.LoaderConfig {
	return cert.LoaderConfig{
		CertsDir:     viper.GetString("certs-dir"),
		CAKeyPath:    viper.GetString("ca-key"),
		CACertPath:   viper.GetString("ca-cert"),
		NodeKeyPath:  viper.GetString("node-key"),
		NodeCertPath: viper.GetString("node-cert"),
		Logger:       logger.Sugar(),
	}
}

func buildCertFactoryConfig(logger *zap.Logger) cert.FactoryConfig {
	return cert.FactoryConfig{
		LoaderConfig:  buildCertLoaderConfig(logger),
		AllowKeyReuse: config.BoolPointer(viper.GetBool("allow-key-reuse")),
		KeySize:       viper.GetInt("key-size"),
	}
}
