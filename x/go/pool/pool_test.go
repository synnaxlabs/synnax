// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pool_test

import (
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/pool"
	. "github.com/synnaxlabs/x/testutil"
)

// fakeAdapter is a hand-rolled pool.Adapter used by the tests below. The
// closeBlock channel lets a test gate when Close is allowed to return so we
// can exercise concurrent Close + Acquire ordering.
type fakeAdapter struct {
	id         int
	healthy    bool
	closed     atomic.Bool
	closeErr   error
	closeBlock chan struct{}
}

func (a *fakeAdapter) Healthy() bool  { return a.healthy }
func (a *fakeAdapter) Acquire() error { return nil }
func (a *fakeAdapter) Release()       {}
func (a *fakeAdapter) Close() error {
	if a.closeBlock != nil {
		<-a.closeBlock
	}
	a.closed.Store(true)
	return a.closeErr
}

type fakeFactory struct {
	mu         sync.Mutex
	nextID     int
	healthy    bool
	closeErr   error
	closeBlock chan struct{}
	created    []*fakeAdapter
	failNext   bool
}

func (f *fakeFactory) New(_ string) (*fakeAdapter, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return nil, errors.New("factory: failed")
	}
	a := &fakeAdapter{
		id:         f.nextID,
		healthy:    f.healthy,
		closeErr:   f.closeErr,
		closeBlock: f.closeBlock,
	}
	f.nextID++
	f.created = append(f.created, a)
	return a, nil
}

func newFactory() *fakeFactory { return &fakeFactory{healthy: true} }

var _ = Describe("Pool", func() {
	Describe("Acquire", func() {
		It("returns the same adapter for the same key when it stays healthy", func() {
			f := newFactory()
			p := pool.New(f)
			defer func() { Expect(p.Close()).To(Succeed()) }()

			a1 := MustSucceed(p.Acquire("k"))
			a2 := MustSucceed(p.Acquire("k"))
			Expect(a2).To(BeIdenticalTo(a1))
			Expect(f.created).To(HaveLen(1))
		})

		It("creates a new adapter when the cached one is unhealthy", func() {
			f := newFactory()
			p := pool.New(f)
			defer func() { Expect(p.Close()).To(Succeed()) }()

			a1 := MustSucceed(p.Acquire("k"))
			a1.healthy = false

			a2 := MustSucceed(p.Acquire("k"))
			Expect(a2).ToNot(BeIdenticalTo(a1))
			Expect(f.created).To(HaveLen(2))
		})

		It("propagates factory errors", func() {
			f := newFactory()
			f.failNext = true
			p := pool.New(f)
			defer func() { Expect(p.Close()).To(Succeed()) }()

			_, err := p.Acquire("k")
			Expect(err).To(MatchError(ContainSubstring("factory: failed")))
		})

		It("serializes concurrent Acquire calls so the factory only runs once per key", func() {
			f := newFactory()
			p := pool.New(f)
			defer func() { Expect(p.Close()).To(Succeed()) }()

			const goroutines = 32
			var (
				wg      sync.WaitGroup
				mu      sync.Mutex
				results []*fakeAdapter
			)
			start := make(chan struct{})
			for range goroutines {
				wg.Go(func() {
					<-start
					a, err := p.Acquire("shared")
					if err != nil {
						return
					}
					mu.Lock()
					results = append(results, a)
					mu.Unlock()
				})
			}
			close(start)
			wg.Wait()

			Expect(results).To(HaveLen(goroutines))
			Expect(f.created).To(HaveLen(1))
			for _, a := range results {
				Expect(a).To(BeIdenticalTo(f.created[0]))
			}
		})
	})

	Describe("Close", func() {
		It("closes every cached adapter", func() {
			f := newFactory()
			p := pool.New(f)
			MustSucceed(p.Acquire("a"))
			MustSucceed(p.Acquire("b"))

			Expect(p.Close()).To(Succeed())
			for _, a := range f.created {
				Expect(a.closed.Load()).To(BeTrue())
			}
		})

		It("aggregates errors from adapter closes", func() {
			f := newFactory()
			f.closeErr = errors.New("adapter: boom")
			p := pool.New(f)
			MustSucceed(p.Acquire("a"))
			MustSucceed(p.Acquire("b"))

			err := p.Close()
			Expect(err).To(MatchError(ContainSubstring("adapter: boom")))
		})

		It("is idempotent", func() {
			f := newFactory()
			p := pool.New(f)
			MustSucceed(p.Acquire("a"))

			Expect(p.Close()).To(Succeed())
			Expect(p.Close()).To(Succeed())
		})

		It("does not deadlock when an adapter's Close blocks", func() {
			// Regression test for the original implementation, which held
			// the pool mutex across adapter.Close. With that bug, the
			// concurrent Acquire below would block indefinitely waiting on
			// the lock instead of immediately returning ErrClosed.
			f := newFactory()
			f.closeBlock = make(chan struct{})
			p := pool.New(f)
			MustSucceed(p.Acquire("a"))

			closeReturned := make(chan error, 1)
			go func() { closeReturned <- p.Close() }()

			// Wait long enough that Close has reached adapter.Close and is
			// waiting on closeBlock. Then verify a concurrent Acquire is
			// not blocked by the lock — it should observe the closed pool
			// and return ErrClosed straight away.
			Eventually(func() error {
				_, err := p.Acquire("a")
				return err
			}).WithTimeout(500 * time.Millisecond).
				Should(MatchError(pool.ErrClosed))

			close(f.closeBlock)
			Eventually(closeReturned).Should(Receive(BeNil()))
		})
	})

	Describe("Acquire after Close", func() {
		It("returns ErrClosed", func() {
			f := newFactory()
			p := pool.New(f)
			Expect(p.Close()).To(Succeed())

			_, err := p.Acquire("a")
			Expect(err).To(MatchError(pool.ErrClosed))
			Expect(errors.Is(err, pool.ErrClosed)).To(BeTrue())
		})
	})
})
