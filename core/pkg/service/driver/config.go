// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Config struct {
	// Factory is the factory for creating tasks.
	Factory Factory
	// Host is the node key of the current host.
	Host cluster.HostProvider
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
	// Status is the status service for task status updates.
	Status *status.Service
	// HeartbeatInterval is the interval at which the driver reports its health.
	HeartbeatInterval time.Duration
	alamos.Instrumentation
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{HeartbeatInterval: 1 * time.Second}
)

func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.DB = override.Nil(c.DB, other.DB)
	c.Rack = override.Nil(c.Rack, other.Rack)
	c.Task = override.Nil(c.Task, other.Task)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Status = override.Nil(c.Status, other.Status)
	c.Factory = override.Nil(c.Factory, other.Factory)
	c.Host = override.Nil(c.Host, other.Host)
	c.HeartbeatInterval = override.Numeric(c.HeartbeatInterval, other.HeartbeatInterval)
	return c
}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("go_driver")
	validate.NotNil(v, "db", c.DB)
	validate.NotNil(v, "rack", c.Rack)
	validate.NotNil(v, "task", c.Task)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "status", c.Status)
	validate.NotNil(v, "factory", c.Factory)
	validate.NotNil(v, "host", c.Host)
	return v.Error()
}
