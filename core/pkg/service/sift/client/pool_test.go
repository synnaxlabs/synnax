// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package client_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Pool", func() {
	var (
		ctx     context.Context
		factory client.Factory
	)

	BeforeEach(func() {
		ctx = context.Background()
		factory = MustSucceed(client.NewMockFactory())
	})

	It("Should return the same client for the same URI", func() {
		pool := client.NewPool(factory)
		client1 := MustSucceed(pool.Get(ctx, "uri1", "key1"))
		Expect(pool.Get(ctx, "uri1", "key2")).To(BeIdenticalTo(client1))
	})

	It("Should return an error when the factory returns an error", func() {
		errTest := errors.New("test error")
		factory := func(context.Context, string, string) (client.Client, error) {
			return nil, errTest
		}
		pool := client.NewPool(factory)
		Expect(pool.Get(ctx, "uri1", "key1")).Error().To(MatchError(errTest))
	})

	It("Should return different clients for different URIs", func() {
		var callCount int
		factory := func(
			ctx context.Context,
			uri,
			apiKey string,
		) (client.Client, error) {
			callCount++
			return factory(ctx, uri, apiKey)
		}
		pool := client.NewPool(factory)
		client1 := MustSucceed(pool.Get(ctx, "uri1", "key1"))
		Expect(pool.Get(ctx, "uri2", "key1")).ToNot(BeIdenticalTo(client1))
		Expect(callCount).To(Equal(2))
	})

	It("Should close all clients on Close", func() {
		var (
			closed1, closed2 bool
			idx              int
		)
		closeFuncs := []func() error{
			func() error { closed1 = true; return nil },
			func() error { closed2 = true; return nil },
		}
		factory := MustSucceed(client.NewMockFactory(client.MockFactoryConfig{
			OnClose: func() error {
				if err := closeFuncs[idx](); err != nil {
					return err
				}
				idx++
				return nil
			},
		}))

		pool := client.NewPool(factory)

		client1 := MustSucceed(pool.Get(ctx, "uri1", "key1"))
		Expect(pool.Get(ctx, "uri2", "key1")).ToNot(BeIdenticalTo(client1))
		Expect(pool.Close()).To(Succeed())
		Expect(closed1).To(BeTrue())
		Expect(closed2).To(BeTrue())
	})
})
