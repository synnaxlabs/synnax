// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"
	"os"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
)

// Cluster is a utility for provisioning a set of storage layers in a mock cluster.
// The provisioned
type Cluster struct {
	// cfg is the configuration used to provision new stores.
	cfg storage.Config
	// Stores is a slice all stores provisioned by the Cluster.
	Stores []*storage.Layer
}

// NewCluster opens a new Cluster that provisions stores using the given configuration.
func NewCluster(configs ...storage.Config) *Cluster {
	cfg := lo.Must(config.New(storage.DefaultConfig, append([]storage.Config{{
		InMemory: config.True(),
	}}, configs...)...))
	if !*cfg.InMemory {
		lo.Must0(os.MkdirAll(cfg.Dirname, cfg.Perm))
	}

	return &Cluster{cfg: cfg}
}

// Provision provisions a new independent storage layer.
func (b *Cluster) Provision(ctx context.Context) (store *storage.Layer) {
	if *b.cfg.InMemory {
		store = b.newMemBacked(ctx)
	} else {
		store = b.newFSBacked(ctx)
	}
	b.Stores = append(b.Stores, store)
	return store
}

// Close closes all stores provisioned by the Cluster. Close is not safe to call
// concurrently with any other Cluster or provisioned storage.Layer methods.
func (b *Cluster) Close() error {
	var c errors.Catcher
	for _, store := range b.Stores {
		c.Exec(store.Close)
	}
	if !*b.cfg.InMemory {
		c.Exec(func() error { return os.RemoveAll(b.cfg.Dirname) })
	}
	return c.Error()
}

func (b *Cluster) newMemBacked(ctx context.Context) *storage.Layer {
	return lo.Must(storage.Open(ctx, b.cfg))
}

func (b *Cluster) newFSBacked(ctx context.Context) *storage.Layer {
	// open a temporary directory prefixed with ServiceConfig.dirname
	tempDir := lo.Must(os.MkdirTemp(b.cfg.Dirname, "delta-test-"))
	nCfg := b.cfg
	nCfg.Dirname = tempDir
	return lo.Must(storage.Open(ctx, nCfg))
}
