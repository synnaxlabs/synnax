// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package signals implements change data capture services that allow consumers to publish
// changes to their data through free channels. These changes can then be subscribed to
// using the standard channel streaming pipeline, allowing Synnax users to execute
// arbitrary code in response to changes a cluster's data and metadata.
//
// Users of this library should typically use the PublishFromGorp function, which provide Signals
// functionality for changes to a gorp compatible key-value store with a simple
// configuration. The most common of these is the GorpPublisherConfigUUID function, which
// provides a config for propagating sets and deletes for UUID keyed gorp entries.
package signals

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Provider implements the core functionality for opening signal pipelines. It should be
// passed around as an argument to any services that need to open signal pipelines. It is
// stateless, and does not need to be closed.
type Provider struct{ Config }

// Config is the configuration for opening the core Signals Provider.
type Config struct {
	// Instrumentation is used for logging, tracing, and metrics.
	// [OPTIONAL]
	alamos.Instrumentation
	// Channel is the service used for retrieving and creating free channels.
	//
	// [REQUIRED]
	Channel *channel.Service
	// Framer is the service used for writing frames containing changes to the Synnax
	// telemetry pipeline.
	// [REQUIRED]
	Framer *framer.Service
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for the Signals Provider.
	DefaultConfig = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("signals")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "framer", c.Framer)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// New creates a new Signals Provider using the given configuration. This service is
// stateless, and does not need to be closed.
func New(cfgs ...Config) (*Provider, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Provider{Config: cfg}, nil
}
