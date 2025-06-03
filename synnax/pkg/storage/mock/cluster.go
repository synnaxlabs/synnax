// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

// Cluster is a utility for provisioning a set of storage layers in a mock cluster.
type Cluster struct {
	// Config is the configuration used to provision new stores.
	Config storage.Config
	// Stores is a slice all stores provisioned by the Cluster.
	Stores []*storage.Layer
}

// NewCluster opens a new Cluster that provisions stores using the given configuration.
func NewCluster(configs ...storage.Config) *Cluster {
	cfg, err := config.New(storage.DefaultConfig, append([]storage.Config{{
		InMemory: config.Bool(true),
		Perm:     xfs.OS_USER_RWX,
	}}, configs...)...)
	if err != nil {
		panic(err)
	}

	if !*cfg.InMemory {
		if err := os.MkdirAll(cfg.Dirname, cfg.Perm); err != nil {
			panic(err)
		}
	}

	return &Cluster{Config: cfg}
}

// Provision provisions a new store.
func (b *Cluster) Provision(ctx context.Context) (store *storage.Layer) {
	if *b.Config.InMemory {
		store = b.newMemBacked(ctx)
	} else {
		store = b.newFSBacked(ctx)
	}
	b.Stores = append(b.Stores, store)
	return store
}

// Close closes all stores provisioned by the Cluster. Close is not safe to call concurrently
// with any other Cluster or provisioned Layer methods.
func (b *Cluster) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, store := range b.Stores {
		c.Exec(store.Close)
	}
	if !*b.Config.InMemory {
		c.Exec(func() error { return os.RemoveAll(b.Config.Dirname) })
	}
	return c.Error()
}

func (b *Cluster) newMemBacked(ctx context.Context) *storage.Layer {
	store, err := storage.Open(ctx, b.Config)
	if err != nil {
		panic(err)
	}
	return store
}

func (b *Cluster) newFSBacked(ctx context.Context) *storage.Layer {
	// open a temporary directory prefixed with ServiceConfig.dirname
	tempDir, err := os.MkdirTemp(b.Config.Dirname, "delta-test-")
	if err != nil {
		panic(err)
	}
	nCfg := b.Config
	nCfg.Dirname = tempDir
	store, err := storage.Open(ctx, nCfg)
	if err != nil {
		panic(err)
	}
	return store
}
