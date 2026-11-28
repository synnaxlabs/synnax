// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package udp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/udp"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Properties", func() {
	Describe("CheckLocal", func() {
		DescribeTable("Should accept a local address and port that a sender can reach",
			func(p udp.Properties) { Expect(p.CheckLocal()).To(Succeed()) },
			Entry("all interfaces", udp.Properties{LocalPort: 5032}),
			Entry("an IPv4 address", udp.Properties{
				LocalAddress: "127.0.0.1",
				LocalPort:    5032,
			}),
			Entry("an IPv6 address with a zone", udp.Properties{
				LocalAddress: "fe80::1%eth0",
				LocalPort:    5032,
			}),
			Entry("port 0 with a remote", udp.Properties{
				Remote: &udp.Remote{Host: "10.0.0.7", Port: 5031},
			}),
		)

		DescribeTable(
			"Should refuse each bad value at its field",
			func(p udp.Properties, path, message string) {
				err := p.CheckLocal()
				Expect(err).To(MatchError(validate.ErrValidation))
				Expect(err).To(MatchError(ContainSubstring(path + ": " + message)))
			},
			Entry(
				"a host name",
				udp.Properties{LocalAddress: "localhost", LocalPort: 5032},
				"local_address",
				"must be an IP address",
			),
			Entry("an address with spaces", udp.Properties{
				LocalAddress: " 127.0.0.1",
				LocalPort:    5032,
			}, "local_address", "must be an IP address"),
			Entry("port 0 with no remote", udp.Properties{LocalAddress: "127.0.0.1"},
				"local_port", "required when the device has no remote"),
		)
	})
})
