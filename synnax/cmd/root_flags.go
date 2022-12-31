// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import "github.com/synnaxlabs/synnax/pkg/security/cert"

func configureRootFlags() {
	rootCmd.PersistentFlags().StringP(
		"config",
		"c",
		"",
		"config file (default is $HOME/.synnax.yaml)",
	)

	rootCmd.PersistentFlags().String(
		"certs-dir",
		cert.DefaultLoaderConfig.CertsDir,
		"The directory to store the certificates in.",
	)

	rootCmd.PersistentFlags().String(
		"ca-key",
		cert.DefaultLoaderConfig.CAKeyPath,
		"The path to the CA key.",
	)

	rootCmd.PersistentFlags().String(
		"ca-cert",
		cert.DefaultLoaderConfig.CACertPath,
		"The path to the CA certificate.",
	)

	rootCmd.PersistentFlags().String(
		"node-key",
		cert.DefaultLoaderConfig.NodeKeyPath,
		"The path to the node key.",
	)

	rootCmd.PersistentFlags().String(
		"node-cert",
		cert.DefaultLoaderConfig.NodeCertPath,
		"The path to the node certificate.",
	)

	rootCmd.PersistentFlags().Bool(
		"allow-key-reuse",
		*cert.DefaultFactoryConfig.AllowKeyReuse,
		"Whether to allow the reuse of CA keys for certificate generation.",
	)

	rootCmd.PersistentFlags().Int(
		"key-size",
		cert.DefaultFactoryConfig.KeySize,
		"The size to use for certificate key generation.",
	)

	rootCmd.PersistentFlags().BoolP(
		"verbose",
		"v",
		false,
		"Enable verbose debugging.",
	)

}
