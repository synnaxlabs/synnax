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
	"maps"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/coalesce"
	. "github.com/synnaxlabs/x/testutil"
)

// recorder records the values of each call of Handle.
type recorder struct {
	mu    sync.Mutex
	calls []map[int]string
}

func (r *recorder) handle(_ context.Context, latest map[int]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, latest)
}

// handled returns the last value that a call of Handle got for each key.
func (r *recorder) handled() map[int]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	all := make(map[int]string)
	for _, c := range r.calls {
		maps.Copy(all, c)
	}
	return all
}

var _ = Describe("Worker", func() {
	It("Should fail without a Handle function", func() {
		Expect(coalesce.Open[int, string]()).Error().
			To(MatchError(ContainSubstring("handle")))
	})

	It("Should handle every key that Add receives from many goroutines", func() {
		rec := &recorder{}
		w := DeferClose(MustOpen(coalesce.Open(coalesce.Config[int, string]{
			Handle: rec.handle,
		})))
		var wg sync.WaitGroup
		for g := range 8 {
			wg.Go(func() {
				for i := range 100 {
					w.Add(g*100+i, "v")
				}
			})
		}
		wg.Wait()
		want := make(map[int]string)
		for k := range 800 {
			want[k] = "v"
		}
		Eventually(rec.handled).Should(Equal(want))
	})

	It("Should hand only the latest value of a key to Handle", func() {
		var (
			started = make(chan struct{})
			release = make(chan struct{})
			calls   = make(chan map[int]string, 2)
		)
		w := DeferClose(MustOpen(coalesce.Open(coalesce.Config[int, string]{
			Handle: func(_ context.Context, latest map[int]string) {
				if _, ok := latest[0]; ok {
					close(started)
					<-release
				}
				calls <- latest
			},
		})))
		w.Add(0, "hold")
		Eventually(started).Should(BeClosed())
		w.Add(1, "set")
		w.Add(1, "delete")
		w.Add(2, "set")
		close(release)
		Eventually(calls).Should(Receive(Equal(map[int]string{0: "hold"})))
		Eventually(calls).Should(Receive(Equal(map[int]string{1: "delete", 2: "set"})))
	})

	It("Should end the context of a running Handle when Close runs", func() {
		started := make(chan struct{})
		w := MustOpen(coalesce.Open(coalesce.Config[int, string]{
			Handle: func(ctx context.Context, _ map[int]string) {
				close(started)
				<-ctx.Done()
			},
		}))
		w.Add(1, "v")
		Eventually(started).Should(BeClosed())
		Expect(w.Close()).To(Succeed())
	})

	It("Should not wait in Add while Handle runs", func() {
		release := make(chan struct{})
		w := DeferClose(MustOpen(coalesce.Open(coalesce.Config[int, string]{
			Handle: func(ctx context.Context, _ map[int]string) {
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
				w.Add(i, "v")
			}
		}()
		Eventually(done).Should(BeClosed())
	})
})
