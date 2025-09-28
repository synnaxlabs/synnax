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
	configFlag            = "config"
	versionFlag           = "version"
	certsDirFlag          = "certs-dir"
	caKeyFlag             = "ca-key"
	caCertFlag            = "ca-cert"
	nodeKeyFlag           = "node-key"
	nodeCertFlag          = "node-cert"
	allowKeyReuseFlag     = "allow-key-reuse"
	keySizeFlag           = "key-size"
	verboseFlag           = "verbose"
	debugFlag             = "debug"
	logFilePathFlag       = "log-file-path"
	logFileMaxSizeFlag    = "log-file-max-size"
	logFileMaxBackupsFlag = "log-file-max-backups"
	logFileMaxAgeFlag     = "log-file-max-age"
	logFileCompressFlag   = "log-file-compress"
)

func configureRootFlags() {
	root.PersistentFlags().StringP(
		configFlag,
		"c",
		"/usr/local/synnax/config.yaml",
		"config file",
	)

	root.Flags().Bool(
		versionFlag,
		false,
		"Print the version of Synnax",
	)

	root.PersistentFlags().String(
		certsDirFlag,
		cert.DefaultLoaderConfig.CertsDir,
		"The directory where certificates should be stored and/or written to.",
	)

	root.PersistentFlags().String(
		caKeyFlag,
		cert.DefaultLoaderConfig.CAKeyPath,
		"The path to the CA key. This is relative to certs-dir.",
	)

	root.PersistentFlags().String(
		caCertFlag,
		cert.DefaultLoaderConfig.CACertPath,
		"The path to the CA certificate. This is relative to certs-dir.",
	)

	root.PersistentFlags().String(
		nodeKeyFlag,
		cert.DefaultLoaderConfig.NodeKeyPath,
		"The path to the node key. This is relative to certs-dir.",
	)

	root.PersistentFlags().String(
		nodeCertFlag,
		cert.DefaultLoaderConfig.NodeCertPath,
		"The path to the node certificate. This is relative to certs-dir.",
	)

	root.PersistentFlags().Bool(
		allowKeyReuseFlag,
		*cert.DefaultFactoryConfig.AllowKeyReuse,
		"Whether to allow the reuse of CA keys for certificate generation.",
	)

	root.PersistentFlags().Int(
		keySizeFlag,
		cert.DefaultFactoryConfig.KeySize,
		"The size to use for certificate key generation.",
	)

	root.PersistentFlags().BoolP(
		verboseFlag,
		"v",
		false,
		"Enable verbose debugging.",
	)

	root.PersistentFlags().Bool(
		debugFlag,
		false,
		"Enable debug logging.",
	)

	root.PersistentFlags().String(
		logFilePathFlag,
		"./synnax-logs/synnax.log",
		"Log file path",
	)

	root.PersistentFlags().Int(
		logFileMaxSizeFlag,
		50,
		"Maximum size of log file in megabytes",
	)

	root.PersistentFlags().Int(
		logFileMaxBackupsFlag,
		5,
		"Maximum number of log files to retain",
	)

	root.PersistentFlags().Int(
		logFileMaxAgeFlag,
		30,
		"Maximum age of log files to retain",
	)

	root.PersistentFlags().Bool(
		logFileCompressFlag,
		false,
		"Compress log files",
	)
}
