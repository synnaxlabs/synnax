// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"fmt"
	"sync"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/address"
)

// Network is a mock network implementation that is ideal for in-memory testing
// scenarios. It serves as a factory for freighter.Stream and freighter.Unary.
type Network[RQ, RS freighter.Payload] struct {
	mu struct {
		// entries records every unary exchange the network has carried, in order.
		entries []NetworkEntry[RQ, RS]
		// unaryRoutes holds the unary server hosted at each address.
		unaryRoutes map[address.Address]*UnaryServer[RQ, RS]
		// streamRoutes holds the stream server hosted at each address.
		streamRoutes map[address.Address]*StreamServer[RQ, RS]
		sync.RWMutex
	}
}

// Entries returns a copy of every unary exchange the network has carried, oldest first.
// Streams are not recorded.
func (n *Network[RQ, RS]) Entries() []NetworkEntry[RQ, RS] {
	n.mu.RLock()
	defer n.mu.RUnlock()
	cp := make([]NetworkEntry[RQ, RS], len(n.mu.entries))
	copy(cp, n.mu.entries)
	return cp
}

// EntryCount returns how many unary exchanges the network has carried. It avoids the
// copy Entries makes when only the count is needed.
func (n *Network[RQ, RS]) EntryCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.mu.entries)
}

// NetworkEntry is a single unary exchange in the network's history.
type NetworkEntry[RQ, RS freighter.Payload] struct {
	// Request is what the client sent.
	Request RQ
	// Response is what the handler returned. Zero valued when Error is not nil.
	Response RS
	// Error is what the handler returned, nil on success.
	Error error
	// Target is the address the client dialed.
	Target address.Address
}

// UnaryServer returns a new freighter.UnaryServer hosted at the given address. An empty
// host takes the next free localhost port. The server is not reachable until
// BindHandler is called.
func (n *Network[RQ, RS]) UnaryServer(host address.Address) *UnaryServer[RQ, RS] {
	n.mu.Lock()
	defer n.mu.Unlock()
	pHost := n.parseTarget(host)
	s := &UnaryServer[RQ, RS]{network: n, address: pHost}
	n.mu.unaryRoutes[pHost] = s
	return s
}

// UnaryClient returns a new freighter.UnaryClient that dials servers on this network.
func (n *Network[RQ, RS]) UnaryClient() *UnaryClient[RQ, RS] {
	n.mu.Lock()
	defer n.mu.Unlock()
	return &UnaryClient[RQ, RS]{network: n}
}

func (n *Network[RQ, RS]) resolveUnaryTarget(
	target address.Address,
) (*UnaryServer[RQ, RS], bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	t, ok := n.mu.unaryRoutes[target]
	return t, ok
}

// StreamServer returns a new freighter.StreamServer hosted at the given address. An
// empty host takes the next free localhost port. Buffer sets the capacity of the
// response channel given to each stream. The server is not reachable until BindHandler
// is called.
func (n *Network[RQ, RS]) StreamServer(
	host address.Address,
	buffer ...int,
) *StreamServer[RQ, RS] {
	n.mu.Lock()
	defer n.mu.Unlock()
	addr := n.parseTarget(host)
	b, _ := parseBuffers(buffer)
	s := &StreamServer[RQ, RS]{bufferSize: b, address: addr}
	n.mu.streamRoutes[addr] = s
	return s
}

// StreamClient returns a new freighter.StreamClient that dials servers on this network.
// Buffers sets the capacity of the request channel given to each stream.
func (n *Network[RQ, RS]) StreamClient(buffers ...int) *StreamClient[RQ, RS] {
	b, _ := parseBuffers(buffers)
	return &StreamClient[RQ, RS]{network: n, bufferSize: b}
}

func (n *Network[RQ, RS]) resolveStreamTarget(
	target address.Address,
) (*StreamServer[RQ, RS], bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	t, ok := n.mu.streamRoutes[target]
	return t, ok
}

func (n *Network[RQ, RS]) parseTarget(target address.Address) address.Address {
	if target == "" {
		return address.Address(
			fmt.Sprintf("localhost:%v", len(n.mu.unaryRoutes)+len(n.mu.streamRoutes)),
		)
	}
	return target
}

func (n *Network[RQ, RS]) appendEntry(
	target address.Address,
	req RQ,
	res RS,
	err error,
) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.mu.entries = append(n.mu.entries, NetworkEntry[RQ, RS]{
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
