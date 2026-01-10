// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gossip

import (
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/freighter"
)

type (
	TransportServer = freighter.UnaryServer[Message, Message]
	TransportClient = freighter.UnaryClient[Message, Message]
)

type Message struct {
	Digests node.Digests
	Nodes   node.Group
}

func (msg Message) variant() messageVariant {
	if len(msg.Nodes) == 0 && len(msg.Digests) != 0 {
		return messageVariantSync
	}
	if len(msg.Digests) == 0 && len(msg.Nodes) != 0 {
		return messageVariantAck2
	}
	return messageVariantInvalid
}

type messageVariant byte

const (
	messageVariantSync = iota
	messageVariantAck2
	messageVariantInvalid
)
