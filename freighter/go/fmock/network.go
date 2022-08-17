package fmock

import (
	"fmt"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/address"
	"sync"
)

// Network is a mock network implementation that is ideal for in-memory testing
// scenarios. It serves as a factory for freighter.StreamTransport and freighter.Unary.
type Network[RQ, RS freighter.Payload] struct {
	mu sync.Mutex
	// Entries is a slice of entries in the network. Entries currently only supports
	// unary entries.
	Entries      []NetworkEntry[RQ, RS]
	UnaryRoutes  map[address.Address]*Unary[RQ, RS]
	StreamRoutes map[address.Address]*StreamTransport[RQ, RS]
}

// NetworkEntry is a single entry in the network's history. NetworkEntry
type NetworkEntry[RQ, RS freighter.Payload] struct {
	Host     address.Address
	Target   address.Address
	Request  RQ
	Response RS
	Error    error
}

// RouteUnary returns a new freighter.Unary hosted at the given address. This transport
// is not reachable by other hosts in the network until freighter.Unary.ServeHTTP is called.
func (n *Network[RQ, RS]) RouteUnary(host address.Address) *Unary[RQ, RS] {
	pHost := n.parseTarget(host)
	t := &Unary[RQ, RS]{Address: pHost, Network: n}
	n.UnaryRoutes[pHost] = t
	return t
}

const defaultStreamBuffer = 10

// RouteStream returns a new freighter.StreamTransport hosted at the given address.
// This transport is not reachable by other hosts in the network until
// freighter.StreamTransport.ServeHTTP is called.
func (n *Network[RQ, RS]) RouteStream(host address.Address, buffer int) *StreamTransport[RQ, RS] {
	addr := n.parseTarget(host)
	if buffer <= 0 {
		buffer = defaultStreamBuffer
	}
	t := &StreamTransport[RQ, RS]{Address: addr, Network: n, BufferSize: buffer}
	n.StreamRoutes[addr] = t
	return t
}

func (n *Network[RQ, RS]) parseTarget(target address.Address) address.Address {
	if target == "" {
		return address.Address(fmt.Sprintf("localhost:%v", len(n.UnaryRoutes)+len(n.StreamRoutes)))
	}
	return target
}

func (n *Network[RQ, RS]) appendEntry(host, target address.Address, req RQ, res RS, err error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Entries = append(n.Entries, NetworkEntry[RQ, RS]{
		Host:     host,
		Target:   target,
		Request:  req,
		Response: res,
		Error:    err,
	})
}

// NewNetwork returns a new network that can exchange the provided message types.
func NewNetwork[RQ, RS freighter.Payload]() *Network[RQ, RS] {
	return &Network[RQ, RS]{
		UnaryRoutes:  make(map[address.Address]*Unary[RQ, RS]),
		StreamRoutes: make(map[address.Address]*StreamTransport[RQ, RS]),
	}
}
