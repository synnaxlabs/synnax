// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	distwriter "github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/transport/mock/framer/writer"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	leaseholder = address.Address("leaseholder")
	gateway     = address.Address("gateway")
)

var _ = Describe("Transport", func() {
	var (
		net    *writer.Network
		server distwriter.Transport
		client distwriter.Transport
	)
	BeforeEach(func() {
		net = writer.NewNetwork()
		server = net.New(leaseholder, 1)
		client = net.New(gateway, 1)
	})

	It("Should round-trip a request through the streaming transport", func(ctx SpecContext) {
		server.Server().BindHandler(func(
			_ context.Context,
			srv freighter.ServerStream[distwriter.Request, distwriter.Response],
		) error {
			req, err := srv.Receive()
			if err != nil {
				return err
			}
			return srv.Send(distwriter.Response{SeqNum: req.SeqNum})
		})
		stream := MustSucceed(client.Client().Stream(ctx, leaseholder))
		Expect(stream.Send(distwriter.Request{
			SeqNum:  42,
			Command: distwriter.CommandWrite,
		})).To(Succeed())
		Expect(MustSucceed(stream.Receive()).SeqNum).To(Equal(42))
		Expect(stream.CloseSend()).To(Succeed())
	})
})
