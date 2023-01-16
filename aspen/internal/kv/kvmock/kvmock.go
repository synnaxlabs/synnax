// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kvmock

import (
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/signal"
	"go/types"
)

type Builder struct {
	clustermock.Builder
	BaseCfg     kv.Config
	OpNet       *fmock.Network[kv.BatchRequest, kv.BatchRequest]
	FeedbackNet *fmock.Network[kv.FeedbackMessage, types.Nil]
	LeaseNet    *fmock.Network[kv.BatchRequest, types.Nil]
	KVs         map[node.Key]kv.DB
}

func NewBuilder(baseKVCfg kv.Config, baseClusterCfg cluster.Config) *Builder {
	return &Builder{
		BaseCfg:     baseKVCfg,
		Builder:     *clustermock.NewBuilder(baseClusterCfg),
		OpNet:       fmock.NewNetwork[kv.BatchRequest, kv.BatchRequest](),
		FeedbackNet: fmock.NewNetwork[kv.FeedbackMessage, types.Nil](),
		LeaseNet:    fmock.NewNetwork[kv.BatchRequest, types.Nil](),
		KVs:         make(map[node.Key]kv.DB),
	}
}

func (b *Builder) New(ctx signal.Context, kvCfg kv.Config, clusterCfg cluster.Config) (kv.DB, error) {
	clust, err := b.Builder.New(ctx, clusterCfg)
	if err != nil {
		return nil, err
	}
	kvCfg = b.BaseCfg.Override(kvCfg)
	if kvCfg.Engine == nil {
		kvCfg.Engine = memkv.New()
	}
	kvCfg.Cluster = clust
	addr := clust.Host().Address
	kvCfg.BatchTransportClient = b.OpNet.UnaryClient()
	kvCfg.BatchTransportServer = b.OpNet.UnaryServer(addr)
	kvCfg.FeedbackTransportServer = b.FeedbackNet.UnaryServer(addr)
	kvCfg.FeedbackTransportClient = b.FeedbackNet.UnaryClient()
	kvCfg.LeaseTransportServer = b.LeaseNet.UnaryServer(addr)
	kvCfg.LeaseTransportClient = b.LeaseNet.UnaryClient()
	kve, err := kv.Open(ctx, kvCfg)
	if err != nil {
		return nil, err
	}
	b.KVs[clust.Host().Key] = kve
	return kve, nil
}
