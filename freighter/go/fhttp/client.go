// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp

import (
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/httputil"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type ClientFactoryConfig struct{ Codec httputil.Codec }

var _ config.Config[ClientFactoryConfig] = ClientFactoryConfig{}

func (cfc ClientFactoryConfig) Validate() error {
	v := validate.New("[ws.streamClient]")
	validate.NotNil(v, "Codec", cfc.Codec)
	return v.Error()
}

func (cfc ClientFactoryConfig) Override(other ClientFactoryConfig) ClientFactoryConfig {
	cfc.Codec = override.Nil(cfc.Codec, other.Codec)
	return cfc
}

var DefaultClientConfig = ClientFactoryConfig{Codec: httputil.MsgPackCodec}

type ClientFactory struct{ ClientFactoryConfig }

func NewClientFactory(cfgs ...ClientFactoryConfig) *ClientFactory {
	cfg, err := config.New(DefaultClientConfig, cfgs...)
	if err != nil {
		panic(err)
	}
	return &ClientFactory{ClientFactoryConfig: cfg}
}

func StreamClient[RQ, RS freighter.Payload](cf *ClientFactory) freighter.StreamClient[RQ, RS] {
	return &streamClient[RQ, RS]{codec: cf.Codec}
}

func UnaryClient[RQ, RS freighter.Payload](cf *ClientFactory) freighter.UnaryClient[RQ, RS] {
	return &unaryClient[RQ, RS]{codec: cf.Codec}
}
