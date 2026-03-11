// Copyright 2026 Synnax Labs, Inc.
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
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// persistSplitter fans out persisted transactions to two outlets. Out[0] receives
// blocking sends for the gossip store. Out[1] receives non-blocking sends via a
// buffered relay channel for the observable, dropping values when the buffer is full.
type persistSplitter struct {
	confluence.UnarySink[TxRequest]
	confluence.AbstractMultiSource[TxRequest]
	alamos.Instrumentation
	bufferSize int
}

func newPersistSplitter(bufferSize int, ins alamos.Instrumentation) *persistSplitter {
	s := &persistSplitter{bufferSize: bufferSize, Instrumentation: ins}
	if s.L != nil {
		s.L = s.L.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSamplerWithOptions(core, time.Second, 1, 0)
		}))
	}
	return s
}

func (s *persistSplitter) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(confluence.InletsToClosables(s.Out)...)
	relay := make(chan TxRequest, s.bufferSize)

	ctx.Go(func(ctx context.Context) error {
		defer close(relay)
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case v, ok := <-s.In.Outlet():
				if !ok {
					return nil
				}
				if err := signal.SendUnderContext(ctx, s.Out[0].Inlet(), v); err != nil {
					return err
				}
				select {
				case relay <- v:
				default:
					s.L.Warn("observable relay buffer full, dropping transaction")
				}
			}
		}
	}, o.Signal...)

	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case v, ok := <-relay:
				if !ok {
					return nil
				}
				if err := signal.SendUnderContext(ctx, s.Out[1].Inlet(), v); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}
