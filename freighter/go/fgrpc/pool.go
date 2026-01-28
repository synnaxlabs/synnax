// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fgrpc

import (
	"path"

	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// ClientConn is a wrapper around grpc.ClientConn that implements the
// pool.Adapter interface.
type ClientConn struct {
	*grpc.ClientConn
	demand *pool.Demand
}

// Acquire implements pool.Adapter.
func (c *ClientConn) Acquire() error {
	c.demand.Increase(1)
	return nil
}

// Release implements pool.Adapter.
func (c *ClientConn) Release() {
	c.demand.Decrease(1)
}

// Close implements pool.Adapter.
func (c *ClientConn) Close() error { return c.ClientConn.Close() }

// Healthy implements pool.Adapter
func (c *ClientConn) Healthy() bool {
	state := c.GetState()
	return state != connectivity.TransientFailure && state != connectivity.Shutdown
}

type Pool struct {
	pool.Pool[address.Address, *ClientConn]
}

func NewPool(targetPrefix address.Address, dialOpts ...grpc.DialOption) *Pool {
	return &Pool{Pool: pool.New[address.Address, *ClientConn](&factory{dialOpts: dialOpts, targetPrefix: targetPrefix})}
}

// factory implements the pool.Factory interface.
type factory struct {
	targetPrefix address.Address
	dialOpts     []grpc.DialOption
}

func (f *factory) New(addr address.Address) (*ClientConn, error) {
	c, err := grpc.NewClient(path.Join(f.targetPrefix.String(), addr.String()), f.dialOpts...)
	d := pool.Demand(1)
	return &ClientConn{ClientConn: c, demand: &d}, err
}
