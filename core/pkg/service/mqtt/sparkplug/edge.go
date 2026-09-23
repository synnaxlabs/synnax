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

// Edge is the session state of one edge node: its aliases, its birth-death sequence
// number, and its message sequence number. It is not safe for concurrent use.
type Edge struct {
	tags    []Tag
	aliases map[string]uint64
	byAlias map[uint64]Tag
	bdSeq   uint8
	seq     uint8
}

// NewEdge returns the session state of an edge node that declares tags. The alias of
// each tag is its position in tags, from 1.
func NewEdge(tags []Tag) *Edge {
	e := &Edge{
		tags:    tags,
		aliases: make(map[string]uint64, len(tags)),
		byAlias: make(map[uint64]Tag, len(tags)),
	}
	for i, t := range tags {
		alias := uint64(i) + 1
		e.aliases[t.Name] = alias
		e.byAlias[alias] = t
	}
	return e
}

func (e *Edge) bdSeqMetric() *pb.Payload_Metric {
	return &pb.Payload_Metric{
		Name:     new(BdSeqMetric),
		Datatype: new(uint32(Int64)),
		Value:    &pb.Payload_Metric_LongValue{LongValue: uint64(e.bdSeq)},
	}
}

// Death returns the NDEATH payload of the current session. An edge node sets it as
// the last will before it connects, and publishes it before a clean disconnect.
func (e *Edge) Death() ([]byte, error) {
	return proto.Marshal(&pb.Payload{Metrics: []*pb.Payload_Metric{e.bdSeqMetric()}})
}

// Birth returns the NBIRTH payload that starts the sequence of the current session.
// values holds the current value of each tag by name. A tag with no value is null.
func (e *Edge) Birth(values map[string]Metric, now telem.TimeStamp) ([]byte, error) {
	e.seq = 0
	metrics := make([]*pb.Payload_Metric, 0, len(e.tags)+2)
	metrics = append(metrics, e.bdSeqMetric(), &pb.Payload_Metric{
		Name:     new(RebirthMetric),
		Datatype: new(uint32(Boolean)),
		Value:    &pb.Payload_Metric_BooleanValue{BooleanValue: false},
	})
	for _, t := range e.tags {
		m := &pb.Payload_Metric{
			Name:      new(t.Name),
			Alias:     new(e.aliases[t.Name]),
			Datatype:  new(uint32(t.DataType)),
			Timestamp: toMillis(now),
		}
		if v, ok := values[t.Name]; ok {
			if v.Timestamp != 0 {
				m.Timestamp = toMillis(v.Timestamp)
			}
			if err := encodeValue(m, t.DataType, v.Value); err != nil {
				return nil, errors.Wrapf(err, "tag %s", t.Name)
			}
		} else {
			m.IsNull = new(true)
		}
		metrics = append(metrics, m)
	}
	return e.marshal(metrics, now)
}

// Data returns an NDATA payload that carries metrics. Each metric names its tag, and
// the payload identifies the tag by its alias only. A metric with a zero timestamp
// takes now.
func (e *Edge) Data(metrics []Metric, now telem.TimeStamp) ([]byte, error) {
	out := make([]*pb.Payload_Metric, 0, len(metrics))
	for _, m := range metrics {
		alias, ok := e.aliases[m.Name]
		if !ok {
			return nil, errors.Newf("tag %s is not a tag of the edge node", m.Name)
		}
		dt := e.byAlias[alias].DataType
		pm := &pb.Payload_Metric{
			Alias:     new(alias),
			Datatype:  new(uint32(dt)),
			Timestamp: toMillis(now),
		}
		if m.Timestamp != 0 {
			pm.Timestamp = toMillis(m.Timestamp)
		}
		if err := encodeValue(pm, dt, m.Value); err != nil {
			return nil, errors.Wrapf(err, "tag %s", m.Name)
		}
		out = append(out, pm)
	}
	return e.marshal(out, now)
}

func (e *Edge) marshal(
	metrics []*pb.Payload_Metric,
	now telem.TimeStamp,
) ([]byte, error) {
	seq := uint64(e.seq)
	e.seq++
	return proto.Marshal(&pb.Payload{
		Timestamp: toMillis(now),
		Metrics:   metrics,
		Seq:       &seq,
	})
}

// EndSession moves to the next birth-death sequence number. An edge node calls it
// after each connection ends, before it builds the last will of the next one.
func (e *Edge) EndSession() { e.bdSeq++ }

// Command is what one NCMD message asks of an edge node.
type Command struct {
	// Rebirth is true when the host asks for the birth messages again.
	Rebirth bool
	// Metrics holds the values to write, with every alias resolved to its tag name.
	// Metrics of tags that the edge node does not declare are left out.
	Metrics []Metric
}

// DecodeCommand parses the payload of an NCMD message.
func (e *Edge) DecodeCommand(payload []byte) (Command, error) {
	var p pb.Payload
	if err := proto.Unmarshal(payload, &p); err != nil {
		return Command{}, errors.Wrap(err, "invalid command payload")
	}
	var cmd Command
	for _, m := range p.Metrics {
		if m.GetName() == RebirthMetric {
			cmd.Rebirth = cmd.Rebirth || m.GetBooleanValue()
			continue
		}
		var (
			t  Tag
			ok bool
		)
		if m.Name != nil && m.GetName() != "" {
			alias, known := e.aliases[m.GetName()]
			t, ok = e.byAlias[alias], known
		} else if m.Alias != nil {
			t, ok = e.byAlias[m.GetAlias()]
		}
		if !ok {
			continue
		}
		dt := t.DataType
		if m.Datatype != nil {
			dt = DataType(m.GetDatatype())
		}
		cmd.Metrics = append(cmd.Metrics, Metric{
			Name:      t.Name,
			DataType:  dt,
			Value:     decodeValue(m, dt),
			Timestamp: timestampOf(&p, m),
		})
	}
	return cmd, nil
}
