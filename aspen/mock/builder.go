// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"os"
	"path/filepath"
	"strconv"
)

type Builder struct {
	PortRangeStart  int
	DataDir         string
	DefaultOptions  []aspen.Option
	peerAddresses   []address.Address
	TmpDirs         map[aspen.NodeKey]string
	tmpDir          string
	_addressFactory *address.Factory
	Nodes           map[aspen.NodeKey]NodeInfo
	memBacked       bool
}

type NodeInfo struct {
	Addr address.Address
	Dir  string
	DB   *aspen.DB
}

func (b *Builder) Dir() string {
	if b.tmpDir == "" {
		var err error
		b.tmpDir, err = os.MkdirTemp(b.DataDir, "aspen")
		if err != nil {
			panic(err)
		}
	}
	return b.tmpDir
}

func (b *Builder) addressFactory() *address.Factory {
	if b._addressFactory == nil {
		b._addressFactory = address.NewLocalFactory(b.PortRangeStart)
	}
	return b._addressFactory
}

func (b *Builder) New(opts ...aspen.Option) (*aspen.DB, error) {
	dir := filepath.Join(b.Dir(), strconv.Itoa(len(b.peerAddresses)))
	if len(b.Nodes) == 0 {
		opts = append(opts, aspen.Bootstrap())
	}
	addr := b.addressFactory().Next()
	db, err := aspen.Open(context.TODO(), dir, addr, b.peerAddresses, append(b.DefaultOptions, opts...)...)
	if err != nil {
		return nil, err
	}
	b.Nodes[db.Cluster.HostKey()] = NodeInfo{
		Addr: addr,
		Dir:  dir,
		DB:   db,
	}
	b.peerAddresses = append(b.peerAddresses, addr)
	return db, nil
}

func (b *Builder) Close() error {
	c := errors.NewCatcher(errors.WithAggregation())
	for _, ni := range b.Nodes {
		c.Exec(ni.DB.Close)
	}
	c.Exec(b.Cleanup)
	return c.Error()
}

func (b *Builder) Cleanup() error {
	if !b.memBacked {
		return os.RemoveAll(b.Dir())
	}
	return nil
}
