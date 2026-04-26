// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kvmock

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/x/errors"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

type Builder struct {
	clustermock.Builder
	OpNet       *mock.Network[kv.TxRequest, kv.TxRequest]
	FeedbackNet *mock.Network[kv.FeedbackMessage, types.Nil]
	LeaseNet    *mock.Network[kv.TxRequest, types.Nil]
	RecoveryNet *mock.Network[kv.RecoveryRequest, kv.RecoveryResponse]
	KVs         map[node.Key]xkv.DB
	// engines holds memkv.New() backing stores keyed by node so Close can
	// shut them down. The kv.DB does not own its Engine, so the Builder
	// has to keep references to close them.
	engines map[node.Key]xkv.DB
	BaseCfg kv.Config
}

func NewBuilder(baseKVCfg kv.Config, baseClusterCfg cluster.Config) *Builder {
	return &Builder{
		BaseCfg:     baseKVCfg,
		Builder:     *clustermock.NewBuilder(baseClusterCfg),
		OpNet:       mock.NewNetwork[kv.TxRequest, kv.TxRequest](),
		FeedbackNet: mock.NewNetwork[kv.FeedbackMessage, types.Nil](),
		LeaseNet:    mock.NewNetwork[kv.TxRequest, types.Nil](),
		RecoveryNet: mock.NewNetwork[kv.RecoveryRequest, kv.RecoveryResponse](),
		KVs:         make(map[node.Key]xkv.DB),
		engines:     make(map[node.Key]xkv.DB),
	}
}

func (b *Builder) New(ctx context.Context, kvCfg kv.Config, clusterCfg cluster.Config) (*kv.DB, error) {
	c, err := b.Builder.New(ctx, clusterCfg)
	if err != nil {
		return nil, err
	}
	kvCfg = b.BaseCfg.Override(kvCfg)
	ownsEngine := kvCfg.Engine == nil
	if ownsEngine {
		kvCfg.Engine = memkv.New()
	}
	kvCfg.Cluster = c
	addr := c.Host().Address
	kvCfg.BatchTransportClient = b.OpNet.UnaryClient()
	kvCfg.BatchTransportServer = b.OpNet.UnaryServer(addr)
	kvCfg.FeedbackTransportServer = b.FeedbackNet.UnaryServer(addr)
	kvCfg.FeedbackTransportClient = b.FeedbackNet.UnaryClient()
	kvCfg.LeaseTransportServer = b.LeaseNet.UnaryServer(addr)
	kvCfg.LeaseTransportClient = b.LeaseNet.UnaryClient()
	kvCfg.RecoveryTransportServer = b.RecoveryNet.StreamServer(addr)
	kvCfg.RecoveryTransportClient = b.RecoveryNet.StreamClient()
	kve, err := kv.Open(ctx, kvCfg)
	if err != nil {
		return nil, err
	}
	b.KVs[c.Host().Key] = kve
	if ownsEngine {
		b.engines[c.Host().Key] = kvCfg.Engine
	}
	return kve, nil
}

func (b *Builder) Close() error {
	var err error
	// Close cluster gossip first so it can no longer issue background reads
	// or writes against the kv layer.
	err = errors.Combine(err, b.Builder.Close())
	// Then close the kv DBs themselves so any in-flight operations complete.
	for _, db := range b.KVs {
		err = errors.Combine(err, db.Close())
	}
	// Finally close any engines we own; everything above must be quiet by now
	// or pebble will complain about leaked iterators.
	for _, e := range b.engines {
		err = errors.Combine(err, e.Close())
	}
	b.KVs = make(map[node.Key]xkv.DB)
	b.engines = make(map[node.Key]xkv.DB)
	return err
}
