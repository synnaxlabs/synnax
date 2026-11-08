// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package signals writes changes to the data of the Core to free channels, which
// clients stream through the standard channel streaming pipeline. Each service
// publishes the set, delete, and action channels of its tables with PublishTable and
// PublishActions, and the service layer calls Start once every service is open. From
// then on, the Provider writes each commit of the DB as one frame across every channel
// that the commit changed.
package signals

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Provider publishes changes to the data of the Core on free channels. Pass it to each
// service that publishes a change channel. PublishTable, PublishActions, and Start must
// run on one goroutine.
type Provider struct {
	cfg Config
	// feeds write the samples of the channels in keys.
	feeds []feed
	keys  channel.Keys
	// started is true once Start runs.
	started bool
}

// Config is the configuration for opening the core Signals Provider.
type Config struct {
	// DB is the database whose commits the Provider writes. Channel creates run in its
	// transactions.
	//
	// [REQUIRED]
	DB *gorp.DB
	// Channel is the service used for retrieving and creating free channels.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer is the service used for writing frames containing changes to the Synnax
	// telemetry pipeline.
	//
	// [REQUIRED]
	Framer *framer.Service
	// Instrumentation is used for logging, tracing, and metrics.
	//
	// [OPTIONAL]
	alamos.Instrumentation
}

var _ config.Config[Config] = Config{}

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("signals")
	v.NotNil("db", c.DB)
	v.NotNil("channel", c.Channel)
	v.NotNil("framer", c.Framer)
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

// New creates a new Signals Provider using the given configuration. The Provider writes
// nothing until Start.
func New(cfgs ...Config) (*Provider, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Provider{cfg: cfg}, nil
}
