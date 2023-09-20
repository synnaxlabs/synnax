// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cdc

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Service struct{ Config }

type Config struct {
	alamos.Instrumentation
	Channel channel.Writeable
	Framer  framer.Writable
	DB      *gorp.DB
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("CDC")
	validate.NotNil(v, "Channel", c.Channel)
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "DB", c.DB)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.DB = override.Nil(c.DB, other.DB)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

func New(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{Config: cfg}, nil
}
