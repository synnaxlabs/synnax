// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"bytes"
	"context"

	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/version"
	"go.uber.org/zap"
)

type RecoveryRequest struct {
	HighWater version.Counter `json:"high_water" msgpack:"high_water"`
}

type RecoveryResponse struct {
	Operations []Operation `json:"operations" msgpack:"operations"`
}

type (
	RecoveryTransportClient       = freighter.StreamClient[RecoveryRequest, RecoveryResponse]
	RecoveryTransportClientStream = freighter.ClientStream[RecoveryRequest, RecoveryResponse]
	RecoveryTransportServer       = freighter.StreamServer[RecoveryRequest, RecoveryResponse]
	RecoveryTransportServerStream = freighter.ServerStream[RecoveryRequest, RecoveryResponse]
)

type recoveryServer struct{ Config }

func newRecoveryServer(cfg Config) *recoveryServer {
	rs := &recoveryServer{Config: cfg}
	rs.RecoveryTransportServer.BindHandler(rs.recoverPeer)
	return rs
}

func (r *recoveryServer) recoverPeer(
	ctx context.Context,
	stream RecoveryTransportServerStream,
) error {
	req, err := stream.Receive()
	if err != nil {
		return err
	}
	iter, err := r.Engine.OpenIterator(kv.IterPrefix([]byte(digestPrefix)))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	var dig Digest
	for iter.First(); iter.Valid(); iter.Next() {
		encodedDig := iter.Value()
		if err = codec.Decode(ctx, encodedDig, &dig); err != nil {
			return err
		}
		if dig.Version.OlderThan(req.HighWater) {
			continue
		}

		op := Operation{}
		op.Key = dig.Key
		op.Version = dig.Version
		op.Leaseholder = dig.Leaseholder
		op.Variant = dig.Variant

		if op.Variant == change.Set {
			v, closer, err := r.Engine.Get(ctx, dig.Key)
			if err != nil {
				return err
			}
			op.Value = bytes.Clone(v)
			if err := closer.Close(); err != nil {
				return err
			}
		}

		if err = stream.Send(RecoveryResponse{Operations: []Operation{op}}); err != nil {
			return err
		}
	}
	return nil
}

func runRecovery(ctx context.Context, cfg Config) error {
	cfg.Instrumentation = cfg.Child("recovery")
	nodes := cfg.Cluster.Nodes()
	sCtx := signal.Wrap(ctx, signal.WithInstrumentation(cfg.Instrumentation))
	cfg.L.Info("recovering lost key-value operations", zap.Int("peer_node_count", len(nodes)-1))
	for _, n := range nodes {
		if n.Key == cfg.Cluster.HostKey() {
			continue
		}
		sCtx.Go(func(ctx context.Context) error {
			return runSingleNodeRecovery(ctx, cfg, n)
		}, signal.WithKeyf("node_%v", n.Key))
	}
	err := sCtx.Wait()
	if err != nil {
		cfg.L.Error("recovery failed", zap.Error(err))
	}
	return err
}

func loadHighWater(ctx context.Context, cfg Config) (highWater version.Counter, err error) {
	iter, err := cfg.Engine.OpenIterator(kv.IterPrefix([]byte(digestPrefix)))
	if err != nil {
		return
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()

	var dig Digest
	for iter.First(); iter.Valid(); iter.Next() {
		v := iter.Value()
		if err = codec.Decode(ctx, v, &dig); err != nil {
			return
		}
		if dig.Version.NewerThan(highWater) {
			highWater = dig.Version
		}
	}
	return
}

func runSingleNodeRecovery(
	ctx context.Context,
	cfg Config,
	node node.Node,
) error {
	hw, err := loadHighWater(ctx, cfg)
	if err != nil {
		return err
	}
	cfg.L.Info("starting recovery for node", zap.Stringer("nodeKey", node.Key), zap.Int64("highWater", int64(hw)))
	stream, err := cfg.RecoveryTransportClient.Stream(ctx, node.Address)
	if err != nil {
		return err
	}
	if err = stream.Send(RecoveryRequest{HighWater: hw}); err != nil {
		return err
	}
	return kv.WithTx(ctx, cfg.Engine, func(tx kv.Tx) error {
		count := 0
		for {
			resp, err := stream.Receive()
			if err != nil {
				if errors.Is(err, freighter.EOF) {
					break
				}
				return err
			}
			count += len(resp.Operations)
			for _, op := range resp.Operations {
				if err = op.apply(ctx, tx); err != nil {
					return err
				}
			}
		}
		cfg.L.Info("successfully recovered lost key-value operations", zap.Stringer("node", node.Key), zap.Int("operations", count))
		return nil
	})
}
