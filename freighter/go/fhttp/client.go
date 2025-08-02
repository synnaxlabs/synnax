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
	contentType string
}

var _ config.Config[ClientConfig] = ClientConfig{}

func (c ClientConfig) Validate() error {
	v := validate.New("fhttp.ClientConfig")
	validate.NotNil(v, "codec", c.Codec)
	validate.NotEmptyString(v, "content_type", c.contentType)
	return v.Error()
}

func (c ClientConfig) Override(other ClientConfig) ClientConfig {
	c.Codec = override.Nil(c.Codec, other.Codec)
	c.contentType = override.String(c.contentType, other.contentType)
	return c
}

var DefaultClientConfig = ClientConfig{
	Codec:       &binary.MsgPackCodec{},
	contentType: "application/msgpack",
}
