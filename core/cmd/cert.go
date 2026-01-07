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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/flags"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
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
		ins := configureInstrumentation()

		factory, err := cert.NewFactory(buildCertFactoryConfig(ins))
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
		ins := configureInstrumentation()
		addresses := make([]address.Address, len(hosts))
		for i, host := range hosts {
			addresses[i] = address.Address(host)
		}
		cfg := buildCertFactoryConfig(ins)
		cfg.Hosts = addresses
		factory, err := cert.NewFactory(cfg)
		if err != nil {
			return err
		}
		return factory.CreateNodePair()
	},
}

func init() {
	root.AddCommand(certCmd)

	certCmd.AddCommand(certCA)
	certCmd.AddCommand(certNode)
}

func buildCertLoaderConfig(ins alamos.Instrumentation) cert.LoaderConfig {
	return cert.LoaderConfig{
		Instrumentation: ins,
		CertsDir:        viper.GetString(certsDirFlag),
		CAKeyPath:       viper.GetString(caKeyFlag),
		CACertPath:      viper.GetString(caCertFlag),
		NodeKeyPath:     viper.GetString(nodeKeyFlag),
		NodeCertPath:    viper.GetString(nodeCertFlag),
	}
}

func buildCertFactoryConfig(ins alamos.Instrumentation) cert.FactoryConfig {
	return cert.FactoryConfig{
		LoaderConfig:  buildCertLoaderConfig(ins),
		AllowKeyReuse: config.Bool(viper.GetBool(allowKeyReuseFlag)),
		KeySize:       viper.GetInt(keySizeFlag),
	}
}

func generateAutoCerts(ins alamos.Instrumentation) error {
	cfg := buildCertFactoryConfig(ins)
	cfg.Hosts = []address.Address{address.Address(viper.GetString(flags.Listen))}
	factory, err := cert.NewFactory(cfg)
	if err != nil {
		return err
	}
	err = factory.CreateCAPairIfMissing()
	if err != nil {
		return err
	}
	return factory.CreateNodePairIfMissing()
}
