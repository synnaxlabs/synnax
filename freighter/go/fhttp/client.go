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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type ClientConfig struct {
	binary.Encoder
	binary.Decoder
	ContentType string
	Accept      string
}

var _ config.Config[ClientConfig] = ClientConfig{}

// Validate validates the ClientConfig
func (c ClientConfig) Validate() error {
	v := validate.New("fhttp.ClientConfig")
	validate.NotNil(v, "encoder", c.Encoder)
	validate.NotEmptyString(v, "content_type", c.ContentType)
	validate.NotNil(v, "decoder", c.Decoder)
	validate.NotEmptyString(v, "accept", c.Accept)
	return v.Error()
}

// Override overrides valid fields with the fields in the other config.
func (c ClientConfig) Override(other ClientConfig) ClientConfig {
	c.Encoder = override.Nil(c.Encoder, other.Encoder)
	c.Decoder = override.Nil(c.Decoder, other.Decoder)
	c.ContentType = override.String(c.ContentType, other.ContentType)
	c.Accept = override.String(c.Accept, other.Accept)
	return c
}

var DefaultClientConfig = ClientConfig{
	Encoder:     binary.JSONCodec,
	Decoder:     binary.JSONCodec,
	ContentType: MIMEApplicationJSON,
	Accept:      MIMEApplicationJSON,
}
