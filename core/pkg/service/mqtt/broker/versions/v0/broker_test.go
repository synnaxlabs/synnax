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
	"encoding/json/v2"
	"io"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/mqtt/broker/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Broker", func() {
	broker := func() v0.Broker {
		return v0.Broker{
			Key:       uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:      1,
			Name:      "Plant Broker",
			Host:      "mqtt.example.com",
			Transport: v0.Transport{Variant: v0.TcpTransport{}},
			Auth:      v0.Auth{Variant: v0.NoneAuth{}},
			KeepAlive: 30 * telem.Second,
			Sparkplug: v0.Sparkplug{
				Mode: v0.SparkplugMode{Variant: v0.PassiveSparkplugMode{}},
			},
		}
	}

	plainRead := func() v0.PlainRead {
		return v0.PlainRead{
			Topic:   "plant/temp",
			Message: uuid.New(),
			Index:   1,
			Channel: 2,
		}
	}

	sparkplugRead := func() v0.SparkplugRead {
		return v0.SparkplugRead{
			Group:    "Plant",
			EdgeNode: "Line1",
			Tag:      "Pressure",
			Index:    1,
			Channel:  2,
		}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Broker]()).To(Equal("mqtt_broker"))
	})

	It("Should identify itself by its key", func() {
		b := broker()
		Expect(b.GorpKey()).To(Equal(b.Key))
		Expect(b.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeMqttBroker,
			Key:  b.Key.String(),
		}))
	})

	It("Should apply the defaults", func() {
		var b v0.Broker
		Expect(json.Unmarshal([]byte(`{
			"key": "22222222-2222-4222-8222-222222222222",
			"rack": 1,
			"name": "Plant Broker",
			"host": "mqtt.example.com"
		}`), &b)).To(Succeed())
		Expect(b).To(Equal(broker()))
		Expect(b.Validate()).To(Succeed())
	})

	It("Should round trip a broker through JSON", func() {
		b := broker()
		b.Port = new(uint16(8883))
		b.ClientID = new("plant-core")
		b.Transport = v0.Transport{Variant: v0.TlsTransport{
			CaCertificate:                 new(uuid.New()),
			ClientCertificate:             new(uuid.New()),
			CertificateValidationDisabled: true,
		}}
		b.Auth = v0.Auth{Variant: v0.UsernameAuth{
			Username: "admin",
			Password: new(uuid.New()),
		}}
		b.Sparkplug = v0.Sparkplug{
			Mode:   v0.SparkplugMode{Variant: v0.PrimarySparkplugMode{HostID: "scada"}},
			Groups: []string{"Plant"},
		}
		b.Reads = []v0.Read{{Variant: plainRead()}, {Variant: sparkplugRead()}}
		b.Writes = []v0.Write{
			{Variant: v0.PlainWrite{
				Topic:   "plant/valve",
				Field:   uuid.New(),
				Channel: 3,
			}},
			{Variant: v0.SparkplugWrite{
				Group:    "Plant",
				EdgeNode: "Line1",
				Device:   "Pump",
				Tag:      "Speed",
				Channel:  4,
			}},
		}
		b.EdgeCommands = []v0.EdgeCommand{{
			Group:    "Plant",
			EdgeNode: "Synnax",
			Tag:      "Setpoint",
			Channel:  5,
		}}
		Expect(b.Validate()).To(Succeed())
		var decoded v0.Broker
		Expect(json.Unmarshal(MustSucceed(json.Marshal(b)), &decoded)).To(Succeed())
		Expect(decoded).To(Equal(b))
	})

	DescribeTable(
		"Should reject an invalid broker",
		func(change func(*v0.Broker), path string) {
			b := broker()
			change(&b)
			Expect(b.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("no name", func(b *v0.Broker) { b.Name = "" }, "name"),
		Entry("no host", func(b *v0.Broker) { b.Host = "" }, "host"),
		Entry("a host with a scheme", func(b *v0.Broker) {
			b.Host = "mqtt://mqtt.example.com"
		}, "host"),
		Entry("a host with a port", func(b *v0.Broker) {
			b.Host = "mqtt.example.com:1883"
		}, "host"),
		Entry("port 0", func(b *v0.Broker) { b.Port = new(uint16(0)) }, "port"),
		Entry("an empty client ID", func(b *v0.Broker) {
			b.ClientID = new("")
		}, "client_id"),
		Entry("a keep alive under 1 s", func(b *v0.Broker) {
			b.KeepAlive = 999 * telem.Millisecond
		}, "keep_alive"),
		Entry("a keep alive over 65535 s", func(b *v0.Broker) {
			b.KeepAlive = 65536 * telem.Second
		}, "keep_alive"),
		Entry(
			"no transport",
			func(b *v0.Broker) { b.Transport = v0.Transport{} },
			"type",
		),
		Entry("no auth", func(b *v0.Broker) { b.Auth = v0.Auth{} }, "type"),
		Entry("a user name auth with no user name", func(b *v0.Broker) {
			b.Auth = v0.Auth{Variant: v0.UsernameAuth{}}
		}, "username"),
		Entry("a primary host with no ID", func(b *v0.Broker) {
			b.Sparkplug.Mode = v0.SparkplugMode{Variant: v0.PrimarySparkplugMode{}}
		}, "host_id"),
		Entry("a primary host ID with a wildcard", func(b *v0.Broker) {
			b.Sparkplug.Mode = v0.SparkplugMode{
				Variant: v0.PrimarySparkplugMode{HostID: "scada/+"},
			}
		}, "host_id"),
		Entry("a plain read topic with NUL", func(b *v0.Broker) {
			r := plainRead()
			r.Topic = "plant/\x00"
			b.Reads = []v0.Read{{Variant: r}}
		}, "topic"),
		Entry("a plain write topic that starts with $", func(b *v0.Broker) {
			b.Writes = []v0.Write{{Variant: v0.PlainWrite{
				Topic:   "$SYS/valve",
				Field:   uuid.New(),
				Channel: 3,
			}}}
		}, "topic"),
		Entry("a Sparkplug tag with NUL", func(b *v0.Broker) {
			r := sparkplugRead()
			r.Tag = "Pressure\x00"
			b.Reads = []v0.Read{{Variant: r}}
		}, "tag"),
		Entry("a plain read with a wildcard topic", func(b *v0.Broker) {
			r := plainRead()
			r.Topic = "plant/#"
			b.Reads = []v0.Read{{Variant: r}}
		}, "topic"),
		Entry("a plain read with no message", func(b *v0.Broker) {
			r := plainRead()
			r.Message = uuid.Nil()
			b.Reads = []v0.Read{{Variant: r}}
		}, "message"),
		Entry("a plain read pointer with no leading slash", func(b *v0.Broker) {
			r := plainRead()
			r.Pointer = "temp"
			b.Reads = []v0.Read{{Variant: r}}
		}, "pointer"),
		Entry("a plain read with no index", func(b *v0.Broker) {
			r := plainRead()
			r.Index = 0
			b.Reads = []v0.Read{{Variant: r}}
		}, "index"),
		Entry("a Sparkplug read with no group", func(b *v0.Broker) {
			r := sparkplugRead()
			r.Group = ""
			b.Reads = []v0.Read{{Variant: r}}
		}, "group"),
		Entry("a Sparkplug read with a slash in its edge node", func(b *v0.Broker) {
			r := sparkplugRead()
			r.EdgeNode = "Line/1"
			b.Reads = []v0.Read{{Variant: r}}
		}, "edge_node"),
		Entry("a Sparkplug read with a wildcard device", func(b *v0.Broker) {
			r := sparkplugRead()
			r.Device = "#"
			b.Reads = []v0.Read{{Variant: r}}
		}, "device"),
		Entry("a Sparkplug read with no tag", func(b *v0.Broker) {
			r := sparkplugRead()
			r.Tag = ""
			b.Reads = []v0.Read{{Variant: r}}
		}, "tag"),
		Entry("a plain write with no field", func(b *v0.Broker) {
			b.Writes = []v0.Write{{Variant: v0.PlainWrite{Topic: "a", Channel: 3}}}
		}, "field"),
		Entry("a Sparkplug write with no channel", func(b *v0.Broker) {
			b.Writes = []v0.Write{{Variant: v0.SparkplugWrite{
				Group:    "Plant",
				EdgeNode: "Line1",
				Tag:      "Speed",
			}}}
		}, "channel"),
		Entry("an edge command with no channel", func(b *v0.Broker) {
			b.EdgeCommands = []v0.EdgeCommand{{
				Group:    "Plant",
				EdgeNode: "Synnax",
				Tag:      "Setpoint",
			}}
		}, "channel"),
		Entry("an edge command with a wildcard group", func(b *v0.Broker) {
			b.EdgeCommands = []v0.EdgeCommand{{
				Group:    "+",
				EdgeNode: "Synnax",
				Tag:      "Setpoint",
				Channel:  5,
			}}
		}, "group"),
	)

	DescribeTable("Should accept a valid broker",
		func(change func(*v0.Broker)) {
			b := broker()
			change(&b)
			Expect(b.Validate()).To(Succeed())
		},
		Entry("an IPv4 host", func(b *v0.Broker) { b.Host = "192.168.1.10" }),
		Entry("an IPv6 host", func(b *v0.Broker) { b.Host = "fe80::1" }),
		Entry("a single label host", func(b *v0.Broker) { b.Host = "localhost" }),
		Entry("TLS with no certificates", func(b *v0.Broker) {
			b.Transport = v0.Transport{Variant: v0.TlsTransport{}}
		}),
		Entry("a user name with no password", func(b *v0.Broker) {
			b.Auth = v0.Auth{Variant: v0.UsernameAuth{Username: "admin"}}
		}),
		Entry("a Sparkplug tag whose name holds a slash", func(b *v0.Broker) {
			r := sparkplugRead()
			r.Tag = "Node Control/Rebirth"
			b.Reads = []v0.Read{{Variant: r}}
		}),
		Entry("a whole message pointer", func(b *v0.Broker) {
			b.Reads = []v0.Read{{Variant: plainRead()}}
		}),
	)
})

