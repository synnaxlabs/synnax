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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/net/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Address", func() {
	It("Should leave an absent port absent", func() {
		var a v0.Address
		Expect(json.Unmarshal([]byte(`{"host":"kafka.example.com"}`), &a)).
			To(Succeed())
		Expect(a).To(Equal(v0.Address{Host: "kafka.example.com"}))
		Expect(a.Validate()).To(Succeed())
	})

	DescribeTable("Should round trip an address",
		func(a v0.Address) {
			var fromJSON, fromMsgpack v0.Address
			Expect(json.Unmarshal(MustSucceed(json.Marshal(a)), &fromJSON)).
				To(Succeed())
			Expect(fromJSON).To(Equal(a))
			Expect(msgpack.Unmarshal(MustSucceed(msgpack.Marshal(a)), &fromMsgpack)).
				To(Succeed())
			Expect(fromMsgpack).To(Equal(a))
		},
		Entry("with no port", v0.Address{Host: "plc.example.com"}),
		Entry("with a port", v0.Address{Host: "10.0.0.2", Port: new(uint16(502))}),
	)

	DescribeTable("Should accept a valid host",
		func(host string) {
			Expect(v0.Address{Host: host}.Validate()).To(Succeed())
		},
		Entry("a hostname", "kafka.example.com"),
		Entry("a single label", "localhost"),
		Entry("an IPv4 address", "192.168.1.10"),
		Entry("an IPv6 address", "fe80::1"),
		Entry("a full IPv6 address", "2001:0db8:0000:0000:0000:ff00:0042:8329"),
	)

	DescribeTable(
		"Should reject an invalid address",
		func(a v0.Address, path string) {
			Expect(a.Validate()).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(path)),
			))
		},
		Entry("no host", v0.Address{}, "host"),
		Entry(
			"a host with a scheme",
			v0.Address{Host: "tcp://plc.example.com"},
			"host",
		),
		Entry("a host with a port", v0.Address{Host: "plc.example.com:502"}, "host"),
		Entry("a host with a path", v0.Address{Host: "plc.example.com/a"}, "host"),
		Entry("port 0", v0.Address{Host: "plc", Port: new(uint16(0))}, "port"),
	)
})
