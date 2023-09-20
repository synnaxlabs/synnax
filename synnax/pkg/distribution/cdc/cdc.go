// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cdc implements change data capture services that allow consumers to publish
// changes to their data through free channels. These changes can then be subscribed to
// using the standard channel streaming pipeline, allowing Synnax users to execute
// arbitrary code in response to changes a cluster's data and metadata.
//
// Users of this library should typically use the OpenGorp function, which provide CDC
// functionality for changes to a gorp compatible key-value store with a simple
// configuration. The most common of these is the UUIDGorpConfig function, which
// provides a config for propagating sets and deletes for UUID keyed gorp entries.
package cdc

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

type Service struct{ Config }

// Config is the configuration for opening the core CDC Service.
type Config struct {
	// Instrumentation is used for logging, tracing, and metrics.
	// [OPTIONAL]
	alamos.Instrumentation
	// Channel is the service used for retrieving and creating free channels.
	// [REQUIRED]
	Channel channel.Writeable
	// Framer is the service used for writing frames containing changes to the Synnax
	// telemetry pipeline.
	// [REQUIRED]
	Framer framer.Writable
}

var (
	_ config.Config[Config] = Config{}
	// DefaultConfig is the default configuration for the CDC Service.
	DefaultConfig = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("CDC")
	validate.NotNil(v, "Channel", c.Channel)
	validate.NotNil(v, "Framer", c.Framer)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// New creates a new CDC Service using the given configuration. This service is
// stateless, and does not need to be closed.
func New(cfgs ...Config) (*Service, error) {
	cfg, err := config.New(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Service{Config: cfg}, nil
}
