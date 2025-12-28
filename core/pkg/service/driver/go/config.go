// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package godriver

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	alamos.Instrumentation
	// DB is the gorp database for observing task changes.
	DB *gorp.DB
	// Rack is the rack service for creating/managing racks.
	Rack *rack.Service
	// Task is the task service for managing tasks.
	Task *task.Service
	// Framer is the framer service for streaming commands.
	Framer *framer.Service
	// Channel is the channel service for retrieving command channels.
	Channel *channel.Service
	// Factory is the factory for creating tasks.
	Factory Factory
	// HostKey is the node key of the current host.
	HostKey cluster.NodeKey
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Task = override.Nil(c.Task, other.Task)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Factory = override.Nil(c.Factory, other.Factory)
	c.HostKey = override.Numeric(c.HostKey, other.HostKey)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("godriver")
	validate.NotNil(v, "DB", c.DB)
	validate.NotNil(v, "Rack", c.Rack)
	validate.NotNil(v, "Task", c.Task)
	validate.NotNil(v, "Framer", c.Framer)
	validate.NotNil(v, "Channel", c.Channel)
	validate.NotNil(v, "Factory", c.Factory)
	validate.NonZero(v, "HostKey", c.HostKey)
	return v.Error()
}
