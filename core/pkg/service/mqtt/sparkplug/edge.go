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
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/proto"
)

// Tag declares one tag of an edge node.
type Tag struct {
	Name     string
	DataType DataType
}

// Edge is the fixed state of one edge node: its ID and its tags with their aliases. It
// is safe for concurrent use.
type Edge struct {
	node NodeID
	// tags holds the declared tags in alias order.
	tags  []tag
	table tagTable
}

// NewEdge returns the edge node node that declares tags. The alias of each tag is its
// position in tags, from 1.
func NewEdge(node NodeID, tags []Tag) *Edge {
	e := &Edge{node: node, tags: make([]tag, len(tags)), table: makeTagTable(len(tags))}
	for i, t := range tags {
		tg := tag{name: t.Name, alias: uint64(i) + 1, dataType: t.DataType}
		e.tags[i] = tg
		e.table.byName[tg.name] = tg
		e.table.byAlias[tg.alias] = tg
	}
	return e
}

func (e *Edge) topic(
	t MessageType,
) string {
	return Topic{Type: t, Node: e.node}.String()
}

// Command is what one NCMD message asks of an edge node.
type Command struct {
	// Rebirth is true when the host asks for the birth messages again.
	Rebirth bool
	// Metrics holds the values to write, with every alias resolved to its tag name.
	// Metrics of tags that the edge node does not declare are left out.
	Metrics []Metric
}

// Command returns what m asks of the edge node. A message that is not an NCMD message
// to the edge node asks nothing.
func (e *Edge) Command(m Message) Command {
	var cmd Command
	if m.Topic.Type != NCmd || m.Topic.Node != e.node {
		return cmd
	}
	for _, pm := range m.payload.Metrics {
		if pm.GetName() == RebirthMetric {
			cmd.Rebirth = cmd.Rebirth || pm.GetBooleanValue()
			continue
		}
		if tg, ok := e.table.resolve(pm); ok {
			cmd.Metrics = append(cmd.Metrics, decodeMetric(m.payload, pm, tg))
		}
	}
	return cmd
}

// Sequence is the changing state of an edge node: its birth-death sequence number and
// its message sequence number. It is not safe for concurrent use.
type Sequence struct {
	edge  *Edge
	bdSeq uint8
	seq   uint8
}

// NewSequence returns the sequence state of e, before its first session.
func NewSequence(e *Edge) *Sequence { return &Sequence{edge: e} }

func (s *Sequence) bdSeqMetric() *pb.Payload_Metric {
	return &pb.Payload_Metric{
		Name:     new(BdSeqMetric),
		Datatype: new(uint32(Int64)),
		Value:    &pb.Payload_Metric_LongValue{LongValue: uint64(s.bdSeq)},
	}
}

// Death returns the NDEATH message of the current session, at quality of service 1. An
// edge node sets it as the last will before it connects, and publishes it before a
// clean disconnect.
func (s *Sequence) Death() (Publication, error) {
	payload, err := proto.Marshal(
		&pb.Payload{Metrics: []*pb.Payload_Metric{s.bdSeqMetric()}},
	)
	if err != nil {
		return Publication{}, err
	}
	return Publication{Topic: s.edge.topic(NDeath), Payload: payload, QoS: 1}, nil
}

// Birth returns the NBIRTH message that starts the sequence of the current session.
// values holds the current value of each tag by name. A tag with no value is null. A
// birth that fails leaves the sequence as it was.
func (s *Sequence) Birth(
	values map[string]Metric,
	now telem.TimeStamp,
) (Publication, error) {
	metrics := make([]*pb.Payload_Metric, 0, len(s.edge.tags)+2)
	metrics = append(metrics, s.bdSeqMetric(), &pb.Payload_Metric{
		Name:     new(RebirthMetric),
		Datatype: new(uint32(Boolean)),
		Value:    &pb.Payload_Metric_BooleanValue{BooleanValue: false},
	})
	for _, t := range s.edge.tags {
		v, ok := values[t.name]
		m := &pb.Payload_Metric{
			Name:      new(t.name),
			Alias:     new(t.alias),
			Datatype:  new(uint32(t.dataType)),
			Timestamp: toMillis(stampOr(v.Timestamp, now)),
		}
		if !ok {
			m.IsNull = new(true)
		} else if err := encodeValue(m, t.dataType, v.Value); err != nil {
			return Publication{}, errors.Wrapf(err, "tag %s", t.name)
		}
		metrics = append(metrics, m)
	}
	payload, err := marshal(metrics, now, 0)
	if err != nil {
		return Publication{}, err
	}
	s.seq = 1
	return Publication{Topic: s.edge.topic(NBirth), Payload: payload}, nil
}

// Data returns an NDATA message that carries the metrics that fit their tags, and those
// metrics. Each metric names its tag, and the message identifies the tag by its alias
// only. A metric with a zero timestamp takes now. The error names each metric that Data
// leaves out. When Data leaves out every metric, the payload is nil and the sequence
// does not move.
func (s *Sequence) Data(
	metrics []Metric,
	now telem.TimeStamp,
) (pub Publication, carried []Metric, err error) {
	out := make([]*pb.Payload_Metric, 0, len(metrics))
	carried = make([]Metric, 0, len(metrics))
	for _, m := range metrics {
		pm := &pb.Payload_Metric{Timestamp: toMillis(stampOr(m.Timestamp, now))}
		tg, encodeErr := s.edge.encode(pm, m)
		if encodeErr != nil {
			err = errors.Join(err, encodeErr)
			continue
		}
		pm.Alias, pm.Datatype = new(tg.alias), new(uint32(tg.dataType))
		out = append(out, pm)
		carried = append(carried, m)
	}
	if err != nil && len(out) == 0 {
		return Publication{}, nil, err
	}
	payload, marshalErr := marshal(out, now, s.seq)
	if marshalErr != nil {
		return Publication{}, nil, marshalErr
	}
	s.seq++
	return Publication{Topic: s.edge.topic(NData), Payload: payload}, carried, err
}

// EndSession moves to the next birth-death sequence number. An edge node calls it after
// each connection ends, before it builds the last will of the next one.
func (s *Sequence) EndSession() { s.bdSeq++ }

// encode sets the value of pm to the value of m as the data type of the tag that m
// names, and returns that tag.
func (e *Edge) encode(pm *pb.Payload_Metric, m Metric) (tag, error) {
	tg, ok := e.table.byName[m.Name]
	if !ok {
		return tag{}, errors.Newf("tag %s is not a tag of the edge node", m.Name)
	}
	if err := encodeValue(pm, tg.dataType, m.Value); err != nil {
		return tag{}, errors.Wrapf(err, "tag %s", m.Name)
	}
	return tg, nil
}

// stampOr returns stamp, or now when stamp is zero.
func stampOr(stamp, now telem.TimeStamp) telem.TimeStamp {
	if stamp == 0 {
		return now
	}
	return stamp
}

func marshal(
	metrics []*pb.Payload_Metric,
	now telem.TimeStamp,
	seq uint8,
) ([]byte, error) {
	return proto.Marshal(&pb.Payload{
		Timestamp: toMillis(now),
		Metrics:   metrics,
		Seq:       new(uint64(seq)),
	})
}
