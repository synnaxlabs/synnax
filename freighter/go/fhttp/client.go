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
	binary.Codec
	ContentType string
}

var _ config.Config[ClientConfig] = ClientConfig{}

// Validate validates the ClientConfig
func (c ClientConfig) Validate() error {
	v := validate.New("fhttp.ClientConfig")
	validate.NotNil(v, "codec", c.Codec)
	validate.NotEmptyString(v, "content_type", c.ContentType)
	return v.Error()
}

// Override overrides valid fields with the fields in the other config.
func (c ClientConfig) Override(other ClientConfig) ClientConfig {
	c.Codec = override.Nil(c.Codec, other.Codec)
	c.ContentType = override.String(c.ContentType, other.ContentType)
	return c
}

var DefaultClientConfig = ClientConfig{
	Codec:       binary.JSONCodec,
	ContentType: MIMEApplicationJSON,
}
