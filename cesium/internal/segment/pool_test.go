// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment_test

import (
	"fmt"
	"slices"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// queue is an executor that holds each function until the test runs it.
type queue struct {
	mu    sync.Mutex
	funcs []func()
}

func (q *queue) exec(f func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.funcs = append(q.funcs, f)
}

func (q *queue) len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.funcs)
}

// take returns the held functions and holds no more of them.
func (q *queue) take() []func() {
	q.mu.Lock()
	defer q.mu.Unlock()
	funcs := q.funcs
	q.funcs = nil
	return funcs
}

var _ = Describe("Pool", func() {
	DescribeTable(
		"Should refuse a pool with no width or no executor",
		func(width int, exec segment.Executor) {
			Expect(segment.NewPool(width, exec)).Error().
				To(MatchError(validate.ErrValidation))
		},
		Entry("no width", 0, segment.Executor(segment.GoExecutor)),
		Entry("no executor", 32, nil),
	)

	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var (
				fs     *opFS
				budget *segment.Budget
			)
			BeforeEach(func() {
				fs = newOpFS(makeFS())
				budget = MustSucceed(segment.NewBudget(16))
			})

			// dirty returns n handles with one write each since their last sync.
			dirty := func(n int, prefix string) []*segment.Handle {
				GinkgoHelper()
				hs := make([]*segment.Handle, n)
				for i := range hs {
					name := fmt.Sprintf("%s%d.seg", prefix, i)
					hs[i] = MustSucceed(budget.Create(fs, name))
					Expect(hs[i].Sync()).To(Succeed())
					Expect(hs[i].WriteAt([]byte(name), 0)).To(Equal(len(name)))
				}
				DeferCleanup(func() {
					for _, h := range hs {
						Expect(h.Close()).To(Succeed())
					}
				})
				return hs
			}

			It("Should run at most its width of syncs at once across callers", func() {
				pool := MustSucceed(segment.NewPool(2, segment.GoExecutor))
				a, b := dirty(3, "a"), dirty(3, "b")
				release := fs.Hold()
				done := make(chan error, 2)
				go func() { done <- pool.Sync(a) }()
				go func() { done <- pool.Sync(b) }()
				Eventually(fs.held).Should(Receive())
				Eventually(fs.held).Should(Receive())
				Consistently(fs.held).ShouldNot(Receive())
				fs.Unhold()
				close(release)
				Expect(<-done).To(Succeed())
				Expect(<-done).To(Succeed())
				Expect(fs.MaxSyncs()).To(Equal(2))
				for _, h := range slices.Concat(a, b) {
					Expect(h.Sync()).To(Succeed())
				}
				Expect(fs.Ops()).To(HaveLen(6*3), "a synced handle syncs once")
			})

			It(
				"Should return each failed sync, in order, and still sync the rest",
				func() {
					pool := MustSucceed(segment.NewPool(32, segment.GoExecutor))
					hs := dirty(4, "")
					fs.FailSync("1.seg", errInjected)
					fs.FailSync("3.seg", errInjected)
					err := pool.Sync(hs)
					Expect(err).To(MatchError(errInjected))
					Expect(err.Error()).To(MatchRegexp(`(?s)file 1\.seg.*file 3\.seg`))
					ops := fs.Ops()
					Expect(ops).To(ContainElements("sync 0.seg", "sync 2.seg"))
					Expect(
						hs[1].WriteAt([]byte("x"), 0),
					).Error().
						To(MatchError(errInjected))
					Expect(hs[2].WriteAt([]byte("x"), 0)).To(Equal(1))
				},
			)

			It(
				"Should run every sync on its executor and return after the last",
				func() {
					q := &queue{}
					pool := MustSucceed(segment.NewPool(32, q.exec))
					hs := dirty(3, "")
					done := make(chan error, 1)
					go func() { done <- pool.Sync(hs) }()
					Eventually(q.len).Should(Equal(3))
					funcs := q.take()
					for _, f := range slices.Backward(funcs[1:]) {
						f()
					}
					Consistently(done).ShouldNot(Receive())
					funcs[0]()
					Eventually(done).Should(Receive(BeNil()))
					Expect(fs.Ops()[6:]).To(Equal([]string{
						"sync 2.seg", "sync 1.seg", "sync 0.seg",
					}))
				},
			)
		})
	}
})
