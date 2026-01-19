// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmock

import (
	"fmt"
	"sync"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

// Network is a mock network implementation that is ideal for in-memory testing
// scenarios. It serves as a factory for freighter.Stream and freighter.Unary.
type Network[RQ, RS freighter.Payload] struct {
	// Entries is a slice of entries in the network. Entries currently only supports
	// unary entries.
	Entries []NetworkEntry[RQ, RS]
	mu      struct {
		unaryRoutes  map[address.Address]*UnaryServer[RQ, RS]
		streamRoutes map[address.Address]*StreamServer[RQ, RS]
		sync.RWMutex
	}
}

// NetworkEntry is a single entry in the network's history. NetworkEntry
type NetworkEntry[RQ, RS freighter.Payload] struct {
	Request  RQ
	Response RS
	Error    error
	Host     address.Address
	Target   address.Address
}

// UnaryServer returns a new freighter.Unary hosted at the given address. This transport
// is not reachable by other hosts in the network until freighter.UnaryServer.ServeHTTP is called.
func (n *Network[RQ, RS]) UnaryServer(host address.Address) *UnaryServer[RQ, RS] {
	n.mu.Lock()
	defer n.mu.Unlock()
	pHost := n.parseTarget(host)
	s := &UnaryServer[RQ, RS]{Network: n, Address: pHost}
	n.mu.unaryRoutes[pHost] = s
	return s
}

func (n *Network[RQ, RS]) UnaryClient() *UnaryClient[RQ, RS] {
	n.mu.Lock()
	defer n.mu.Unlock()
	return &UnaryClient[RQ, RS]{Network: n}
}

func (n *Network[RQ, RS]) resolveUnaryTarget(target address.Address) (*UnaryServer[RQ, RS], bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	t, ok := n.mu.unaryRoutes[target]
	return t, ok
}

// StreamServer returns a new freighter.Stream hosted at the given address.
// This transport is not reachable by other hosts in the network until
// freighter.Stream.ServeHTTP is called.
func (n *Network[RQ, RS]) StreamServer(host address.Address, buffer ...int) *StreamServer[RQ, RS] {
	n.mu.Lock()
	defer n.mu.Unlock()
	addr := n.parseTarget(host)
	b, _ := parseBuffers(buffer)
	s := &StreamServer[RQ, RS]{Reporter: reporter, BufferSize: b, Address: addr}
	n.mu.streamRoutes[addr] = s
	return s
}

func (n *Network[RQ, RS]) StreamClient(buffers ...int) *StreamClient[RQ, RS] {
	b, _ := parseBuffers(buffers)
	return &StreamClient[RQ, RS]{
		Network:    n,
		BufferSize: b,
	}
}

func (n *Network[RQ, RS]) resolveStreamTarget(target address.Address) (*StreamServer[RQ, RS], bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	t, ok := n.mu.streamRoutes[target]
	return t, ok
}

func (n *Network[RQ, RS]) parseTarget(target address.Address) address.Address {
	if target == "" {
		return address.Address(fmt.Sprintf("localhost:%v", len(n.mu.unaryRoutes)+len(n.mu.streamRoutes)))
	}
	return target
}

func (n *Network[RQ, RS]) appendEntry(target address.Address, req RQ, res RS, err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Entries = append(n.Entries, NetworkEntry[RQ, RS]{
		Target:   target,
		Request:  req,
		Response: res,
		Error:    err,
	})
}

// NewNetwork returns a new network that can exchange the provided message types.
func NewNetwork[RQ, RS freighter.Payload]() *Network[RQ, RS] {
	n := &Network[RQ, RS]{}
	n.mu.unaryRoutes = make(map[address.Address]*UnaryServer[RQ, RS])
	n.mu.streamRoutes = make(map[address.Address]*StreamServer[RQ, RS])
	return n
}
