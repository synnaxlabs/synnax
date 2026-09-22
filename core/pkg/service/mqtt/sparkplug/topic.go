// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package sparkplug implements the Sparkplug B payloads, topics, and host session
// state. It has no MQTT client: the caller moves the messages.
package sparkplug

import (
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Namespace is the first level of every Sparkplug B topic.
const Namespace = "spBv1.0"

// MessageType is the message type level of a Sparkplug B topic.
type MessageType string

// The message types of the Sparkplug B specification.
const (
	NBirth MessageType = "NBIRTH"
	NDeath MessageType = "NDEATH"
	DBirth MessageType = "DBIRTH"
	DDeath MessageType = "DDEATH"
	NData  MessageType = "NDATA"
	DData  MessageType = "DDATA"
	NCmd   MessageType = "NCMD"
	DCmd   MessageType = "DCMD"
	State  MessageType = "STATE"
)

// Session reports whether a message of type t is part of the session of an edge node:
// a birth, a death, or data. Only such messages change the state of a host.
func (t MessageType) Session() bool {
	switch t {
	case NBirth, NDeath, DBirth, DDeath, NData, DData:
		return true
	}
	return false
}

// hasDevice reports whether a topic of type t ends with a device ID.
func (t MessageType) hasDevice() (device, ok bool) {
	switch t {
	case DBirth, DDeath, DData, DCmd:
		return true, true
	case NBirth, NDeath, NData, NCmd:
		return false, true
	}
	return false, false
}

// NodeID identifies one edge node.
type NodeID struct {
	Group    string
	EdgeNode string
}

// String returns the ID as group/edge_node.
func (n NodeID) String() string { return n.Group + "/" + n.EdgeNode }

// Validate checks that both IDs can be a level of a topic.
func (n NodeID) Validate() error {
	if err := ValidateID("group", n.Group); err != nil {
		return err
	}
	return ValidateID("edge_node", n.EdgeNode)
}

// Filters returns the topic filters that match every message of the edge node and of
// its devices. They hold no multi-level wildcard: some brokers do not match a/# against
// the parent level a.
func (n NodeID) Filters() []string {
	node := Namespace + "/" + n.Group + "/+/" + n.EdgeNode
	return []string{node, node + "/+"}
}

// ValidateID checks that id can be one level of a Sparkplug B topic. field names the
// ID in the error.
func ValidateID(field, id string) error {
	if id == "" {
		return errors.Wrapf(validate.ErrValidation, "%s: required", field)
	}
	if strings.ContainsAny(id, "/+#") {
		return errors.Wrapf(
			validate.ErrValidation, "%s: %s must not hold /, +, or #", field, id,
		)
	}
	return nil
}

// Topic is a parsed Sparkplug B topic.
type Topic struct {
	Type MessageType
	Node NodeID
	// Device is the device ID. Empty for a message of the edge node.
	Device string
	// HostID is the host application ID of a STATE topic.
	HostID string
}

// ErrTopic is returned when a topic is not a Sparkplug B topic.
var ErrTopic = errors.New("not a Sparkplug B topic")

// ParseTopic parses a topic. It returns an error that wraps ErrTopic for a topic
// outside the namespace or with the wrong shape for its message type.
func ParseTopic(topic string) (Topic, error) {
	levels := strings.Split(topic, "/")
	if len(levels) < 3 || levels[0] != Namespace {
		return Topic{}, errors.Wrap(ErrTopic, topic)
	}
	if levels[1] == string(State) {
		if len(levels) != 3 {
			return Topic{}, errors.Wrap(ErrTopic, topic)
		}
		return Topic{Type: State, HostID: levels[2]}, nil
	}
	if len(levels) < 4 {
		return Topic{}, errors.Wrap(ErrTopic, topic)
	}
	t := Topic{
		Type: MessageType(levels[2]),
		Node: NodeID{Group: levels[1], EdgeNode: levels[3]},
	}
	device, ok := t.Type.hasDevice()
	if !ok || (device && len(levels) != 5) || (!device && len(levels) != 4) {
		return Topic{}, errors.Wrap(ErrTopic, topic)
	}
	if device {
		t.Device = levels[4]
	}
	return t, nil
}

// String returns the topic text.
func (t Topic) String() string {
	if t.Type == State {
		return Namespace + "/" + string(State) + "/" + t.HostID
	}
	s := Namespace + "/" + t.Node.Group + "/" + string(t.Type) + "/" + t.Node.EdgeNode
	if t.Device != "" {
		s += "/" + t.Device
	}
	return s
}

// CommandTopic returns the topic of a command for a tag of node, or of device when it
// is not empty.
func CommandTopic(node NodeID, device string) Topic {
	if device == "" {
		return Topic{Type: NCmd, Node: node}
	}
	return Topic{Type: DCmd, Node: node, Device: device}
}
