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
	"sync/atomic"

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

var _ = Describe("Budget", func() {
	DescribeTable(
		"Should refuse a budget with no limit, no width, or no executor",
		func(limit, width int, exec segment.Executor, field string) {
			Expect(segment.NewBudget(limit, width, exec)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(field)),
			))
		},
		Entry("no limit", 0, 32, segment.Executor(segment.GoExecutor), "limit"),
		Entry("no width", 4, 0, segment.Executor(segment.GoExecutor), "width"),
		Entry("no executor", 4, 32, nil, "executor"),
	)

	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var fs *opFS
			BeforeEach(func() { fs = newOpFS(makeFS()) })

			// dirty returns n handles of b with one write each since their last sync.
			dirty := func(b *segment.Budget, n int, prefix string) []*segment.Handle {
				GinkgoHelper()
				hs := make([]*segment.Handle, n)
				for i := range hs {
					name := fmt.Sprintf("%s%d.seg", prefix, i)
					hs[i] = MustSucceed(b.Create(fs, name))
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
				b := MustSucceed(segment.NewBudget(16, 2, segment.GoExecutor))
				first, second := dirty(b, 3, "a"), dirty(b, 3, "b")
				release := fs.Hold()
				done := make(chan error, 2)
				go func() { done <- b.Sync(first) }()
				go func() { done <- b.Sync(second) }()
				Eventually(fs.held).Should(Receive())
				Eventually(fs.held).Should(Receive())
				Consistently(fs.held).ShouldNot(Receive())
				fs.Unhold()
				close(release)
				Eventually(done).Should(Receive(BeNil()))
				Eventually(done).Should(Receive(BeNil()))
				Expect(fs.MaxSyncs()).To(Equal(2))
				for _, h := range slices.Concat(first, second) {
					Expect(h.Sync()).To(Succeed())
				}
				Expect(fs.Ops()).To(HaveLen(6*3), "a synced handle syncs once")
			})

			It(
				"Should return each failed sync, in order, and still sync the rest",
				func() {
					b := MustSucceed(segment.NewBudget(16, 32, segment.GoExecutor))
					hs := dirty(b, 4, "")
					fs.FailSync("1.seg", errInjected)
					fs.FailSync("3.seg", errInjected)
					err := b.Sync(hs)
					Expect(err).To(MatchError(errInjected))
					Expect(err.Error()).To(MatchRegexp(`(?s)file 1\.seg.*file 3\.seg`))
					Expect(fs.Ops()).To(ContainElements("sync 0.seg", "sync 2.seg"))
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
					b := MustSucceed(segment.NewBudget(16, 32, q.exec))
					hs := dirty(b, 3, "")
					done := make(chan error, 1)
					go func() { done <- b.Sync(hs) }()
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

			It("Should run the sync before a close on its executor, inside its width",
				func() {
					var execs atomic.Int64
					b := MustSucceed(segment.NewBudget(2, 1, func(f func()) {
						execs.Add(1)
						go f()
					}))
					hs := dirty(b, 2, "")
					release := fs.Hold()
					synced := make(chan error, 1)
					go func() { synced <- b.Sync(hs[:1]) }()
					Eventually(fs.held).Should(Receive(Equal("0.seg")))
					created := make(chan *segment.Handle, 1)
					go func() { created <- MustSucceed(b.Create(fs, "2.seg")) }()
					Eventually(execs.Load).Should(Equal(int64(2)))
					Consistently(fs.held).ShouldNot(Receive())
					fs.Unhold()
					close(release)
					Eventually(synced).Should(Receive(BeNil()))
					var third *segment.Handle
					Eventually(created).Should(Receive(&third))
					Expect(third.Close()).To(Succeed())
					Expect(fs.MaxSyncs()).To(Equal(1))
					ops := fs.Ops()
					Expect(slices.Index(ops, "sync 1.seg")).
						To(BeNumerically("<", slices.Index(ops, "close 1.seg")))
				},
			)

			It(
				"Should sync a handle that waits for its close without a slot of "+
					"its own",
				func() {
					b := MustSucceed(segment.NewBudget(1, 1, segment.GoExecutor))
					hs := dirty(b, 1, "")
					release := fs.Hold()
					created := make(chan *segment.Handle, 1)
					go func() { created <- MustSucceed(b.Create(fs, "1.seg")) }()
					Eventually(fs.held).Should(Receive(Equal("0.seg")))
					synced := make(chan error, 1)
					go func() { synced <- b.Sync(hs) }()
					Consistently(synced).ShouldNot(Receive())
					fs.Unhold()
					close(release)
					Eventually(synced).Should(Receive(BeNil()))
					var second *segment.Handle
					Eventually(created).Should(Receive(&second))
					Expect(second.Close()).To(Succeed())
					opens := 0
					for _, op := range fs.Ops() {
						if op == "open 0.seg" {
							opens++
						}
					}
					Expect(opens).To(Equal(1), "a clean closed file opens for no sync")
				},
			)
		})
	}
})
