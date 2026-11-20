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
	"io"
	"math/rand/v2"
	"os"
	"sync"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// tracked is one handle of the random spec: its bytes, the length that a passed sync
// covers, and whether a sync of it failed.
type tracked struct {
	h      *segment.Handle
	name   string
	mu     sync.Mutex
	data   []byte
	acked  int
	failed atomic.Bool
}

var _ = Describe("Round 2 check, random handles", func() {
	for k := range uint64(32) {
		It(fmt.Sprintf("Should never ack a byte that a crash loses, nor pass a call "+
			"after a failed sync, run key %d", k), func() {
			fs := MustSucceed(fault.New(fault.Config{
				RunKey:        k,
				SyncErrorRate: 0.03,
				EvictionRate:  0.5,
			}))
			b := MustSucceed(segment.NewBudget(2, 2, segment.GoExecutor))
			ts := make([]*tracked, 4)
			for i := range ts {
				name := fmt.Sprintf("%d.seg", i)
				h, err := b.Create(fs, name)
				Expect(err).ToNot(HaveOccurred())
				ts[i] = &tracked{h: h, name: name}
			}
			// syncOne syncs t, and acks the bytes written before the call when the
			// sync passes.
			syncOne := func(t *tracked) {
				failedBefore := t.failed.Load()
				t.mu.Lock()
				n := len(t.data)
				t.mu.Unlock()
				err := b.Sync([]*segment.Handle{t.h})
				if failedBefore {
					Expect(err).To(HaveOccurred(), "a sync after a failed sync of %s",
						t.name)
				}
				if err != nil {
					t.failed.Store(true)
					return
				}
				t.mu.Lock()
				t.acked = max(t.acked, n)
				t.mu.Unlock()
			}
			var wg sync.WaitGroup
			for i, t := range ts {
				wg.Go(func() {
					defer GinkgoRecover()
					rng := rand.New(rand.NewPCG(k, uint64(i)))
					for range 40 {
						chunk := make([]byte, 1+rng.IntN(6000))
						for j := range chunk {
							chunk[j] = byte(rng.Uint32())
						}
						failedBefore := t.failed.Load()
						t.mu.Lock()
						off := int64(len(t.data))
						t.mu.Unlock()
						_, err := t.h.WriteAt(chunk, off)
						if failedBefore {
							Expect(err).To(HaveOccurred(), "a write after a failed "+
								"sync of %s", t.name)
						}
						if err != nil {
							t.failed.Store(true)
							return
						}
						t.mu.Lock()
						t.data = append(t.data, chunk...)
						t.mu.Unlock()
						if rng.IntN(3) == 0 {
							syncOne(t)
						}
					}
				})
			}
			wg.Go(func() {
				defer GinkgoRecover()
				rng := rand.New(rand.NewPCG(k, 99))
				for range 60 {
					syncOne(ts[rng.IntN(len(ts))])
				}
			})
			wg.Wait()
			for _, t := range ts {
				Expect(t.h.Close()).To(Succeed())
			}
			img := MustSucceed(fs.Crash(fault.CrashConfig{RunKey: k}))
			for _, t := range ts {
				if t.acked == 0 {
					continue
				}
				f := MustSucceed(img.Open(t.name, os.O_RDONLY))
				got := MustSucceed(io.ReadAll(f))
				Expect(f.Close()).To(Succeed())
				Expect(len(got)).To(BeNumerically(">=", t.acked), t.name)
				Expect(got[:t.acked]).To(Equal(t.data[:t.acked]), t.name)
			}
			for _, t := range ts {
				fmt.Fprintf(GinkgoWriter, "key %d %s: %d bytes, %d acked, failed %v\n",
					k, t.name, len(t.data), t.acked, t.failed.Load())
			}
		})
	}
})
