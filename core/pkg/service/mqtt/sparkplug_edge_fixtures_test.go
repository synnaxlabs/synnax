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
	"fmt"
	"sync"

	mochi "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/proto"
)

// hostMessage is one message of an edge node that a test host received.
type hostMessage struct {
	topic    string
	payload  *pb.Payload
	qos      byte
	retained bool
}

// metric returns the metric of m that name names. It fails when there is none.
func (m hostMessage) metric(name string) *pb.Payload_Metric {
	GinkgoHelper()
	for _, pm := range m.payload.Metrics {
		if pm.GetName() == name {
			return pm
		}
	}
	Fail("no metric named " + name + " on " + m.topic)
	return nil
}

// metricByAlias returns the metric of m that carries alias. It fails when there is
// none.
func (m hostMessage) metricByAlias(alias uint64) *pb.Payload_Metric {
	GinkgoHelper()
	for _, pm := range m.payload.Metrics {
		if pm.Alias != nil && pm.GetAlias() == alias {
			return pm
		}
	}
	Fail(fmt.Sprintf("no metric with alias %d on %s", alias, m.topic))
	return nil
}

// bdSeq returns the value of the bdSeq metric of m.
func (m hostMessage) bdSeq() uint64 {
	GinkgoHelper()
	return m.metric(sparkplug.BdSeqMetric).GetLongValue()
}

// testHost is a Sparkplug B host application on a test broker. It records every
// message of one edge node, and sends commands to it.
type testHost struct {
	broker *testBroker
	node   sparkplug.NodeID
	mu     struct {
		sync.Mutex
		messages []hostMessage
	}
}

// startHost starts a host that follows the edge node group/edgeNode.
func startHost(b *testBroker, group, edgeNode string) *testHost {
	GinkgoHelper()
	h := &testHost{broker: b, node: sparkplug.NodeID{Group: group, EdgeNode: edgeNode}}
	onMessage := func(_ *mochi.Client, _ packets.Subscription, pk packets.Packet) {
		var p pb.Payload
		if proto.Unmarshal(pk.Payload, &p) != nil {
			return
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		h.mu.messages = append(h.mu.messages, hostMessage{
			topic:    pk.TopicName,
			payload:  &p,
			qos:      pk.FixedHeader.Qos,
			retained: pk.FixedHeader.Retain,
		})
	}
	Expect(b.server.Subscribe(h.node.Filters()[0], 20, onMessage)).To(Succeed())
	return h
}

// messages returns the messages of type t that the host received, in order.
func (h *testHost) messages(t sparkplug.MessageType) []hostMessage {
	topic := sparkplug.Topic{Type: t, Node: h.node}.String()
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []hostMessage
	for _, m := range h.mu.messages {
		if m.topic == topic {
			out = append(out, m)
		}
	}
	return out
}

func (h *testHost) births() []hostMessage { return h.messages(sparkplug.NBirth) }

func (h *testHost) deaths() []hostMessage { return h.messages(sparkplug.NDeath) }

func (h *testHost) data() []hostMessage { return h.messages(sparkplug.NData) }

// dataAt returns the NDATA messages whose first metric is stamped at stamp.
func (h *testHost) dataAt(stamp telem.TimeStamp) []hostMessage {
	millis := uint64(stamp / telem.MillisecondTS)
	var out []hostMessage
	for _, m := range h.data() {
		if len(m.payload.Metrics) > 0 && m.payload.Metrics[0].GetTimestamp() == millis {
			out = append(out, m)
		}
	}
	return out
}

// sequence returns the sequence numbers of the birth and data messages, in order.
func (h *testHost) sequence() []uint64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []uint64
	for _, m := range h.mu.messages {
		if m.payload.Seq != nil {
			out = append(out, m.payload.GetSeq())
		}
	}
	return out
}

func (h *testHost) send(payload []byte) {
	GinkgoHelper()
	topic := sparkplug.CommandTopic(h.node, "").String()
	Expect(h.broker.server.Publish(topic, payload, false, 0)).To(Succeed())
}

// sendRebirth asks the edge node for its birth again.
func (h *testHost) sendRebirth() {
	GinkgoHelper()
	h.send(MustSucceed(sparkplug.EncodeRebirth(telem.Now())))
}

// sendCommand sends one NCMD that carries metrics.
func (h *testHost) sendCommand(metrics ...*pb.Payload_Metric) {
	GinkgoHelper()
	h.send(MustSucceed(proto.Marshal(&pb.Payload{
		Timestamp: new(uint64(telem.Now() / telem.MillisecondTS)),
		Metrics:   metrics,
	})))
}

// commandMetric returns the metric of one command value. It names the tag by name
// when name is not empty, and by alias otherwise.
func commandMetric(name string, alias uint64, value any) *pb.Payload_Metric {
	GinkgoHelper()
	m := tagValue{name: name, alias: alias, value: value}.metric(name != "")
	if name != "" {
		m.Alias = nil
	}
	return m
}
