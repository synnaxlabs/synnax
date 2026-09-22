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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/proto"
)

// DefaultRebirthInterval is the shortest time between two rebirth requests to one edge
// node.
const DefaultRebirthInterval = 5 * time.Second

// tag is what a birth message declares about one tag.
type tag struct {
	name     string
	dataType DataType
}

// tagTable holds the tags of an edge node, or of one of its devices.
type tagTable struct {
	byName  map[string]tag
	byAlias map[uint64]tag
}

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
}

// Host is the session state of a Sparkplug B host application: for each edge node,
// the tags of its last birth, its aliases, and its sequence number. It is not safe for
// concurrent use.
type Host struct {
	nodes map[NodeID]*nodeState
	// RebirthInterval is the shortest time between two rebirth requests to one edge
	// node.
	RebirthInterval time.Duration
}

// NewHost returns a host that knows no edge node.
func NewHost() *Host {
	return &Host{
		nodes:           make(map[NodeID]*nodeState),
		RebirthInterval: DefaultRebirthInterval,
	}
}

// Reset forgets the births of every edge node. A host that lost its connection has
// missed messages, so it must see a new birth from each one.
func (h *Host) Reset() {
	for _, n := range h.nodes {
		n.born, n.tables = false, nil
	}
}

// Forget drops the state of node, for a host that no longer follows it.
func (h *Host) Forget(node NodeID) { delete(h.nodes, node) }

// Born reports whether the host holds a valid birth of node.
func (h *Host) Born(node NodeID) bool {
	n, ok := h.nodes[node]
	return ok && n.born
}

// TakeRebirth returns zero when the host may send a rebirth request to node at now,
// and records the request. Otherwise it returns the time left until it may.
func (h *Host) TakeRebirth(node NodeID, now time.Time) time.Duration {
	n := h.node(node)
	if left := h.RebirthInterval - now.Sub(n.lastRebirth); left > 0 {
		return left
	}
	n.lastRebirth = now
	return 0
}

func (h *Host) node(id NodeID) *nodeState {
	n, ok := h.nodes[id]
	if !ok {
		n = &nodeState{}
		h.nodes[id] = n
	}
	return n
}

// DecodePayload parses the payload of a message on topic. It needs no session state,
// so a host can decode outside the lock that guards it.
func DecodePayload(topic Topic, payload []byte) (*pb.Payload, error) {
	p := new(pb.Payload)
	if err := proto.Unmarshal(payload, p); err != nil {
		return nil, errors.Wrapf(err, "invalid payload on %s", topic)
	}
	return p, nil
}

// Handle applies one decoded message to the session state. A message whose type is
// not part of a session gives an empty event.
func (h *Host) Handle(topic Topic, p *pb.Payload) Event {
	if !topic.Type.Session() {
		return Event{}
	}
	var (
		n       = h.node(topic.Node)
		dropped = Event{Node: topic.Node, Device: topic.Device}
		ev      = Event{Type: topic.Type, Node: topic.Node, Device: topic.Device}
	)
	switch topic.Type {
	case NBirth:
		n.born = true
		n.nextSeq = uint8(p.GetSeq()) + 1
		n.tables = map[string]tagTable{"": newTagTable(p.Metrics)}
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
			dropped.Rebirth = true
			return dropped
		}
		n.nextSeq++
	}
	switch topic.Type {
	case DBirth:
		n.tables[topic.Device] = newTagTable(p.Metrics)
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

func newTagTable(metrics []*pb.Payload_Metric) tagTable {
	t := tagTable{
		byName:  make(map[string]tag, len(metrics)),
		byAlias: make(map[uint64]tag),
	}
	for _, m := range metrics {
		tg := tag{name: m.GetName(), dataType: DataType(m.GetDatatype())}
		t.byName[tg.name] = tg
		if m.Alias != nil {
			t.byAlias[m.GetAlias()] = tg
		}
	}
	return t
}

func findBdSeq(metrics []*pb.Payload_Metric) (uint64, bool) {
	for _, m := range metrics {
		if m.GetName() != BdSeqMetric {
			continue
		}
		if v, ok := decodeValue(m, UInt64).(uint64); ok {
			return v, true
		}
	}
	return 0, false
}

func timestampOf(p *pb.Payload, m *pb.Payload_Metric) telem.TimeStamp {
	if m.Timestamp != nil {
		return telem.TimeStamp(m.GetTimestamp()) * millisecond
	}
	return telem.TimeStamp(p.GetTimestamp()) * millisecond
}

func birthMetrics(p *pb.Payload) []Metric {
	metrics := make([]Metric, 0, len(p.Metrics))
	for _, m := range p.Metrics {
		dt := DataType(m.GetDatatype())
		metrics = append(metrics, Metric{
			Name:       m.GetName(),
			DataType:   dt,
			Value:      decodeValue(m, dt),
			Timestamp:  timestampOf(p, m),
			Historical: m.GetIsHistorical(),
		})
	}
	return metrics
}

// dataMetrics resolves the tags of a data message against the table of its birth. It
// reports a tag that the birth did not declare as a need for a rebirth.
func dataMetrics(p *pb.Payload, table tagTable) (metrics []Metric, rebirth bool) {
	metrics = make([]Metric, 0, len(p.Metrics))
	for _, m := range p.Metrics {
		var (
			tg tag
			ok bool
		)
		if m.Name != nil && m.GetName() != "" {
			tg, ok = table.byName[m.GetName()]
		} else if m.Alias != nil {
			tg, ok = table.byAlias[m.GetAlias()]
		}
		if !ok {
			rebirth = true
			continue
		}
		dt := tg.dataType
		if m.Datatype != nil {
			dt = DataType(m.GetDatatype())
		}
		metrics = append(metrics, Metric{
			Name:       tg.name,
			DataType:   dt,
			Value:      decodeValue(m, dt),
			Timestamp:  timestampOf(p, m),
			Historical: m.GetIsHistorical(),
		})
	}
	return metrics, rebirth
}
