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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
)

// Transport bundles the in-memory channel and framer transports for a single node. It
// implements the distribution.Transport interface.
type Transport struct {
	// ChannelTransport is the in-memory channel transport for the node.
	ChannelTransport channel.Transport
	// FramerTransport is the in-memory framer transport for the node.
	FramerTransport framer.Transport
}

// Channel implements distribution.Transport.
func (t Transport) Channel() channel.Transport { return t.ChannelTransport }

// Framer implements distribution.Transport.
func (t Transport) Framer() framer.Transport { return t.FramerTransport }
