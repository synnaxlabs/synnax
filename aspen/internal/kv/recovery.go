// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/version"
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
	rs.RecoveryTransportServer.BindHandler(rs.recover)
	return rs
}

func (r *recoveryServer) recover(
	ctx context.Context,
	stream RecoveryTransportServerStream,
) error {
	req, err := stream.Receive()
	if err != nil {
		return err
	}
	iter, err := r.Engine.OpenIterator(kv.IterPrefix([]byte("--dig/")))
	defer func() {
		err = errors.CombineErrors(err, iter.Close())
	}()
	if err != nil {
		return err
	}
	var dig Digest
	for iter.First(); iter.Valid(); iter.Next() {
		v := iter.Value()
		if err = codec.Decode(ctx, v, &dig); err != nil {
			return err
		}
		if dig.Version.OlderThan(req.HighWater) {
			continue
		}
		v, closer, err := r.Engine.Get(ctx, dig.Key)
		if err != nil {
			return err
		}
		op := Operation{}
		op.Key = dig.Key
		op.Version = dig.Version
		op.Leaseholder = dig.Leaseholder
		op.Variant = dig.Variant
		op.Value = v
		if err := stream.Send(RecoveryResponse{Operations: []Operation{op}}); err != nil {
			return err
		}
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

func runRecovery(ctx context.Context, cfg Config) error {
	nodes := cfg.Cluster.Nodes()
	sCtx := signal.Wrap(
		ctx,
		signal.WithInstrumentation(cfg.Instrumentation.Child("recovery")),
	)
	for _, n := range nodes {
		if n.Key == cfg.Cluster.HostKey() {
			continue
		}
		sCtx.Go(func(ctx context.Context) error {
			return runSingleNodeRecovery(ctx, cfg, n)
		}, signal.WithKeyf("node-%v", n.Key))
	}
	return sCtx.Wait()
}

func runSingleNodeRecovery(
	ctx context.Context,
	cfg Config,
	node node.Node,
) error {
	iter, err := cfg.Engine.OpenIterator(kv.IterPrefix([]byte("--dig/")))
	defer func() {
		err = errors.CombineErrors(err, iter.Close())
	}()
	if err != nil {
		return err
	}
	var highWater version.Counter
	var dig Digest
	for iter.First(); iter.Valid(); iter.Next() {
		v := iter.Value()
		if err = codec.Decode(ctx, v, &dig); err != nil {
			return err
		}
		if dig.Leaseholder != node.Key {
			continue
		}
		if highWater.NewerThan(dig.Version) {
			highWater = dig.Version
		}
	}
	req := RecoveryRequest{HighWater: highWater}
	stream, err := cfg.RecoveryTransportClient.Stream(ctx, node.Address)
	if err != nil {
		return err
	}
	if err := stream.Send(req); err != nil {
		return err
	}
	for {
		resp, err := stream.Receive()
		if err != nil {
			if errors.Is(err, freighter.EOF) {
				return nil
			}
			return err
		}
		for _, op := range resp.Operations {
			res := make(map[string]interface{})
			if err = codec.Decode(ctx, op.Value, &res); err != nil {
				fmt.Print(err.Error())
			}
			if err := op.apply(ctx, cfg.Engine); err != nil {
				return err
			}
		}
	}
}
