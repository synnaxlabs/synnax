// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import (
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	pledge_ "github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

const FlushOnEvery = -1 * time.Second

type Config struct {
	alamos.Instrumentation
	// HostAddress is the reachable address of the host node.
	// [REQUIRED]
	HostAddress address.Address
	// Storage is a key-value storage backend for the Cluster. Cluster will flush
	// changes to its state to this backend based on Config.StorageFlushInterval.
	// Open will also attempt to load an existing Cluster from this backend.
	// If Config.Storage is not provided, Cluster state will only be stored in memory.
	Storage kv.DB
	// StorageKey is the key used to store the Cluster state in the backend.
	StorageKey []byte
	// StorageFlushInterval	is the interval at which the Cluster state is flushed
	// to the backend. If this is set to FlushOnEvery, the Cluster state is flushed on
	// every change.
	StorageFlushInterval time.Duration
	// Gossip is the configuration for propagating Cluster state through gossip.
	// See the gossip package for more details on how to configure this.
	Gossip gossip.Config
	// Pledge is the configuration for pledging to the Cluster upon a Open call.
	// See the pledge package for more details on how to configure this.
	Pledge pledge_.Config
	// Codec is the encoder/decoder to use for encoding and decoding the
	// Cluster state.
	Codec binary.Codec
}

var _ config.Config[Config] = Config{}

// Override implements config.Config.
func (cfg Config) Override(other Config) Config {
	cfg.HostAddress = override.String(cfg.HostAddress, other.HostAddress)
	cfg.Codec = override.Nil(cfg.Codec, other.Codec)
	cfg.StorageFlushInterval = override.Numeric(cfg.StorageFlushInterval, other.StorageFlushInterval)
	cfg.StorageKey = override.Slice(cfg.StorageKey, other.StorageKey)
	cfg.Storage = override.Nil(cfg.Storage, other.Storage)
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.Gossip = cfg.Gossip.Override(other.Gossip)
	cfg.Pledge = cfg.Pledge.Override(other.Pledge)
	return cfg
}

// Validate implements config.Config.
func (cfg Config) Validate() error {
	v := validate.New("aspen.cluster")
	validate.NotEmptyString(v, "host_address", cfg.HostAddress)
	validate.NotNil(v, "codec", cfg.Codec)
	validate.NonZero(v, "storage_flush_interval", cfg.StorageFlushInterval)
	validate.NotEmptySlice(v, "local_key", cfg.StorageKey)
	return v.Error()
}

// Report implements the alamos.ReportProvider interface.
func (cfg Config) Report() alamos.Report {
	report := make(alamos.Report)
	if cfg.Storage != nil {
		report["storage"] = cfg.Storage.Report()
	} else {
		report["storage"] = "not provided"
	}
	report["storage_key"] = string(cfg.StorageKey)
	report["storage_flush_interval"] = cfg.StorageFlushInterval
	return report
}

var (
	DefaultConfig = Config{
		Pledge:               pledge_.DefaultConfig,
		StorageKey:           []byte("aspen.cluster"),
		Gossip:               gossip.DefaultConfig,
		StorageFlushInterval: 1 * time.Second,
		// This used to be implemented by a gob codec, but we want to switch to msgpack.
		// Instead, we will use a fallback codec that tries msgpack to decode first, then gob.
		Codec: binary.NewDecodeFallbackCodec(&binary.JSONCodec{}, &binary.GobCodec{}),
	}
	FastConfig = DefaultConfig.Override(Config{
		Pledge: pledge_.FastConfig,
		Gossip: gossip.FastConfig,
	})
	BlazingFastConfig = DefaultConfig.Override(Config{
		Pledge: pledge_.BlazingFastConfig,
		Gossip: gossip.FastConfig,
	})
)
