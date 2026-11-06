// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator_test

import (
	"context"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

var _ = Describe("Pipeline", func() {
	errRead := errors.New("read failed")

	// open flows a StreamIterator whose routine runs f. Closing the output stream on
	// exit matches the contract of a real StreamIterator.
	open := func(
		f func(
			context.Context,
			confluence.Outlet[iterator.Request],
			confluence.Inlet[iterator.Response],
		) error,
	) *iterator.Pipeline {
		GinkgoHelper()
		sCtx, cancel := signal.Isolated()
		DeferCleanup(cancel)
		req := confluence.NewStream[iterator.Request]()
		res := confluence.NewStream[iterator.Response]()
		res.Acquire(1)
		sCtx.Go(func(ctx context.Context) error {
			defer res.Close()
			defer GinkgoRecover()
			return f(ctx, req, res)
		})
		return &iterator.Pipeline{
			Requests:  req,
			Responses: res,
			Shutdown:  cancel,
			WaitGroup: sCtx,
		}
	}

	// expectReturns fails the spec with every goroutine's stack if f stalls.
	expectReturns := func(f func()) {
		GinkgoHelper()
		done := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			f()
		}()
		Eventually(done).WithTimeout(10*time.Second).Should(BeClosed(), func() string {
			stacks := make([]byte, 1<<20)
			return string(stacks[:runtime.Stack(stacks, true)])
		})
	}

	It("Should return the ack of a request and collect its data", func() {
		p := open(func(
			_ context.Context,
			in confluence.Outlet[iterator.Request],
			out confluence.Inlet[iterator.Response],
		) error {
			var req iterator.Request
			Eventually(in.Outlet()).Should(Receive(&req))
			Eventually(out.Inlet()).Should(BeSent(iterator.Response{
				Variant: iterator.ResponseVariantData,
				SeqNum:  req.SeqNum,
			}))
			Eventually(out.Inlet()).Should(BeSent(iterator.Response{
				Variant: iterator.ResponseVariantAck,
				Ack:     true,
				SeqNum:  req.SeqNum,
			}))
			Eventually(in.Outlet()).Should(BeClosed())
			return nil
		})
		var value []iterator.Response
		expectReturns(func() {
			Expect(p.Exec(iterator.Request{SeqNum: 3}, &value)).To(BeTrue())
			Expect(value).To(HaveLen(1))
			Expect(value[0].SeqNum).To(Equal(3))
			Expect(p.Close()).To(Succeed())
		})
	})

	It("Should return the error of an exit before the request is read", func() {
		exited := make(chan struct{})
		p := open(func(
			context.Context,
			confluence.Outlet[iterator.Request],
			confluence.Inlet[iterator.Response],
		) error {
			defer close(exited)
			return errRead
		})
		Eventually(exited).Should(BeClosed())
		var value []iterator.Response
		expectReturns(func() {
			Expect(p.Exec(iterator.Request{}, &value)).Error().To(MatchError(errRead))
			Expect(p.Exec(iterator.Request{}, &value)).Error().To(MatchError(errRead))
			Expect(p.Close()).To(MatchError(errRead))
			Expect(p.Close()).To(Succeed())
			Expect(p.Exec(iterator.Request{}, &value)).Error().
				To(MatchError(iterator.ErrClosed))
		})
	})

	It("Should return the error of an exit after the request is read", func() {
		p := open(func(
			_ context.Context,
			in confluence.Outlet[iterator.Request],
			_ confluence.Inlet[iterator.Response],
		) error {
			Eventually(in.Outlet()).Should(Receive())
			return errRead
		})
		var value []iterator.Response
		expectReturns(func() {
			Expect(p.Exec(iterator.Request{}, &value)).Error().To(MatchError(errRead))
			Expect(p.Close()).To(MatchError(errRead))
		})
	})

	It("Should return an error for an exit that reports no error", func() {
		p := open(func(
			context.Context,
			confluence.Outlet[iterator.Request],
			confluence.Inlet[iterator.Response],
		) error {
			return nil
		})
		var value []iterator.Response
		expectReturns(func() {
			Expect(p.Exec(iterator.Request{}, &value)).Error().To(SatisfyAll(
				MatchError(iterator.ErrClosed),
				MatchError(ContainSubstring("without a response")),
			))
			Expect(p.Close()).To(MatchError(iterator.ErrClosed))
		})
	})

	It("Should return when a routine outlives the output stream", func() {
		p := open(func(
			_ context.Context,
			in confluence.Outlet[iterator.Request],
			out confluence.Inlet[iterator.Response],
		) error {
			out.Close()
			Eventually(in.Outlet()).Should(BeClosed())
			return errRead
		})
		var value []iterator.Response
		expectReturns(func() {
			Expect(p.Exec(iterator.Request{}, &value)).Error().To(MatchError(errRead))
			Expect(p.Close()).To(MatchError(errRead))
		})
	})
})
