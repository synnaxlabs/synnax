// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package breaker_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/breaker"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Breaker", func() {
	It("By default should not allow retry", func() {
		ctx, cancel := context.WithCancel(context.Background())
		b := MustSucceed(breaker.NewBreaker(ctx))
		Expect(b.Wait()).To(BeFalse())
		cancel()
	})
	It("Should be canceled as the underlying context is canceled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		b := MustSucceed(breaker.NewBreaker(ctx, breaker.Config{BaseInterval: 1 * time.Hour}))
		go func() {
			time.Sleep(500 * time.Millisecond)
			cancel()
		}()
		go func() {
			Expect(b.Wait()).To(BeFalse())
			done <- struct{}{}
		}()
		Eventually(done).Should(Receive())
	})
	It("Should scale the timeout every time it fails", func() {
		b := MustSucceed(breaker.NewBreaker(ctx, breaker.Config{BaseInterval: 10 * time.Millisecond, Scale: 2, MaxRetries: 10}))
		start := time.Now()
		Expect(b.Wait()).To(BeTrue()) // 10ms
		Expect(b.Wait()).To(BeTrue()) // 20ms
		Expect(b.Wait()).To(BeTrue()) // 40ms
		Expect(b.Wait()).To(BeTrue()) // 80ms
		Expect(b.Wait()).To(BeTrue()) // 160ms
		duration := time.Since(start)
		Expect(duration).To(BeNumerically("~", 310*time.Millisecond, 100*time.Millisecond))
	})
})
