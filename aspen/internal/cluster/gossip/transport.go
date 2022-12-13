package gossip

import (
	"github.com/synnaxlabs/aspen/internal/node"
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
