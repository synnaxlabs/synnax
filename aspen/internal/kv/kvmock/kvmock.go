// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/freighter/fmock"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/kv/memkv"
)

type Builder struct {
	clustermock.Builder
	BaseCfg     kv.Config
	OpNet       *fmock.Network[kv.TxRequest, kv.TxRequest]
	FeedbackNet *fmock.Network[kv.FeedbackMessage, types.Nil]
	LeaseNet    *fmock.Network[kv.TxRequest, types.Nil]
	RecoveryNet *fmock.Network[kv.RecoveryRequest, kv.RecoveryResponse]
	KVs         map[node.Key]xkv.DB
}

func NewBuilder(baseKVCfg kv.Config, baseClusterCfg cluster.Config) *Builder {
	return &Builder{
		BaseCfg:     baseKVCfg,
		Builder:     *clustermock.NewBuilder(baseClusterCfg),
		OpNet:       fmock.NewNetwork[kv.TxRequest, kv.TxRequest](),
		FeedbackNet: fmock.NewNetwork[kv.FeedbackMessage, types.Nil](),
		LeaseNet:    fmock.NewNetwork[kv.TxRequest, types.Nil](),
		RecoveryNet: fmock.NewNetwork[kv.RecoveryRequest, kv.RecoveryResponse](),
		KVs:         make(map[node.Key]xkv.DB),
	}
}

func (b *Builder) New(ctx context.Context, kvCfg kv.Config, clusterCfg cluster.Config) (*kv.DB, error) {
	c, err := b.Builder.New(ctx, clusterCfg)
	if err != nil {
		return nil, err
	}
	kvCfg = b.BaseCfg.Override(kvCfg)
	if kvCfg.Engine == nil {
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
	return kve, nil
}

func (b *Builder) Close() error {
	for _, db := range b.KVs {
		if err := db.Close(); err != nil {
			return err
		}
	}

	for _, n := range b.ClusterAPIs {
		if err := n.Close(); err != nil {
			return err
		}
	}
	return nil
}
