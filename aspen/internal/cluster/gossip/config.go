// Copyright 2026 Synnax Labs, Inc.
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
	// TransportClient is the transport used to exchange gossip between nodes.
	// [Required]
	TransportClient TransportClient
	// TransportServer is the transport used to exchange gossip between nodes.
	// [Required]
	TransportServer TransportServer
	// Store is where cluster state will be synchronized to and from.
	// [Required]
	Store store.Store
	alamos.Instrumentation
	// Interval is the interval at which a node will gossip its state.
	Interval time.Duration
}

// Override implements the config.ServiceConfig interface.
func (cfg Config) Override(other Config) Config {
	cfg.Interval = override.Numeric(cfg.Interval, other.Interval)
	cfg.TransportClient = override.Nil(cfg.TransportClient, other.TransportClient)
	cfg.TransportServer = override.Nil(cfg.TransportServer, other.TransportServer)
	cfg.Store = override.Nil(cfg.Store, other.Store)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	return cfg
}

// Validate implements the config.ServiceConfig interface.
func (cfg Config) Validate() error {
	v := validate.New("gossip")
	validate.NotNil(v, "transport_client", cfg.TransportClient)
	validate.NotNil(v, "transport_server", cfg.TransportServer)
	validate.NotNil(v, "store", cfg.Store)
	validate.Positive(v, "interval", cfg.Interval)
	return v.Error()
}

// Report implements the alamos.ReportProvider interface. Assumes the config is valid.
func (cfg Config) Report() alamos.Report {
	return alamos.Report{
		"interval":         cfg.Interval,
		"transport_client": cfg.TransportClient.Report(),
		"transport_server": cfg.TransportServer.Report(),
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
