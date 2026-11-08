// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package coalesce_test

import (
	"context"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/coalesce"
	"github.com/synnaxlabs/x/set"
	. "github.com/synnaxlabs/x/testutil"
)

// recorder records the keys of each call of Handle.
type recorder struct {
	mu    sync.Mutex
	calls []set.Set[int]
}

func (r *recorder) handle(_ context.Context, keys set.Set[int]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, keys)
}

func (r *recorder) handled() set.Set[int] {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make(set.Set[int])
	for _, c := range r.calls {
		all.Add(c.Slice()...)
	}
	return all
}

var _ = Describe("Worker", func() {
	It("Should fail without a Handle function", func() {
		Expect(coalesce.Open[int]()).Error().To(MatchError(ContainSubstring("handle")))
	})

	It("Should handle every key that Add receives from many goroutines", func() {
		rec := &recorder{}
		w := DeferClose(
			MustOpen(coalesce.Open(coalesce.Config[int]{Handle: rec.handle})),
		)
		var wg sync.WaitGroup
		for g := range 8 {
			wg.Go(func() {
				for i := range 100 {
					w.Add(g*100 + i)
				}
			})
		}
		wg.Wait()
		want := make(set.Set[int])
		for k := range 800 {
			want.Add(k)
		}
		Eventually(rec.handled).Should(Equal(want))
	})

	It("Should hand a key added during a call of Handle to a later call", func() {
		var (
			started = make(chan struct{})
			release = make(chan struct{})
			calls   = make(chan set.Set[int], 2)
		)
		w := DeferClose(MustOpen(coalesce.Open(coalesce.Config[int]{
			Handle: func(_ context.Context, keys set.Set[int]) {
				if keys.Contains(1) {
					close(started)
					<-release
				}
				calls <- keys
			},
		})))
		w.Add(1)
		Eventually(started).Should(BeClosed())
		w.Add(2, 3, 2)
		close(release)
		Eventually(calls).Should(Receive(Equal(set.New(1))))
		Eventually(calls).Should(Receive(Equal(set.New(2, 3))))
	})

	It("Should end the context of a running Handle when Close runs", func() {
		started := make(chan struct{})
		w := MustOpen(coalesce.Open(coalesce.Config[int]{
			Handle: func(ctx context.Context, _ set.Set[int]) {
				close(started)
				<-ctx.Done()
			},
		}))
		w.Add(1)
		Eventually(started).Should(BeClosed())
		Expect(w.Close()).To(Succeed())
	})

	It("Should not wait in Add while Handle runs", func() {
		release := make(chan struct{})
		w := DeferClose(MustOpen(coalesce.Open(coalesce.Config[int]{
			Handle: func(ctx context.Context, _ set.Set[int]) {
				select {
				case <-release:
				case <-ctx.Done():
				}
			},
		})))
		DeferCleanup(func() { close(release) })
		done := make(chan struct{})
		go func() {
			defer close(done)
			for i := range 1000 {
				w.Add(i)
			}
		}()
		Eventually(done).Should(BeClosed())
	})
})
