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
	"sync"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Budget check", func() {
	It("Should keep the limit and every byte with handles used from many goroutines",
		func() {
			for runKey := range uint64(32) {
				fs := newOpFS(xfs.NewMem())
				b := MustSucceed(segment.NewBudget(3, 2, segment.GoExecutor))
				hs := make([]*segment.Handle, 8)
				for i := range hs {
					hs[i] = MustSucceed(b.Create(fs, fmt.Sprintf("%d.seg", i)))
				}
				var wg sync.WaitGroup
				errs := make([]error, len(hs))
				for i := range hs {
					wg.Go(func() {
						r := rand.New(rand.NewPCG(runKey, uint64(i)))
						for n := range 150 {
							o := hs[r.IntN(len(hs))]
							var err error
							switch r.IntN(4) {
							case 0:
								_, err = hs[i].WriteAt([]byte{byte(n)}, int64(n))
							case 1:
								_, err = o.ReadAt(make([]byte, 1), 0)
								if errors.Is(err, io.EOF) {
									err = nil
								}
							case 2:
								err = b.Sync([]*segment.Handle{o})
							default:
								err = b.Sync([]*segment.Handle{o, hs[i]})
							}
							if err != nil {
								errs[i] = err
								return
							}
							if r.IntN(4) == 0 {
								_, err = hs[i].WriteAt([]byte{byte(n)}, int64(n))
								if err != nil {
									errs[i] = err
									return
								}
							}
						}
					})
				}
				wg.Wait()
				Expect(errors.Join(errs...)).ToNot(HaveOccurred(), "run key %d", runKey)
				Expect(b.Sync(hs)).To(Succeed())
				for _, h := range hs {
					Expect(h.Close()).To(Succeed())
				}
				Expect(fs.MaxOpen()).To(BeNumerically("<=", 3), "run key %d", runKey)
			}
		},
	)

	It("Should refuse a write after another caller's sync of the handle failed",
		func() {
			var tried, wrote []uint64
			for runKey := range uint64(128) {
				ffs := MustSucceed(fault.New(fault.Config{
					RunKey:        runKey,
					SyncErrorRate: 0.5,
				}))
				g := newGateFS(ffs)
				budget := MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor))
				h := MustSucceed(budget.Create(g, "1.seg"))
				if budget.Sync([]*segment.Handle{h}) != nil {
					Expect(h.Close()).To(Succeed())
					continue
				}
				bErr := make(chan error, 1)
				at := int64(0)
				for {
					Expect(h.WriteAt([]byte("abc"), at)).To(Equal(3))
					at += 3
					go func() { bErr <- budget.Sync([]*segment.Handle{h}) }()
					select {
					case err := <-bErr:
						Expect(err).ToNot(HaveOccurred())
						continue
					case <-g.failed:
					}
					break
				}
				// A write that does not wait for B passes inside the window.
				wDone := callWithin(func() error {
					_, err := h.WriteAt([]byte("late"), at)
					return err
				})
				close(g.resume)
				Expect(<-bErr).To(MatchError(syscall.EIO))
				wErr := <-wDone
				tried = append(tried, runKey)
				if wErr == nil {
					wrote = append(wrote, runKey)
				}
				Expect(h.Close()).To(Succeed())
			}
			Expect(len(tried)).To(BeNumerically(">=", 32))
			Expect(wrote).To(BeEmpty(), fmt.Sprintf(
				"of %d run keys, a write after a failed sync of the file passed",
				len(tried),
			))
		},
	)
})
