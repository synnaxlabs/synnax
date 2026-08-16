// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter

import (
	"go/types"

	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/x/address"
)

// Network backs the in-memory deleter transport for a cluster of nodes.
type Network struct {
	net *mock.Network[deleter.Request, types.Nil]
}

// NewNetwork constructs a Network with a freshly initialized deleter network.
func NewNetwork() *Network {
	return &Network{net: mock.NewNetwork[deleter.Request, types.Nil]()}
}

// New provisions an in-memory deleter.Transport whose server is hosted at addr.
func (n *Network) New(addr address.Address) deleter.Transport {
	return transport{client: n.net.UnaryClient(), server: n.net.UnaryServer(addr)}
}

type transport struct {
	client deleter.Client
	server deleter.Server
}

func (t transport) Client() deleter.Client { return t.client }

func (t transport) Server() deleter.Server { return t.server }
