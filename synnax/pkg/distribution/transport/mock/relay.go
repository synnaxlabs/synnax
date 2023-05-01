// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/relay"
	"github.com/synnaxlabs/x/address"
)

type RelayNetwork struct {
	internal *fmock.Network[relay.ReadRequest, framer.Frame]
}

func NewRelayNetwork() *RelayNetwork {
	return &RelayNetwork{
		internal: fmock.NewNetwork[relay.ReadRequest, framer.Frame](),
	}
}

func (r *RelayNetwork) New(addr address.Address, buffers ...int) relay.Transport {
	return &RelayTransport{
		client: r.internal.StreamClient(buffers...),
		server: r.internal.StreamServer(addr, buffers...),
	}
}

type RelayTransport struct {
	client relay.TransportClient
	server relay.TransportServer
}

var _ relay.Transport = (*RelayTransport)(nil)

func (r RelayTransport) Client() relay.TransportClient { return r.client }

func (r RelayTransport) Server() relay.TransportServer { return r.server }
