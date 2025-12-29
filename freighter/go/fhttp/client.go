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

type ClientFactoryConfig struct {
	Codec httputil.Codec
}

func (c ClientFactoryConfig) Validate() error {
	v := validate.New("ws.stream_client")
	validate.NotNil(v, "codec", c.Codec)
	return v.Error()
}

func (c ClientFactoryConfig) Override(other ClientFactoryConfig) ClientFactoryConfig {
	c.Codec = override.Nil(c.Codec, other.Codec)
	return c
}

var DefaultClientConfig = ClientFactoryConfig{
	Codec: httputil.MsgPackCodec,
}

type ClientFactory struct {
	ClientFactoryConfig
}

func NewClientFactory(configs ...ClientFactoryConfig) *ClientFactory {
	cfg, err := config.New(DefaultClientConfig, configs...)
	if err != nil {
		panic(err)
	}
	return &ClientFactory{ClientFactoryConfig: cfg}
}

func StreamClient[RQ, RS freighter.Payload](c *ClientFactory) freighter.StreamClient[RQ, RS] {
	return &streamClient[RQ, RS]{codec: c.Codec}
}

func UnaryClient[RQ, RS freighter.Payload](c *ClientFactory) freighter.UnaryClient[RQ, RS] {
	return &unaryClient[RQ, RS]{codec: c.Codec}
}
