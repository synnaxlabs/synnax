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
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"os"
)

// Builder is a utility for provisioning mock stores.
type Builder struct {
	// Config is the configuration used to provision new stores.
	Config storage.Config
	// Stores is a slice all stores provisioned by the Builder.
	Stores []*storage.Storage
}

// NewBuilder opens a new Builder that provisions stores using the given configuration.
func NewBuilder(configs ...storage.Config) *Builder {
	cfg, err := config.New(storage.DefaultConfig, append([]storage.Config{{
		MemBacked: config.Bool(true),
		Perm:      xfs.OS_USER_RWX,
	}}, configs...)...)
	if err != nil {
		panic(err)
	}

	if !*cfg.MemBacked {
		if err := os.MkdirAll(cfg.Dirname, cfg.Perm); err != nil {
			panic(err)
		}
	}

	return &Builder{Config: cfg}
}

// New provisions a new store.
func (b *Builder) New() (store *storage.Storage) {
	if *b.Config.MemBacked {
		store = b.newMemBacked()
	} else {
		store = b.newFSBacked()
	}
	b.Stores = append(b.Stores, store)
	return store
}

// Cleanup removes all test data written to disk by the stores provisioned by the Builder.
// Cleanup should only be called after Close, and is not safe to call concurrently
// with any other Builder or Storage methods.
func (b *Builder) Cleanup() error {
	if *b.Config.MemBacked {
		return nil
	}
	return os.RemoveAll(b.Config.Dirname)
}

// Close closes all stores provisioned by the Builder. Close is not safe to call concurrently
// with any other Builder or provisioned Storage methods.
func (b *Builder) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, store := range b.Stores {
		c.Exec(store.Close)
	}
	return c.Error()
}

func (b *Builder) newMemBacked() *storage.Storage {
	store, err := storage.Open(b.Config)
	if err != nil {
		panic(err)
	}
	return store
}

func (b *Builder) newFSBacked() *storage.Storage {
	// open a temporary directory prefixed with ServiceConfig.dirname
	tempDir, err := os.MkdirTemp(b.Config.Dirname, "delta-test-")
	if err != nil {
		panic(err)
	}
	nCfg := b.Config
	nCfg.Dirname = tempDir
	store, err := storage.Open(nCfg)
	if err != nil {
		panic(err)
	}
	return store
}
