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
	ContentType string
	Decoders    map[string]binary.Decoder
}

var _ config.Config[ClientConfig] = ClientConfig{}

// Validate validates the ClientConfig
func (c ClientConfig) Validate() error {
	v := validate.New("fhttp.ClientConfig")
	validate.NotNil(v, "encoder", c.Encoder)
	validate.NotEmptyString(v, "content_type", c.ContentType)
	validate.NotNil(v, "decoders", c.Decoders)
	return v.Error()
}

// Override overrides valid fields with the fields in the other config.
func (c ClientConfig) Override(other ClientConfig) ClientConfig {
	c.Encoder = override.Nil(c.Encoder, other.Encoder)
	c.Decoders = override.Nil(c.Decoders, other.Decoders)
	c.ContentType = override.String(c.ContentType, other.ContentType)
	return c
}

var DefaultClientConfig = ClientConfig{
	Encoder:     binary.JSONCodec,
	Decoders:    map[string]binary.Decoder{MIMEApplicationJSON: binary.JSONCodec},
	ContentType: MIMEApplicationJSON,
}
