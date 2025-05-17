// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence

import (
	"context"
	"reflect"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
)

// UnarySink is a basic implementation of Sink that can receive values from a single Inlet.
type UnarySink[V Value] struct {
	// Sink is called whenever a value is received from the Outlet.
	Sink func(ctx context.Context, value V) error
	AbstractUnarySink[V]
}

// Flow implements the Flow interface.
func (us *UnarySink[V]) Flow(ctx signal.Context, opts ...Option) {
	us.GoRange(ctx, us.Sink, NewOptions(opts).Signal...)
}

func (us *UnarySink[V]) GoRange(ctx signal.Context, f func(context.Context, V) error, opts ...signal.RoutineOption) {
	signal.GoRange(ctx, us.In.Outlet(), f, opts...)
}

type AbstractUnarySink[V Value] struct{ In Outlet[V] }

// InFrom implements the Sink interface.
func (as *AbstractUnarySink[V]) InFrom(outlets ...Outlet[V]) {
	if len(outlets) != 1 {
		panic("[confluence.UnarySink] - must have exactly one outlet")
	}
	as.In = outlets[0]
}

type MultiSink[I Value] struct {
	In    []Outlet[I]
	cases []reflect.SelectCase
	Sink  func(ctx context.Context, origin address.Address, value I) error
}

func (ms *MultiSink[I]) InFrom(outlets ...Outlet[I]) {
	for _, outlet := range outlets {
		if outlet.OutletAddress() == "" {
			panic("[confluence.AbstractAddressableSink[I]] - must have a outlet address")
		}
		ms.In = append(ms.In, outlet)
		ms.cases = append(ms.cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(outlet.Outlet()),
		})
	}
}

func (ms *MultiSink[I]) Receive(sCtx signal.Context) (origin address.Address, value I, err error, ok bool) {
	chosen, recv, ok := reflect.Select(ms.cases)
	if !ok {
		return "", value, nil, ok
	}
	if chosen == len(ms.cases) {
		return "", value, sCtx.Err(), true
	}
	value = recv.Interface().(I)
	return ms.In[chosen].OutletAddress(), value, nil, ok
}

func (ms *MultiSink[I]) Flow(sCtx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	sCtx.Go(func(ctx context.Context) error {
		if len(ms.cases) == len(ms.In) {
			ms.cases = append(ms.cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(sCtx.Done()),
			})
		}
		for {
			origin, value, err, ok := ms.Receive(sCtx)
			if !ok || err != nil {
				return err
			}
			if err = ms.Sink(ctx, origin, value); err != nil {
				return err
			}
		}
	}, o.Signal...)
}
