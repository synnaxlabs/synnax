// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package runtime

import (
	"context"

	"github.com/synnaxlabs/slate/module"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"github.com/tetratelabs/wazero"
)

type Runtime struct {
	wasm wazero.Runtime
}

type Config struct {
	Module  *module.Module
	Channel channel.Readable
	Framer  *framer.Service
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Module = override.Nil(c.Module, other.Module)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("slate.runtime")
	validate.NotNil(v, "module", c.Module)
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	return v.Error()
}

func New(ctx context.Context, cfgs ...Config) (*Runtime, error) {
	_, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	r := &Runtime{wasm: wazero.NewRuntime(ctx)}
	return r, err
}
