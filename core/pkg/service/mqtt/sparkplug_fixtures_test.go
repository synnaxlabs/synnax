// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mqtt_test

import (
	"sync"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/proto"
)

// tagValue is the value of one tag of a test edge node.
type tagValue struct {
	// value is a float64, an int32, a bool, or a string.
	value any
	name  string
	alias uint64
	// timestamp is in milliseconds. Zero leaves the timestamp out.
	timestamp uint64
}

// metric returns the wire form of v. A data message names its tag by alias only.
func (v tagValue) metric(birth bool) *pb.Payload_Metric {
	m := &pb.Payload_Metric{Alias: new(v.alias)}
	if birth {
		m.Name = new(v.name)
	}
	if v.timestamp != 0 {
		m.Timestamp = new(v.timestamp)
	}
	switch value := v.value.(type) {
	case float64:
		m.Datatype = new(uint32(sparkplug.Double))
		m.Value = &pb.Payload_Metric_DoubleValue{DoubleValue: value}
	case int32:
		m.Datatype = new(uint32(sparkplug.Int32))
		m.Value = &pb.Payload_Metric_IntValue{IntValue: uint32(value)}
	case bool:
		m.Datatype = new(uint32(sparkplug.Boolean))
		m.Value = &pb.Payload_Metric_BooleanValue{BooleanValue: value}
	case string:
		m.Datatype = new(uint32(sparkplug.String))
		m.Value = &pb.Payload_Metric_StringValue{StringValue: value}
	default:
		Fail("unsupported tag value")
	}
	return m
}

// command is one NCMD or DCMD message that a test edge node received.
type command struct {
	topic   string
	payload *pb.Payload
	qos     byte
}

// testEdgeNode is a Sparkplug B edge node on a test broker. It answers a rebirth
// request with its birth messages.
type testEdgeNode struct {
	broker *testBroker
	id     sparkplug.NodeID
	// rebirths wakes the goroutine that answers rebirth requests.
	rebirths chan struct{}
	mu       struct {
		// tags holds the tags of the edge node under the empty device ID, and the tags
		// of each device under its ID.
		tags     map[string][]tagValue
		commands []command
		sync.Mutex
		births int
		bdSeq  uint64
		seq    uint8
		// silent makes the edge node ignore rebirth requests.
		silent bool
	}
}

// startEdgeNode starts an edge node that has not published its birth yet.
func startEdgeNode(b *testBroker, group, edgeNode string) *testEdgeNode {
	GinkgoHelper()
	n := &testEdgeNode{
		broker:   b,
		id:       sparkplug.NodeID{Group: group, EdgeNode: edgeNode},
		rebirths: make(chan struct{}, 1),
	}
	n.mu.tags = make(map[string][]tagValue)
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			case <-n.rebirths:
				n.birth()
			}
		}
	}()
	DeferCleanup(func() {
		close(done)
		<-stopped
	})
	onCommand := func(
		_ *mochi.Client,
		_ packets.Subscription,
		pk packets.Packet,
	) {
		topic, err := sparkplug.ParseTopic(pk.TopicName)
		if err != nil ||
			(topic.Type != sparkplug.NCmd && topic.Type != sparkplug.DCmd) {
			return
		}
		var p pb.Payload
		if proto.Unmarshal(pk.Payload, &p) != nil {
			return
		}
		n.mu.Lock()
		n.mu.commands = append(n.mu.commands, command{
			topic: pk.TopicName, payload: &p, qos: pk.FixedHeader.Qos,
		})
		silent := n.mu.silent
		n.mu.Unlock()
		for _, m := range p.Metrics {
			if m.GetName() == sparkplug.RebirthMetric && m.GetBooleanValue() &&
				!silent {
				select {
				case n.rebirths <- struct{}{}:
				default:
				}
			}
		}
	}
	for i, filter := range n.id.Filters() {
		Expect(b.server.Subscribe(filter, 10+i, onCommand)).To(Succeed())
	}
	return n
}

// setTags sets the tags that the next birth of device declares. An empty device
// selects the edge node.
func (n *testEdgeNode) setTags(device string, tags ...tagValue) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.mu.tags[device] = tags
}

func (n *testEdgeNode) setSilent(silent bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.mu.silent = silent
}

