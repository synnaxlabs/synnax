// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	broker "github.com/synnaxlabs/synnax/pkg/service/mqtt/broker/versions/v0"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("ValidateWrite", func() {
	newBroker := func() broker.Broker {
		var b broker.Broker
		b.SetDefaults()
		b.Rack = 1
		b.Name = "Plant Broker"
		b.Host = "mqtt.example.com"
		return b
	}
	plainRead := func(topic, pointer string, message uuid.UUID) broker.Read {
		return broker.Read{Variant: broker.PlainRead{
			Topic:   topic,
			Message: message,
			Pointer: pointer,
			Index:   1,
			Channel: 2,
		}}
	}
	sparkplugRead := func(device string) broker.Read {
		return broker.Read{Variant: broker.SparkplugRead{
			Group:    "Plant",
			EdgeNode: "Line1",
			Device:   device,
			Tag:      "Pressure",
			Index:    1,
			Channel:  2,
		}}
	}
	plainWrite := func(topic string, field uuid.UUID) broker.Write {
		return broker.Write{Variant: broker.PlainWrite{
			Topic:   topic,
			Field:   field,
			Channel: 3,
		}}
	}
	sparkplugWrite := func(tag string) broker.Write {
		return broker.Write{Variant: broker.SparkplugWrite{
			Group:    "Plant",
			EdgeNode: "Line1",
			Tag:      tag,
			Channel:  3,
		}}
	}
	edgeCommand := func(tag string) broker.EdgeCommand {
		return broker.EdgeCommand{
			Group:    "Plant",
			EdgeNode: "Synnax",
			Tag:      tag,
			Channel:  4,
		}
	}

	It("Should accept bindings that differ in one part of their key", func() {
		b := newBroker()
		b.Sparkplug.Groups = []string{"Plant", "Line A"}
		message, field := uuid.New(), uuid.New()
		b.Reads = []broker.Read{
			plainRead("plant/temp", "/t", message),
			plainRead("plant/temp", "/p", message),
			plainRead("plant/temp", "/t", uuid.New()),
			plainRead("plant/other", "/t", message),
			sparkplugRead(""),
			sparkplugRead("Pump"),
		}
		b.Writes = []broker.Write{
			plainWrite("plant/valve", field),
			plainWrite("plant/valve", uuid.New()),
			plainWrite("plant/pump", field),
			sparkplugWrite("Speed"),
			sparkplugWrite("Mode"),
		}
		b.EdgeCommands = []broker.EdgeCommand{
			edgeCommand("Setpoint"),
			edgeCommand("Mode"),
		}
		Expect(b.ValidateWrite()).To(Succeed())
	})

	DescribeTable("Should reject an invalid broker",
		func(change func(*broker.Broker), path string) {
			b := newBroker()
			change(&b)
			Expect(b.ValidateWrite()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("a schema violation", func(b *broker.Broker) { b.Name = "" }, "name"),
		Entry("a keep alive of a fraction of a second", func(b *broker.Broker) {
			b.KeepAlive = 1500 * telem.Millisecond
		}, "keep_alive"),
		Entry("a Sparkplug group with a wildcard", func(b *broker.Broker) {
			b.Sparkplug.Groups = []string{"Plant", "+"}
		}, "groups"),
		Entry("a Sparkplug group with a slash", func(b *broker.Broker) {
			b.Sparkplug.Groups = []string{"Plant/A"}
		}, "groups"),
		Entry("an empty Sparkplug group", func(b *broker.Broker) {
			b.Sparkplug.Groups = []string{""}
		}, "groups"),
		Entry("a client ID with NUL", func(b *broker.Broker) {
			b.ClientID = new("core\x00")
		}, "client_id"),
		Entry("a duplicate Sparkplug group", func(b *broker.Broker) {
			b.Sparkplug.Groups = []string{"Plant", "Plant"}
		}, "groups.1"),
		Entry("a duplicate plain read", func(b *broker.Broker) {
			m := uuid.New()
			b.Reads = []broker.Read{
				plainRead("plant/temp", "/t", m),
				plainRead("plant/temp", "/t", m),
			}
		}, "reads.1"),
		Entry("a duplicate Sparkplug read", func(b *broker.Broker) {
			b.Reads = []broker.Read{
				sparkplugRead("Pump"),
				sparkplugRead("Pump"),
			}
		}, "reads.1"),
		Entry("a duplicate plain write", func(b *broker.Broker) {
			f := uuid.New()
			b.Writes = []broker.Write{
				plainWrite("plant/valve", f),
				plainWrite("plant/valve", f),
			}
		}, "writes.1"),
		Entry("a duplicate Sparkplug write", func(b *broker.Broker) {
			b.Writes = []broker.Write{sparkplugWrite("Speed"), sparkplugWrite("Speed")}
		}, "writes.1"),
		Entry("a duplicate edge command", func(b *broker.Broker) {
			b.EdgeCommands = []broker.EdgeCommand{
				edgeCommand("Setpoint"),
				edgeCommand("Setpoint"),
			}
		}, "edge_commands.1"),
	)
})
