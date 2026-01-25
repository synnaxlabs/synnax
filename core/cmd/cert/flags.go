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
	"github.com/synnaxlabs/synnax/cmd/instrumentation"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
)

// Flag names used for creating certificates.
const (
	FlagCertsDir      = "certs-dir"
	FlagCAKey         = "ca-key"
	FlagCACert        = "ca-cert"
	FlagNodeKey       = "node-key"
	FlagNodeCert      = "node-cert"
	FlagAllowKeyReuse = "allow-key-reuse"
	FlagKeySize       = "key-size"
)

// AddFlags adds the cert flags to the given command.
func AddFlags(cmd *cobra.Command) {
	instrumentation.AddFlags(cmd)
	cmd.Flags().String(
		FlagCertsDir,
		cert.DefaultLoaderConfig.CertsDir,
		"The directory where certificates should be stored and/or written to",
	)
	cmd.Flags().String(
		FlagCAKey,
		cert.DefaultLoaderConfig.CAKeyPath,
		"The path to the CA key, relative to certs-dir",
	)
	cmd.Flags().String(
		FlagCACert,
		cert.DefaultLoaderConfig.CACertPath,
		"The path to the CA certificate, relative to certs-dir",
	)
	cmd.Flags().String(
		FlagNodeKey,
		cert.DefaultLoaderConfig.NodeKeyPath,
		"The path to the node key, relative to certs-dir",
	)
	cmd.Flags().String(
		FlagNodeCert,
		cert.DefaultLoaderConfig.NodeCertPath,
		"The path to the node certificate, relative to certs-dir",
	)
	cmd.Flags().Bool(
		FlagAllowKeyReuse,
		*cert.DefaultFactoryConfig.AllowKeyReuse,
		"Whether to allow the reuse of CA keys for certificate generation",
	)
	cmd.Flags().Int(
		FlagKeySize,
		cert.DefaultFactoryConfig.KeySize,
		"The size to use for certificate key generation",
	)
}