func (n *testEdgeNode) send(
	msgType sparkplug.MessageType,
	device string,
	p *pb.Payload,
) {
	GinkgoHelper()
	topic := sparkplug.Topic{Type: msgType, Node: n.id, Device: device}.String()
	Expect(n.broker.server.Publish(topic, MustSucceed(proto.Marshal(p)), false, 0)).
		To(Succeed())
}

// nextSeq returns the sequence number of the next message. The caller holds n.mu.
func (n *testEdgeNode) nextSeq() *uint64 {
	seq := uint64(n.mu.seq)
	n.mu.seq++
	return &seq
}

// birth publishes the birth of the edge node and of each of its devices.
func (n *testEdgeNode) birth() {
	GinkgoHelper()
	n.mu.Lock()
	defer n.mu.Unlock()
	n.mu.seq = 0
	n.mu.bdSeq++
	n.mu.births++
	for _, device := range append([]string{""}, n.devices()...) {
		p := &pb.Payload{Seq: n.nextSeq()}
		msgType := sparkplug.DBirth
		if device == "" {
			msgType = sparkplug.NBirth
			p.Metrics = append(p.Metrics, &pb.Payload_Metric{
				Name:     new(sparkplug.BdSeqMetric),
				Datatype: new(uint32(sparkplug.UInt64)),
				Value:    &pb.Payload_Metric_LongValue{LongValue: n.mu.bdSeq},
			})
		}
		for _, t := range n.mu.tags[device] {
			p.Metrics = append(p.Metrics, t.metric(true))
		}
		n.send(msgType, device, p)
	}
}

// devices returns the device IDs in a fixed order. The caller holds n.mu.
func (n *testEdgeNode) devices() []string {
	var out []string
	for device := range n.mu.tags {
		if device != "" {
			out = append(out, device)
		}
	}
	return out
}

// data publishes new values of tags of device, and keeps them for the next birth.
func (n *testEdgeNode) data(device string, values ...tagValue) {
	GinkgoHelper()
	n.mu.Lock()
	defer n.mu.Unlock()
	p := &pb.Payload{Seq: n.nextSeq()}
	for _, v := range values {
		p.Metrics = append(p.Metrics, v.metric(false))
		for i, t := range n.mu.tags[device] {
			if t.alias == v.alias {
				v.name = t.name
				n.mu.tags[device][i] = v
			}
		}
	}
	msgType := sparkplug.NData
	if device != "" {
		msgType = sparkplug.DData
	}
	n.send(msgType, device, p)
}

// skipSeq makes the next message leave a gap in the sequence.
func (n *testEdgeNode) skipSeq() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.mu.seq++
}

// death publishes the death of the edge node.
func (n *testEdgeNode) death() {
	GinkgoHelper()
	n.mu.Lock()
	defer n.mu.Unlock()
	n.send(sparkplug.NDeath, "", &pb.Payload{Metrics: []*pb.Payload_Metric{{
		Name:     new(sparkplug.BdSeqMetric),
		Datatype: new(uint32(sparkplug.UInt64)),
		Value:    &pb.Payload_Metric_LongValue{LongValue: n.mu.bdSeq},
	}}})
}

// deviceDeath publishes the death of device.
func (n *testEdgeNode) deviceDeath(device string) {
	GinkgoHelper()
	n.mu.Lock()
	defer n.mu.Unlock()
	n.send(sparkplug.DDeath, device, &pb.Payload{Seq: n.nextSeq()})
}

// births returns the count of births that the edge node published.
func (n *testEdgeNode) births() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.mu.births
}

// received returns the commands that are not rebirth requests.
func (n *testEdgeNode) received() []command {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []command
	for _, c := range n.mu.commands {
		if len(c.payload.Metrics) == 1 &&
			c.payload.Metrics[0].GetName() == sparkplug.RebirthMetric {
			continue
		}
		out = append(out, c)
	}
	return out
}

// rebirthRequests returns the rebirth requests that the edge node received.
func (n *testEdgeNode) rebirthRequests() []command {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []command
	for _, c := range n.mu.commands {
		if len(c.payload.Metrics) == 1 &&
			c.payload.Metrics[0].GetName() == sparkplug.RebirthMetric {
			out = append(out, c)
		}
	}
	return out
}

// dropClients closes the network connection of every client, as a network failure
// does. The broker then publishes the last will of each one.
func (b *testBroker) dropClients() {
	GinkgoHelper()
	for _, cl := range b.server.Clients.GetAll() {
		if !cl.Net.Inline {
			Expect(cl.Net.Conn.Close()).To(Succeed())
		}
	}
}