var _ = Describe("msgpack", func() {
	roundTrip := func(in, out any) {
		GinkgoHelper()
		Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(in)), out)).To(Succeed())
	}

	DescribeTable("Should round trip a transport",
		func(t v0.Transport) {
			var decoded v0.Transport
			roundTrip(t, &decoded)
			Expect(decoded).To(Equal(t))
		},
		Entry("TCP", v0.Transport{Variant: v0.TcpTransport{}}),
		Entry("TLS", v0.Transport{Variant: v0.TlsTransport{
			CaCertificate:                 new(uuid.New()),
			ClientCertificate:             new(uuid.New()),
			CertificateValidationDisabled: true,
		}}),
		Entry("no variant", v0.Transport{}),
	)

	DescribeTable("Should round trip an auth",
		func(a v0.Auth) {
			var decoded v0.Auth
			roundTrip(a, &decoded)
			Expect(decoded).To(Equal(a))
		},
		Entry("none", v0.Auth{Variant: v0.NoneAuth{}}),
		Entry("user name", v0.Auth{Variant: v0.UsernameAuth{
			Username: "admin",
			Password: new(uuid.New()),
		}}),
		Entry("no variant", v0.Auth{}),
	)

	DescribeTable("Should round trip a Sparkplug mode",
		func(m v0.SparkplugMode) {
			var decoded v0.SparkplugMode
			roundTrip(m, &decoded)
			Expect(decoded).To(Equal(m))
		},
		Entry("passive", v0.SparkplugMode{Variant: v0.PassiveSparkplugMode{}}),
		Entry("primary", v0.SparkplugMode{
			Variant: v0.PrimarySparkplugMode{HostID: "scada"},
		}),
		Entry("no variant", v0.SparkplugMode{}),
	)

	DescribeTable("Should round trip a read",
		func(r v0.Read) {
			var decoded v0.Read
			roundTrip(r, &decoded)
			Expect(decoded).To(Equal(r))
		},
		Entry("plain", v0.Read{Variant: v0.PlainRead{
			Topic:   "plant/temp",
			Message: uuid.New(),
			Pointer: "/t",
			Index:   1,
			Channel: 2,
		}}),
		Entry("Sparkplug", v0.Read{Variant: v0.SparkplugRead{
			Group:    "Plant",
			EdgeNode: "Line1",
			Device:   "Pump",
			Tag:      "Pressure",
			Index:    1,
			Channel:  2,
		}}),
		Entry("no variant", v0.Read{}),
	)

	DescribeTable("Should round trip a write",
		func(w v0.Write) {
			var decoded v0.Write
			roundTrip(w, &decoded)
			Expect(decoded).To(Equal(w))
		},
		Entry("plain", v0.Write{Variant: v0.PlainWrite{
			Topic:   "plant/valve",
			Field:   uuid.New(),
			Channel: 3,
		}}),
		Entry("Sparkplug", v0.Write{Variant: v0.SparkplugWrite{
			Group:    "Plant",
			EdgeNode: "Line1",
			Tag:      "Speed",
			Channel:  4,
		}}),
		Entry("no variant", v0.Write{}),
	)

	DescribeTable("Should reject an unknown type",
		func(target any) {
			raw := MustSucceed(msgpack.Marshal(map[string]any{"type": "unknown"}))
			Expect(msgpack.Unmarshal(raw, target)).
				To(MatchError(validate.ErrValidation))
		},
		Entry("transport", &v0.Transport{}),
		Entry("auth", &v0.Auth{}),
		Entry("Sparkplug mode", &v0.SparkplugMode{}),
		Entry("read", &v0.Read{}),
		Entry("write", &v0.Write{}),
	)

	DescribeTable(
		"Should reject a variant whose fields have the wrong type",
		func(target any, fields map[string]any) {
			raw := MustSucceed(msgpack.Marshal(fields))
			Expect(msgpack.Unmarshal(raw, target)).
				To(MatchError(ContainSubstring("msgpack: ")))
		},
		Entry("TLS", &v0.Transport{}, map[string]any{
			"type":                            "tls",
			"certificate_validation_disabled": "yes",
		}),
		Entry(
			"user name",
			&v0.Auth{},
			map[string]any{"type": "username", "username": 1},
		),
		Entry("primary", &v0.SparkplugMode{}, map[string]any{
			"type":    "primary",
			"host_id": 1,
		}),
		Entry("plain read", &v0.Read{}, map[string]any{"type": "plain", "topic": 1}),
		Entry("Sparkplug write", &v0.Write{}, map[string]any{
			"type": "sparkplug",
			"tag":  1,
		}),
		Entry("Sparkplug read", &v0.Read{}, map[string]any{
			"type":  "sparkplug",
			"group": 1,
		}),
		Entry("plain write", &v0.Write{}, map[string]any{"type": "plain", "topic": 1}),
		Entry("transport type", &v0.Transport{}, map[string]any{"type": []int{1}}),
		Entry("auth type", &v0.Auth{}, map[string]any{"type": []int{1}}),
		Entry(
			"Sparkplug mode type",
			&v0.SparkplugMode{},
			map[string]any{"type": []int{1}},
		),
		Entry("read type", &v0.Read{}, map[string]any{"type": []int{1}}),
		Entry("write type", &v0.Write{}, map[string]any{"type": []int{1}}),
	)

	DescribeTable("Should reject truncated input",
		func(target any) {
			Expect(msgpack.Unmarshal([]byte{0x81}, target)).To(MatchError(io.EOF))
		},
		Entry("transport", &v0.Transport{}),
		Entry("auth", &v0.Auth{}),
		Entry("Sparkplug mode", &v0.SparkplugMode{}),
		Entry("read", &v0.Read{}),
		Entry("write", &v0.Write{}),
	)

	It("Should round trip a broker", func() {
		b := v0.Broker{
			Key:       uuid.New(),
			Rack:      1,
			Name:      "Plant Broker",
			Host:      "mqtt.example.com",
			Transport: v0.Transport{Variant: v0.TlsTransport{}},
			Auth:      v0.Auth{Variant: v0.NoneAuth{}},
			KeepAlive: 30 * telem.Second,
			Sparkplug: v0.Sparkplug{
				Mode:   v0.SparkplugMode{Variant: v0.PassiveSparkplugMode{}},
				Groups: []string{"Plant"},
			},
		}
		var decoded v0.Broker
		roundTrip(b, &decoded)
		Expect(decoded).To(Equal(b))
	})
})
