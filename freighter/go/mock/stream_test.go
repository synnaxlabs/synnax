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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Stream", Ordered, Serial, func() {
	var (
		server *mock.StreamServer[test.Request, test.Response]
		client *mock.StreamClient[test.Request, test.Response]
	)

	BeforeAll(func() {
		ShouldNotLeakGoroutines()
		server, client = mock.NewStreamPair[test.Request, test.Response](11, 11)
	})

	test.StreamSuite(func() (
		freighter.StreamServer[test.Request, test.Response],
		freighter.StreamClient[test.Request, test.Response],
		address.Address,
	) {
		return server, client, "localhost:0"
	})

	It("Should release the handler when the client stops receiving", func(
		ctx SpecContext,
	) {
		ShouldNotLeakGoroutines()
		abandoned, dialer := mock.NewStreamPair[test.Request, test.Response](1, 1)
		returned := make(chan struct{})
		abandoned.BindHandler(func(
			_ context.Context,
			stream freighter.ServerStream[test.Request, test.Response],
		) error {
			defer close(returned)
			if err := stream.Send(test.Response{ID: 1}); err != nil {
				return err
			}
			return stream.Send(test.Response{ID: 2})
		})
		streamCtx, cancel := context.WithCancel(ctx)
		stream := MustSucceed(dialer.Stream(streamCtx, "localhost:0"))
		Expect(stream.Receive()).To(Equal(test.Response{ID: 1}))
		// The client leaves the second response in the buffer, so the closing error
		// the handler emits on return has nowhere to go until the client cancels.
		Eventually(returned).Should(BeClosed())
		cancel()
	})
})
