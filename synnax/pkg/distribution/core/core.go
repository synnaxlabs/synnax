// Copyright 2023 Synnax Labs, Inc.
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
//  1. Storage.TSChannel - A time-series storage engine for writing node-local telemetry
//     frames.
//  2. Storage.KV - An eventually consistent, key-value store for maintaining cluster
//     wide metadata and state.
//  3. Cluster - An API for querying information about the Cluster topology.
type Core struct {
	Config
	// Cluster is the API for the delta Cluster.
	Cluster Cluster
	// Storage is the storage for the node. The distribution layer replaces the original
	// key-value store with a distributed key-value store. The caller should NOT call
	// Close on the storage engine.
	Storage *storage.Storage
}

// Open opens a new  core distribution layer. The caller is responsible for closing the
// distribution layer when it is no longer in use.
func Open(ctx context.Context, configs ...Config) (c Core, err error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return c, err
	}

	cfg.Storage.Instrumentation = cfg.Instrumentation.Child("storage")
	c.Config = cfg
	c.Storage, err = storage.Open(cfg.Storage)
	if err != nil {
		return c, err
	}

	clusterTransport := aspentransport.New(cfg.Pool)
	*cfg.Transports = append(*cfg.Transports, clusterTransport)

	// Since we're using our own key-value engine, the value we use for 'dirname'
	// doesn't matter.
	clusterDB, err := aspen.Open(
		ctx,
		/* dirname */ "",
		cfg.AdvertiseAddress,
		cfg.PeerAddresses,
		aspen.WithEngine(c.Storage.KV),
		aspen.WithTransport(clusterTransport),
		aspen.WithInstrumentation(c.Instrumentation.Child("aspen")),
	)
	if err != nil {
		return c, err
	}
	c.Cluster = clusterDB.Cluster
	// Replace storage's key-value store with a distributed version.
	c.Storage.KV = clusterDB
	return c, nil
}

// Close closes the core of the distribution layer, shutting down cluster operations
// and closing the storage layer.
func (c Core) Close() error {
	// Because we embedded aspen as the storage KV, we can just shut down cluster
	// operations here as well.
	return c.Storage.Close()
}
