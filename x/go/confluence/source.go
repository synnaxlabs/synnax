// Copyright 2023 Synnax Labs, Inc.
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
	"fmt"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/timeout"
	"time"
)

// AbstractMultiSource is a basic implementation of a Source that can send values to
// multiple Outlet(sink). It implements an empty Flow method, as sources are typically
// driven by external events. The user can define a custom Flow method if they wish to
// drive the source themselves.
type AbstractMultiSource[V Value] struct {
	Out []Inlet[V]
}

// OutTo implements the Source interface.
func (ams *AbstractMultiSource[V]) OutTo(inlets ...Inlet[V]) { ams.Out = append(ams.Out, inlets...) }

// SendToEach sends the provided value to each Inlet in the Source.
func (ams *AbstractMultiSource[V]) SendToEach(ctx context.Context, v V) error {
	for _, inlet := range ams.Out {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case inlet.Inlet() <- v:
		}
	}
	return nil
}

func (ams *AbstractMultiSource[V]) SendToEachWithTimeout(
	ctx context.Context,
	v V,
	t time.Duration,
	timer *time.Timer,
) error {
	timerExpired := false
	for _, inlet := range ams.Out {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(t)
			timerExpired = true
		case inlet.Inlet() <- v:
		}
	}
	if timerExpired {
		return timeout.Timeout
	}
	return nil
}

// CloseInlets implements the InletCloser interface.
func (ams *AbstractMultiSource[V]) CloseInlets() {
	for _, inlet := range ams.Out {
		inlet.Close()
	}
}

// AbstractUnarySource is a basic implementation of a Source that sends values to a
// single Outlet. The user can define a custom Flow method if they wish to
// drive the source themselves.
type AbstractUnarySource[O Value] struct {
	Out Inlet[O]
}

// OutTo implements the Source interface.
func (aus *AbstractUnarySource[O]) OutTo(inlets ...Inlet[O]) {
	if len(inlets) != 1 {
		panic("[confluence.AbstractUnarySource] -  must have exactly one outlet")
	}
	aus.Out = inlets[0]
}

// CloseInlets implements the InletCloser interface.
func (aus *AbstractUnarySource[O]) CloseInlets() { aus.Out.Close() }

// AbstractAddressableSource is an implementation of a Source that stores its Inlet(sink) in an
// addressable map. This is ideal for use cases where the address of an Inlet is
// relevant to the routing of the value (such as a Switch).
type AbstractAddressableSource[O Value] struct {
	PanicOnDuplicateAddress bool
	// Out is an address map of all Inlet(sink) reachable by the Source.
	Out map[address.Address]Inlet[O]
}

// OutTo implements the Source interface. Inlets provided must have a valid InletAddress.
// If two inlets are provided with the same address, the last Inlet will override the
// previous one.
func (aas *AbstractAddressableSource[O]) OutTo(inlets ...Inlet[O]) {
	if aas.Out == nil {
		aas.Out = make(map[address.Address]Inlet[O])
	}
	for _, inlet := range inlets {
		if inlet.InletAddress() == "" {
			panic("[confluence.AbstractAddressableSource] - inlet must have a valid address")
		}
		if _, ok := aas.Out[inlet.InletAddress()]; ok && aas.PanicOnDuplicateAddress {
			panic(fmt.Sprintf("[confluence.AbstractAddressableSource] - duplicate address %sink", inlet.InletAddress()))
		}
		aas.Out[inlet.InletAddress()] = inlet
	}
}

// Send sends a value to the target address.
func (aas *AbstractAddressableSource[O]) Send(ctx context.Context, target address.Address, v O) error {
	inlet, ok := aas.Out[target]
	if !ok {
		return address.TargetNotFound(target)
	}
	return signal.SendUnderContext(ctx, inlet.Inlet(), v)
}

func (aas *AbstractAddressableSource[O]) SendToEach(ctx context.Context, v O) error {
	for _, inlet := range aas.Out {
		if err := signal.SendUnderContext(ctx, inlet.Inlet(), v); err != nil {
			return err
		}
	}
	return nil

}

// CloseInlets closes all Inlet(sink) provided to AbstractAddressableSource.OutTo.
func (aas *AbstractAddressableSource[O]) CloseInlets() {
	for _, inlet := range aas.Out {
		inlet.Close()
	}
}
