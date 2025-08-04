// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gossip

import (
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config sets specific parameters for the gossip service. See DefaultConfig
// for default values. It implements the config.ServiceConfig interface.
type Config struct {
	alamos.Instrumentation
	// TransportClient is the transport used to exchange gossip between nodes.
	// [Required]
	TransportClient TransportClient
	// TransportServer is the transport used to exchange gossip between nodes.
	// [Required]
	TransportServer TransportServer
	// Store is where cluster state will be synchronized to and from.
	// [Required]
	Store store.Store
	// Interval is the interval at which a node will gossip its state.
	Interval time.Duration
}

// Override implements the config.ServiceConfig interface.
func (c Config) Override(other Config) Config {
	c.Interval = override.Numeric(c.Interval, other.Interval)
	c.TransportClient = override.Nil(c.TransportClient, other.TransportClient)
	c.TransportServer = override.Nil(c.TransportServer, other.TransportServer)
	c.Store = override.Nil(c.Store, other.Store)
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	return c
}

// Validate implements the config.ServiceConfig interface.
func (c Config) Validate() error {
	v := validate.New("gossip")
	validate.NotNil(v, "transport_client", c.TransportClient)
	validate.NotNil(v, "transport_server", c.TransportServer)
	validate.NotNil(v, "store", c.Store)
	validate.Positive(v, "interval", c.Interval)
	return v.Error()
}

// Report implements the alamos.ReportProvider interface. Assumes the config is valid.
func (c Config) Report() alamos.Report {
	return alamos.Report{
		"interval":         c.Interval,
		"transport_client": c.TransportClient.Report(),
		"transport_server": c.TransportServer.Report(),
	}
}

var (
	DefaultConfig = Config{
		Interval: 1 * time.Second,
	}
	FastConfig = DefaultConfig.Override(Config{
		Interval: 50 * time.Millisecond,
	})
)
