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
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/security/cert"
	"github.com/synnaxlabs/x/address"
)

// BuildLoaderConfig builds a cert.LoaderConfig using the viper configuration.
func BuildLoaderConfig(ins alamos.Instrumentation) cert.LoaderConfig {
	return cert.LoaderConfig{
		Instrumentation: ins,
		CertsDir:        viper.GetString(FlagCertsDir),
		CAKeyPath:       viper.GetString(FlagCAKey),
		CACertPath:      viper.GetString(FlagCACert),
		NodeKeyPath:     viper.GetString(FlagNodeKey),
		NodeCertPath:    viper.GetString(FlagNodeCert),
	}
}

// BuildCertFactoryConfig builds a cert.FactoryConfig using the viper configuration.
func BuildCertFactoryConfig(
	ins alamos.Instrumentation,
	hosts ...address.Address,
) cert.FactoryConfig {
	return cert.FactoryConfig{
		LoaderConfig:  BuildLoaderConfig(ins),
		AllowKeyReuse: new(viper.GetBool(FlagAllowKeyReuse)),
		KeySize:       viper.GetInt(FlagKeySize),
		Hosts:         hosts,
	}
}

// GenerateAuto generates a CA certificate and a certificate for the Core.
func GenerateAuto(cfg cert.FactoryConfig) error {
	factory, err := cert.NewFactory(cfg)
	if err != nil {
		return err
	}
	if err = factory.CreateCAPairIfMissing(); err != nil {
		return err
	}
	return factory.CreateNodePairIfMissing()
}
