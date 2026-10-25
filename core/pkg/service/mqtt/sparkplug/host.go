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
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
)

// DefaultRebirthInterval is the shortest time between two rebirth requests to one edge
// node.
const DefaultRebirthInterval = 5 * time.Second

type nodeState struct {
	// tables holds the tags of the edge node under the empty device ID, and the tags
	// of each device that is online under its ID.
	tables      map[string]tagTable
	lastRebirth time.Time
	bdSeq       uint64
	bdSeqKnown  bool
	// nextSeq is the sequence number that the next message must carry.
	nextSeq uint8
	// born is true from a birth message until a death message or a sequence gap.
	born bool
	// rebirth is true from a need for a new birth until a birth arrives or
	// TakeRebirths returns the request.
	rebirth bool
}

// Event is the result of one message.
type Event struct {
	// Type is the type of the message. It is empty for a message that the host
	// dropped: a message out of sequence, data before a birth, or a stale death.
	Type MessageType
	Node NodeID
	// Device is the device ID. Empty for a message of the edge node.
	Device string
	// Metrics holds the values of a birth or a data message, with every alias
	// resolved to its name.
	Metrics []Metric
	// Rebirth is true when the host must ask the edge node for a new birth.
	Rebirth bool
	// BirthLost is true when the message ended the birth of the edge node. The host
	// drops the data of the node until its next birth.
	BirthLost bool
}

// Host is the session state of a Sparkplug B host application: for each edge node,
// the tags of its last birth, its aliases, and its sequence number. It is not safe for
// concurrent use.
type Host struct {
	nodes           map[NodeID]*nodeState
	rebirthInterval time.Duration
}

// NewHost returns a host that knows no edge node. rebirthInterval is the shortest time
// between two rebirth requests to one edge node.
func NewHost(rebirthInterval time.Duration) *Host {
	return &Host{
		nodes:           make(map[NodeID]*nodeState),
		rebirthInterval: rebirthInterval,
	}
}

// Reset forgets the births of every edge node and leaves a rebirth request for each. A
// host that lost its connection has missed messages, so it must see a new birth from
// each one.
func (h *Host) Reset() {
	for _, n := range h.nodes {
		n.born, n.tables, n.rebirth = false, nil, true
	}
}

// Forget drops the state of node, for a host that no longer follows it.
func (h *Host) Forget(node NodeID) { delete(h.nodes, node) }

// WantRebirth leaves a rebirth request for node, which a birth from node answers.
func (h *Host) WantRebirth(node NodeID) { h.node(node).rebirth = true }

// TakeRebirths returns the edge nodes that the host must send a rebirth request to at
// now, and takes their requests. A node gets at most one request in each rebirth
// interval, so wait is the time until a held request comes due, or zero when none is
// held.
func (h *Host) TakeRebirths(now time.Time) (due []NodeID, wait time.Duration) {
	for id, n := range h.nodes {
		if !n.rebirth {
			continue
		}
		if left := h.rebirthInterval - now.Sub(n.lastRebirth); left > 0 {
			if wait == 0 || left < wait {
				wait = left
			}
			continue
		}
		n.rebirth, n.lastRebirth = false, now
		due = append(due, id)
	}
	return due, wait
}

func (h *Host) node(id NodeID) *nodeState {
	n, ok := h.nodes[id]
	if !ok {
		n = &nodeState{}
		h.nodes[id] = n
	}
	return n
}

// Handle applies one message to the session state. A message whose type is not part of
// a session gives an empty event. An event that asks for a rebirth leaves a rebirth
// request.
func (h *Host) Handle(m Message) Event {
	ev := h.handle(m)
	if ev.Rebirth {
		h.node(ev.Node).rebirth = true
	}
	return ev
}

func (h *Host) handle(m Message) Event {
	topic, p := m.Topic, m.payload
	if !topic.Type.InSession() {
		return Event{}
	}
	var (
		n       = h.node(topic.Node)
		dropped = Event{Node: topic.Node, Device: topic.Device}
		ev      = Event{Type: topic.Type, Node: topic.Node, Device: topic.Device}
	)
	switch topic.Type {
	case NBirth:
		n.born, n.rebirth = true, false
		n.nextSeq = uint8(p.GetSeq()) + 1
		n.tables = map[string]tagTable{"": birthTable(p.Metrics)}
		n.bdSeq, n.bdSeqKnown = findBdSeq(p.Metrics)
		ev.Metrics = birthMetrics(p)
		return ev
	case NDeath:
		// A death with another bdSeq belongs to an older session of the edge node.
		if bdSeq, ok := findBdSeq(p.Metrics); ok && n.bdSeqKnown && bdSeq != n.bdSeq {
			return dropped
		}
		n.born, n.tables = false, nil
		return ev
	}
	if !n.born {
		dropped.Rebirth = true
		return dropped
	}
	if p.Seq != nil {
		if uint8(p.GetSeq()) != n.nextSeq {
			n.born, n.tables = false, nil
			dropped.Rebirth, dropped.BirthLost = true, true
			return dropped
		}
		n.nextSeq++
	}
	switch topic.Type {
	case DBirth:
		n.tables[topic.Device] = birthTable(p.Metrics)
		ev.Metrics = birthMetrics(p)
	case DDeath:
		delete(n.tables, topic.Device)
	default:
		table, ok := n.tables[topic.Device]
		if !ok {
			dropped.Rebirth = true
			return dropped
		}
		ev.Metrics, ev.Rebirth = dataMetrics(p, table)
	}
	return ev
}

// birthTable returns the tags that the metrics of a birth declare.
func birthTable(metrics []*pb.Payload_Metric) tagTable {
	t := makeTagTable(len(metrics))
	for _, m := range metrics {
		tg := birthTag(m)
		t.byName[tg.name] = tg
		if m.Alias != nil {
			t.byAlias[tg.alias] = tg
		}
	}
	return t
}

// birthTag returns what the metric m of a birth declares about its tag.
func birthTag(m *pb.Payload_Metric) tag {
	return tag{
		name:     m.GetName(),
		alias:    m.GetAlias(),
		dataType: DataType(m.GetDatatype()),
	}
}

func findBdSeq(metrics []*pb.Payload_Metric) (uint64, bool) {
	for _, m := range metrics {
		if m.GetName() != BdSeqMetric {
			continue
		}
		if v, ok := decodeValue(m, UInt64).(UintValue); ok {
			return uint64(v), true
		}
	}
	return 0, false
}

func birthMetrics(p *pb.Payload) []Metric {
	metrics := make([]Metric, 0, len(p.Metrics))
	for _, m := range p.Metrics {
		metrics = append(metrics, decodeMetric(p, m, birthTag(m)))
	}
	return metrics
}

// dataMetrics resolves the tags of a data message against the table of its birth. It
// reports a tag that the birth did not declare as a need for a rebirth.
func dataMetrics(p *pb.Payload, table tagTable) (metrics []Metric, rebirth bool) {
	metrics = make([]Metric, 0, len(p.Metrics))
	for _, m := range p.Metrics {
		tg, ok := table.resolve(m)
		if !ok {
			rebirth = true
			continue
		}
		metrics = append(metrics, decodeMetric(p, m, tg))
	}
	return metrics, rebirth
}
