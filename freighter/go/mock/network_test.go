// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/freighter/test"
)

var _ = Describe("Network", func() {
	Describe("Report", func() {
		mockReport := alamos.Report{
			"protocol":  "golang-mock",
			"encodings": []string{"in-memory"},
		}

		DescribeTable(
			"Should describe the mock protocol for every transport it builds",
			func(build func(*mock.Network[test.Request, test.Response]) freighter.Transport) {
				Expect(build(mock.NewNetwork[test.Request, test.Response]()).Report()).
					To(Equal(mockReport))
			},
			Entry("unary server", func(
				n *mock.Network[test.Request, test.Response],
			) freighter.Transport {
				return n.UnaryServer("")
			}),
			Entry("unary client", func(
				n *mock.Network[test.Request, test.Response],
			) freighter.Transport {
				return n.UnaryClient()
			}),
			Entry("stream server", func(
				n *mock.Network[test.Request, test.Response],
			) freighter.Transport {
				return n.StreamServer("")
			}),
			Entry("stream client", func(
				n *mock.Network[test.Request, test.Response],
			) freighter.Transport {
				return n.StreamClient()
			}),
		)

		It("Should describe the mock protocol for a directly linked pair", func() {
			server, client := mock.NewStreamPair[test.Request, test.Response]()
			Expect(server.Report()).To(Equal(mockReport))
			Expect(client.Report()).To(Equal(mockReport))
		})
	})
})
