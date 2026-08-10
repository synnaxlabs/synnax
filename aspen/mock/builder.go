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
	"path/filepath"
	"strconv"

	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/x/address"
)

type Builder struct {
	TmpDirs        map[aspen.NodeKey]string
	Nodes          map[aspen.NodeKey]NodeInfo
	DataDir        string
	tmpDir         string
	DefaultOptions []aspen.Option
	peerAddresses  []address.Address
	memBacked      bool
}

type NodeInfo struct {
	DB   *aspen.DB
	Addr address.Address
	Dir  string
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

// New opens a node on a port the operating system chooses, peered with every node the
// Builder opened before it. The first node bootstraps the cluster.
func (b *Builder) New(ctx context.Context, opts ...aspen.Option) (*aspen.DB, error) {
	dir := filepath.Join(b.Dir(), strconv.Itoa(len(b.peerAddresses)))
	if len(b.Nodes) == 0 {
		opts = append(opts, aspen.Bootstrap())
	}
	db, err := aspen.Open(
		ctx,
		dir,
		"localhost:0",
		b.peerAddresses,
		append(b.DefaultOptions, opts...)...)
	if err != nil {
		return nil, err
	}
	addr := db.Cluster.Host().Address
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
