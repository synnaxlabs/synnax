// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import "github.com/synnaxlabs/synnax/pkg/security/cert"

const (
	flagConfig        = "config"
	flagVersion       = "version"
	flagCertsDir      = "certs-dir"
	flagCAKey         = "ca-key"
	flagCACert        = "ca-cert"
	flagNodeKey       = "node-key"
	flagNodeCert      = "node-cert"
	flagAllowKeyReuse = "allow-key-reuse"
	flagKeySize       = "key-size"
)

func configureRootFlags() {
	root.PersistentFlags().StringP(
		flagConfig,
		"c",
		"/usr/local/synnax/config.yaml",
		"config file",
	)
	root.Flags().Bool(flagVersion, false, "Print the version of Synnax")
	root.PersistentFlags().String(
		flagCertsDir,
		cert.DefaultLoaderConfig.CertsDir,
		"The directory where certificates should be stored and/or written to.",
	)
	root.PersistentFlags().String(
		flagCAKey,
		cert.DefaultLoaderConfig.CAKeyPath,
		"The path to the CA key. This is relative to certs-dir.",
	)
	root.PersistentFlags().String(
		flagCACert,
		cert.DefaultLoaderConfig.CACertPath,
		"The path to the CA certificate. This is relative to certs-dir.",
	)
	root.PersistentFlags().String(
		flagNodeKey,
		cert.DefaultLoaderConfig.NodeKeyPath,
		"The path to the node key. This is relative to certs-dir.",
	)
	root.PersistentFlags().String(
		flagNodeCert,
		cert.DefaultLoaderConfig.NodeCertPath,
		"The path to the node certificate. This is relative to certs-dir.",
	)
	root.PersistentFlags().Bool(
		flagAllowKeyReuse,
		*cert.DefaultFactoryConfig.AllowKeyReuse,
		"Whether to allow the reuse of CA keys for certificate generation.",
	)
	root.PersistentFlags().Int(
		flagKeySize,
		cert.DefaultFactoryConfig.KeySize,
		"The size to use for certificate key generation.",
	)
}
