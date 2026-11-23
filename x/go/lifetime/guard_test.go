// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lifetime_test

import (
	"context"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/lifetime"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Guard", func() {
	var g *lifetime.Guard
	BeforeEach(func() { g = &lifetime.Guard{} })

	Describe("Begin", func() {
		It("Should succeed on the first call", func() {
			Expect(g.Begin()).To(Succeed())
		})

		It("Should fail while the component runs", func() {
			Expect(g.Begin()).To(Succeed())
			Expect(g.Begin()).To(MatchError(ContainSubstring("run called twice")))
		})

		It("Should fail after End", func() {
			Expect(g.Begin()).To(Succeed())
			g.End()
			Expect(g.Begin()).To(MatchError(ContainSubstring("run called twice")))
		})

		It("Should fail after Close", func() {
			Expect(MustSucceed(g.Close())).To(BeTrue())
			Expect(g.Begin()).To(MatchError(ContainSubstring("run called after close")))
		})

		It("Should never wrap context.Canceled", func() {
			Expect(g.Begin()).To(Succeed())
			Expect(g.Begin()).ToNot(MatchError(context.Canceled))
		})
	})

	Describe("End", func() {
		It("Should panic without Begin", func() {
			Expect(g.End).To(PanicWith("lifetime: End called without Begin"))
		})

		It("Should panic on a second call", func() {
			Expect(g.Begin()).To(Succeed())
			g.End()
			Expect(g.End).To(PanicWith("lifetime: End called without Begin"))
		})
	})

	Describe("Close", func() {
		It("Should report the first close before Begin", func() {
			Expect(MustSucceed(g.Close())).To(BeTrue())
		})

		It("Should report the first close after End", func() {
			Expect(g.Begin()).To(Succeed())
			g.End()
			Expect(MustSucceed(g.Close())).To(BeTrue())
		})

		It("Should report later closes as not the first", func() {
			Expect(MustSucceed(g.Close())).To(BeTrue())
			Expect(MustSucceed(g.Close())).To(BeFalse())
		})

		It("Should fail while the component runs, and mark nothing", func() {
			Expect(g.Begin()).To(Succeed())
			Expect(g.Close()).Error().To(SatisfyAll(
				MatchError(ContainSubstring("close called while Run runs")),
				Not(MatchError(context.Canceled)),
			))
			g.End()
			Expect(MustSucceed(g.Close())).To(BeTrue())
		})
	})

	It("Should let exactly one of concurrent Begin and Close calls succeed", func() {
		const n = 50
		var (
			begun  atomic.Int32
			closed atomic.Int32
			wg     sync.WaitGroup
		)
		for range n {
			wg.Go(func() {
				if g.Begin() == nil {
					begun.Add(1)
				}
			})
			wg.Go(func() {
				if first, err := g.Close(); err == nil && first {
					closed.Add(1)
				}
			})
		}
		wg.Wait()
		Expect(begun.Load() + closed.Load()).To(Equal(int32(1)))
	})
})
