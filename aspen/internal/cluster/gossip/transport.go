package gossip

import (
	"github.com/arya-analytics/freighter"
	"github.com/synnaxlabs/aspen/internal/node"
)

type Transport = freighter.Unary[Message, Message]

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
