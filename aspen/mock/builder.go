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
	"path/filepath"
	"strconv"

	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/x/address"
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

func (b *Builder) New(ctx context.Context, opts ...aspen.Option) (*aspen.DB, error) {
	dir := filepath.Join(b.Dir(), strconv.Itoa(len(b.peerAddresses)))
	if len(b.Nodes) == 0 {
		opts = append(opts, aspen.Bootstrap())
	}
	addr := b.addressFactory().Next()
	db, err := aspen.Open(ctx, dir, addr, b.peerAddresses, append(b.DefaultOptions, opts...)...)
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

func (b *Builder) Cleanup() error {
	if !b.memBacked {
		return os.RemoveAll(b.Dir())
	}
	return nil
}
