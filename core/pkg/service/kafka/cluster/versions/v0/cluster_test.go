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
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/cluster/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Cluster", func() {
	cluster := func() v0.Cluster {
		return v0.Cluster{
			Key:       uuid.MustParse("22222222-2222-4222-8222-222222222222"),
			Rack:      1,
			Name:      "Plant Cluster",
			Brokers:   []v0.Broker{{Host: "kafka.example.com"}},
			Transport: v0.Transport{Variant: v0.PlaintextTransport{}},
			Auth:      v0.Auth{Variant: v0.NoneAuth{}},
		}
	}

	read := func() v0.Read {
		return v0.Read{
			Topic:   "plant.temp",
			Message: uuid.New(),
			Index:   1,
			Channel: 2,
		}
	}

	credentials := func() v0.Credentials {
		return v0.Credentials{Username: "admin", Password: uuid.New()}
	}

	It("Should have a table prefix of its own", func() {
		Expect(types.Name[v0.Cluster]()).To(Equal("kafka_cluster"))
	})

	It("Should identify itself by its key", func() {
		c := cluster()
		Expect(c.GorpKey()).To(Equal(c.Key))
		Expect(c.ResourceID()).To(Equal(resource.ID{
			Type: resource.TypeKafkaCluster,
			Key:  c.Key.String(),
		}))
	})

	It("Should apply the defaults", func() {
		var c v0.Cluster
		Expect(json.Unmarshal([]byte(`{
			"key": "22222222-2222-4222-8222-222222222222",
			"rack": 1,
			"name": "Plant Cluster",
			"brokers": [{"host": "kafka.example.com"}]
		}`), &c)).To(Succeed())
		Expect(c).To(Equal(cluster()))
		Expect(c.Validate()).To(Succeed())
	})

	It("Should round trip a cluster through JSON", func() {
		c := cluster()
		c.Brokers = append(c.Brokers, v0.Broker{
			Host: "10.0.0.2",
			Port: new(uint16(9093)),
		})
		c.Transport = v0.Transport{Variant: v0.TlsTransport{
			CaCertificate:     new(uuid.New()),
			ClientCertificate: new(uuid.New()),
		}}
		c.Auth = v0.Auth{Variant: v0.ScramSha512Auth{Credentials: credentials()}}
		r := read()
		r.RecordKey = new("pump")
		r.Pointer = "/temp"
		c.Reads = []v0.Read{read(), r}
		Expect(c.Validate()).To(Succeed())
		var decoded v0.Cluster
		Expect(json.Unmarshal(MustSucceed(json.Marshal(c)), &decoded)).To(Succeed())
		Expect(decoded).To(Equal(c))
	})

	DescribeTable(
		"Should reject an invalid cluster",
		func(change func(*v0.Cluster), path string) {
			c := cluster()
			change(&c)
			Expect(c.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("no name", func(c *v0.Cluster) { c.Name = "" }, "name"),
		Entry("no brokers", func(c *v0.Cluster) { c.Brokers = nil }, "brokers"),
		Entry("an empty broker list", func(c *v0.Cluster) {
			c.Brokers = []v0.Broker{}
		}, "brokers"),
		Entry("a broker with no host", func(c *v0.Cluster) {
			c.Brokers = []v0.Broker{{}}
		}, "host"),
		Entry("a broker host with a port", func(c *v0.Cluster) {
			c.Brokers = []v0.Broker{{Host: "kafka.example.com:9092"}}
		}, "host"),
		Entry("a broker on port 0", func(c *v0.Cluster) {
			c.Brokers = []v0.Broker{{Host: "kafka.example.com", Port: new(uint16(0))}}
		}, "port"),
		Entry(
			"no transport",
			func(c *v0.Cluster) { c.Transport = v0.Transport{} },
			"type",
		),
		Entry("no auth", func(c *v0.Cluster) { c.Auth = v0.Auth{} }, "mechanism"),
		Entry("a PLAIN auth with no user name", func(c *v0.Cluster) {
			cr := credentials()
			cr.Username = ""
			c.Auth = v0.Auth{Variant: v0.PlainAuth{Credentials: cr}}
		}, "username"),
		Entry("a SCRAM-SHA-256 auth with no password", func(c *v0.Cluster) {
			cr := credentials()
			cr.Password = uuid.Nil()
			c.Auth = v0.Auth{Variant: v0.ScramSha256Auth{Credentials: cr}}
		}, "password"),
		Entry("a read with no topic", func(c *v0.Cluster) {
			r := read()
			r.Topic = ""
			c.Reads = []v0.Read{r}
		}, "topic"),
		Entry("a read topic with a slash", func(c *v0.Cluster) {
			r := read()
			r.Topic = "plant/temp"
			c.Reads = []v0.Read{r}
		}, "topic"),
		Entry("a read topic of one dot", func(c *v0.Cluster) {
			r := read()
			r.Topic = "."
			c.Reads = []v0.Read{r}
		}, "topic"),
		Entry("a read topic of two dots", func(c *v0.Cluster) {
			r := read()
			r.Topic = ".."
			c.Reads = []v0.Read{r}
		}, "topic"),
		Entry("a read topic over 249 characters", func(c *v0.Cluster) {
			r := read()
			r.Topic = strings.Repeat("a", 250)
			c.Reads = []v0.Read{r}
		}, "topic"),
		Entry("a read with an empty record key", func(c *v0.Cluster) {
			r := read()
			r.RecordKey = new("")
			c.Reads = []v0.Read{r}
		}, "record_key"),
		Entry("a read with no message", func(c *v0.Cluster) {
			r := read()
			r.Message = uuid.Nil()
			c.Reads = []v0.Read{r}
		}, "message"),
		Entry("a read pointer with no leading slash", func(c *v0.Cluster) {
			r := read()
			r.Pointer = "temp"
			c.Reads = []v0.Read{r}
		}, "pointer"),
		Entry("a read with no index", func(c *v0.Cluster) {
			r := read()
			r.Index = 0
			c.Reads = []v0.Read{r}
		}, "index"),
		Entry("a read with no channel", func(c *v0.Cluster) {
			r := read()
			r.Channel = 0
			c.Reads = []v0.Read{r}
		}, "channel"),
	)

	DescribeTable("Should accept a valid cluster",
		func(change func(*v0.Cluster)) {
			c := cluster()
			change(&c)
			Expect(c.Validate()).To(Succeed())
		},
		Entry("an IPv4 broker", func(c *v0.Cluster) {
			c.Brokers = []v0.Broker{{Host: "192.168.1.10"}}
		}),
		Entry("an IPv6 broker", func(c *v0.Cluster) {
			c.Brokers = []v0.Broker{{Host: "fe80::1"}}
		}),
		Entry("TLS with no CA", func(c *v0.Cluster) {
			c.Transport = v0.Transport{Variant: v0.TlsTransport{}}
		}),
		Entry("TLS with only a client certificate", func(c *v0.Cluster) {
			c.Transport = v0.Transport{Variant: v0.TlsTransport{
				ClientCertificate: new(uuid.New()),
			}}
		}),
		Entry("a PLAIN auth", func(c *v0.Cluster) {
			c.Auth = v0.Auth{Variant: v0.PlainAuth{Credentials: credentials()}}
		}),
		Entry("a read topic of 249 characters", func(c *v0.Cluster) {
			r := read()
			r.Topic = strings.Repeat("a", 249)
			c.Reads = []v0.Read{r}
		}),
		Entry("a read topic of three dots", func(c *v0.Cluster) {
			r := read()
			r.Topic = "..."
			c.Reads = []v0.Read{r}
		}),
		Entry("a read topic that starts with a dot", func(c *v0.Cluster) {
			r := read()
			r.Topic = ".plant_temp-1"
			c.Reads = []v0.Read{r}
		}),
		Entry("a read of one record key", func(c *v0.Cluster) {
			r := read()
			r.RecordKey = new("pump")
			c.Reads = []v0.Read{r}
		}),
		Entry("a read of the records with no key", func(c *v0.Cluster) {
			c.Reads = []v0.Read{read()}
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
		Entry("plaintext", v0.Transport{Variant: v0.PlaintextTransport{}}),
		Entry("TLS", v0.Transport{Variant: v0.TlsTransport{
			CaCertificate: new(uuid.New()),
		}}),
		Entry("TLS with no CA", v0.Transport{Variant: v0.TlsTransport{}}),
		Entry("no variant", v0.Transport{}),
	)

	credentials := v0.Credentials{Username: "admin", Password: uuid.New()}

	DescribeTable("Should round trip an auth",
		func(a v0.Auth) {
			var decoded v0.Auth
			roundTrip(a, &decoded)
			Expect(decoded).To(Equal(a))
		},
		Entry("none", v0.Auth{Variant: v0.NoneAuth{}}),
		Entry("PLAIN", v0.Auth{Variant: v0.PlainAuth{Credentials: credentials}}),
		Entry("SCRAM-SHA-256", v0.Auth{
			Variant: v0.ScramSha256Auth{Credentials: credentials},
		}),
		Entry("SCRAM-SHA-512", v0.Auth{
			Variant: v0.ScramSha512Auth{Credentials: credentials},
		}),
		Entry("no variant", v0.Auth{}),
	)

	It("Should write the mechanism and credentials at the top level", func() {
		raw := MustSucceed(msgpack.Marshal(v0.Auth{
			Variant: v0.PlainAuth{Credentials: credentials},
		}))
		var fields map[string]any
		Expect(msgpack.Unmarshal(raw, &fields)).To(Succeed())
		Expect(fields).To(HaveKeyWithValue("mechanism", "plain"))
		Expect(fields).To(HaveKeyWithValue("username", "admin"))
	})

	DescribeTable("Should reject an unknown discriminator",
		func(target any, field string) {
			raw := MustSucceed(msgpack.Marshal(map[string]any{field: "unknown"}))
			Expect(msgpack.Unmarshal(raw, target)).
				To(MatchError(validate.ErrValidation))
		},
		Entry("transport", &v0.Transport{}, "type"),
		Entry("auth", &v0.Auth{}, "mechanism"),
	)

	DescribeTable("Should reject a variant whose fields have the wrong type",
		func(target any, fields map[string]any) {
			raw := MustSucceed(msgpack.Marshal(fields))
			Expect(msgpack.Unmarshal(raw, target)).
				To(MatchError(ContainSubstring("msgpack: ")))
		},
		Entry("TLS", &v0.Transport{}, map[string]any{
			"type":           "tls",
			"ca_certificate": 1,
		}),
		Entry("SCRAM", &v0.Auth{}, map[string]any{
			"mechanism": "scram_sha_512",
			"username":  1,
		}),
		Entry("plaintext", &v0.Transport{}, map[string]any{"type": []int{1}}),
		Entry("none", &v0.Auth{}, map[string]any{"mechanism": []int{1}}),
	)

	DescribeTable("Should reject truncated input",
		func(target any) {
			Expect(msgpack.Unmarshal([]byte{0x81}, target)).To(MatchError(io.EOF))
		},
		Entry("transport", &v0.Transport{}),
		Entry("auth", &v0.Auth{}),
	)

	It("Should round trip a cluster", func() {
		c := v0.Cluster{
			Key:  uuid.New(),
			Rack: 1,
			Name: "Plant Cluster",
			Brokers: []v0.Broker{
				{Host: "kafka.example.com", Port: new(uint16(9093))},
			},
			Transport: v0.Transport{Variant: v0.TlsTransport{}},
			Auth:      v0.Auth{Variant: v0.ScramSha256Auth{Credentials: credentials}},
			Reads: []v0.Read{{
				Topic:     "plant",
				RecordKey: new("pump"),
				Message:   uuid.New(),
				Pointer:   "/temp",
				Index:     1,
				Channel:   2,
			}},
		}
		var decoded v0.Cluster
		roundTrip(c, &decoded)
		Expect(decoded).To(Equal(c))
	})
})
