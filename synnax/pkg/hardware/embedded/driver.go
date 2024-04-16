// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package embedded

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"io"
	"os/exec"
)

type Config struct {
	alamos.Instrumentation
	Address      address.Address `json:"address"`
	RackName     string          `json:"rack_name"`
	Integrations []string        `json:"integrations"`
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{
		Integrations: make([]string, 0),
	}
)

func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Address = override.String(c.Address, other.Address)
	c.RackName = override.String(c.RackName, other.RackName)
	c.Integrations = override.Slice(c.Integrations, other.Integrations)
	return c
}

func (c Config) Validate() error { return nil }

type Driver struct {
	cmd      *exec.Cmd
	shutdown io.Closer
}
