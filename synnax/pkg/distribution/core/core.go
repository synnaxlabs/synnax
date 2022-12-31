// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package core provides entities for managing distributed storage and networking in a
// synnax Cluster. It serves as the base for the larger distribution layer.
package core

import (
	"context"
	"github.com/synnaxlabs/aspen"
	aspentransport "github.com/synnaxlabs/aspen/transport/grpc"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
)

// Core is the foundational primitive for distributed compute in a synnax Cluster. It
// exposes the following services:
//
//  1. Storage.TSChannel - A time-series storage engine for writing node-local telemetry frames.
//  2. Storage.KV - An eventually consistent, key-value store for maintaining cluster
//     wide meta-data and state.
//  3. Cluster - An API for querying information about the Cluster topology.
type Core struct {
	// Config is the configuration for the distribution layer.
	Config Config
	// Cluster is the API for the delta Cluster.
	Cluster Cluster
	// Storage is the storage for the node. The distribution layer replaces the original
	// key-value store with a distributed key-value store. The caller should NOT call
	// Close on the storage engine.
	Storage *storage.Store
}

// Open opens a new  core distribution layer. The caller is responsible for closing the
// distribution layer when it is no longer in use.
func Open(ctx context.Context, cfg Config) (c Core, err error) {
	cfg, err = config.OverrideAndValidate(DefaultConfig, cfg)
	if err != nil {
		return c, err
	}

	c.Storage, err = storage.Open(cfg.Storage)
	if err != nil {
		return c, err
	}

	clusterTransport := aspentransport.New(cfg.Pool)
	*cfg.Transports = append(*cfg.Transports, clusterTransport)

	// Since we're using our own key-value engine, the value we use for 'dirname'
	// doesn't matter.
	clusterKV, err := aspen.Open(
		ctx,
		/* dirname */ "",
		cfg.AdvertiseAddress,
		cfg.PeerAddresses,
		aspen.WithEngine(c.Storage.KV),
		aspen.WithExperiment(cfg.Experiment),
		aspen.WithLogger(cfg.Logger.Named("aspen").Sugar()),
		aspen.WithTransport(clusterTransport),
	)
	c.Cluster = clusterKV

	// Replace storage's key-value store with a distributed version.
	c.Storage.KV = clusterKV

	return c, err
}
