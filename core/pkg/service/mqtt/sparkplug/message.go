// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sparkplug

import (
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/x/errors"
	"google.golang.org/protobuf/proto"
)

// Message is one decoded message of an edge node or a device.
type Message struct {
	Topic   Topic
	payload *pb.Payload
}

// Decode parses the payload of a message on topic. It needs no session state, so a
// caller can decode outside the lock that guards a Host or a Sequence.
func Decode(topic Topic, payload []byte) (Message, error) {
	p := new(pb.Payload)
	if err := proto.Unmarshal(payload, p); err != nil {
		return Message{}, errors.Wrapf(err, "invalid payload on %s", topic)
	}
	return Message{Topic: topic, payload: p}, nil
}

// Publication is one message to publish, with the quality of service and the retain
// flag that the Sparkplug B specification gives its type.
type Publication struct {
	Topic    string
	Payload  []byte
	QoS      byte
	Retained bool
}
