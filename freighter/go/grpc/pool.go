// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"path"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// ClientConn is a wrapper around [grpc.ClientConn] that implements the [pool.Adapter]
// interface.
type ClientConn struct {
	// ClientConn is the underlying gRPC client connection.
	*grpc.ClientConn
	demand *pool.Demand
}

// Acquire implements [pool.Adapter].
func (c *ClientConn) Acquire() error { c.demand.Increase(1); return nil }

// Release implements [pool.Adapter].
func (c *ClientConn) Release() { c.demand.Decrease(1) }

// Close implements [pool.Adapter].
func (c *ClientConn) Close() error { return c.ClientConn.Close() }

// Healthy implements [pool.Adapter].
func (c *ClientConn) Healthy() bool {
	state := c.GetState()
	return state != connectivity.TransientFailure && state != connectivity.Shutdown
}

// Pool is a pool of reusable gRPC client connections keyed by target address. Open one
// with [OpenPool] and acquire connections through the embedded [pool.Pool] interface.
type Pool struct {
	pool.Pool[address.Address, *ClientConn]
}

// OpenPool returns a [Pool] that dials connections with the given dial options.
// targetPrefix is prepended to every acquired address, allowing callers to scope all
// connections to a common host or namespace.
func OpenPool(targetPrefix address.Address, dialOpts ...grpc.DialOption) *Pool {
	return &Pool{
		Pool: pool.Open(&factory{dialOpts: dialOpts, targetPrefix: targetPrefix}),
	}
}

type factory struct {
	targetPrefix address.Address
	dialOpts     []grpc.DialOption
}

func (f *factory) Open(addr address.Address) (*ClientConn, error) {
	c, err := grpc.NewClient(
		path.Join(f.targetPrefix.String(), addr.String()),
		f.dialOpts...,
	)
	if err != nil {
		return nil, err
	}
	d := pool.Demand(1)
	return &ClientConn{ClientConn: c, demand: &d}, nil
}
