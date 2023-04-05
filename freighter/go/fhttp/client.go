// Copyright 2023 Synnax Labs, Inc.
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
	"go.uber.org/zap"
)

type ClientFactoryConfig struct {
	EncoderDecoder httputil.EncoderDecoder
	Logger         *zap.SugaredLogger
}

func (c ClientFactoryConfig) Validate() error {
	v := validate.New("[ws.streamClient]")
	validate.NotNil(v, "EncoderDecoder", c.EncoderDecoder)
	return v.Error()
}

func (c ClientFactoryConfig) Override(other ClientFactoryConfig) ClientFactoryConfig {
	c.EncoderDecoder = override.Nil(c.EncoderDecoder, other.EncoderDecoder)
	c.Logger = override.Nil(c.Logger, other.Logger)
	return c
}

var DefaultClientConfig = ClientFactoryConfig{
	EncoderDecoder: httputil.MsgPackEncoderDecoder,
	Logger:         zap.S(),
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
	return &streamClient[RQ, RS]{
		logger: c.Logger,
		ecd:    c.EncoderDecoder,
	}
}

func UnaryPostClient[RQ, RS freighter.Payload](c *ClientFactory) freighter.UnaryClient[RQ, RS] {
	return &unaryClient[RQ, RS]{
		ecd: c.EncoderDecoder,
	}
}
