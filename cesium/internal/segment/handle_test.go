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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/errors"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var errInjected = errors.New("injected")

var _ = Describe("Handle", func() {
	It("Should refuse a budget below one handle", func() {
		Expect(segment.NewBudget(0, 4, segment.GoExecutor)).Error().
			To(MatchError(validate.ErrValidation))
	})

	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var fs *opFS
			BeforeEach(func() {
				inner := makeFS()
				MustSucceed(inner.Sub("12"))
				fs = newOpFS(inner)
			})

			newBudget := func(limit int) *segment.Budget {
				GinkgoHelper()
				return MustSucceed(segment.NewBudget(limit, 4, segment.GoExecutor))
			}

			// write writes the name of h's file at offset 0 of it.
			write := func(h *segment.Handle, name string) {
				GinkgoHelper()
				Expect(h.WriteAt([]byte(name), 0)).To(Equal(len(name)))
			}

			read := func(h *segment.Handle, n int) string {
				GinkgoHelper()
				b := make([]byte, n)
				Expect(h.ReadAt(b, 0)).To(Equal(n))
				return string(b)
			}

			It(
				"Should keep at most its limit of files open, and reopen a closed one",
				func() {
					b := newBudget(2)
					handles := make([]*segment.Handle, 5)
					for i := range handles {
						name := fmt.Sprintf("%d.seg", i)
						handles[i] = MustSucceed(b.Create(fs, name))
						write(handles[i], name)
					}
					for i, h := range handles {
						Expect(read(h, 5)).To(Equal(fmt.Sprintf("%d.seg", i)))
					}
					for _, h := range handles {
						Expect(h.Close()).To(Succeed())
					}
					Expect(fs.MaxOpen()).To(Equal(2))
				},
			)

			It("Should close the idle handle that was used least recently", func() {
				b := newBudget(2)
				first := MustSucceed(b.Create(fs, "1.seg"))
				second := MustSucceed(b.Create(fs, "2.seg"))
				write(first, "1.seg")
				third := MustSucceed(b.Create(fs, "3.seg"))
				Expect(fs.Ops()).To(Equal([]string{
					"open 1.seg", "open 2.seg", "close 2.seg", "open 3.seg",
				}))
				for _, h := range []*segment.Handle{first, second, third} {
					Expect(h.Close()).To(Succeed())
				}
			})

			It("Should sync a handle with unsynced bytes before it closes it", func() {
				b := newBudget(1)
				first := MustSucceed(b.Create(fs, "1.seg"))
				write(first, "1.seg")
				second := MustSucceed(b.Create(fs, "2.seg"))
				Expect(fs.Ops()).To(Equal([]string{
					"open 1.seg", "sync 1.seg", "close 1.seg", "open 2.seg",
				}))
				Expect(read(first, 5)).To(Equal("1.seg"))
				Expect(second.Close()).To(Succeed())
				Expect(first.Close()).To(Succeed())
			})

			It("Should close a synced handle with no second sync", func() {
				b := newBudget(1)
				first := MustSucceed(b.Create(fs, "1.seg"))
				write(first, "1.seg")
				Expect(first.Sync()).To(Succeed())
				second := MustSucceed(b.Create(fs, "2.seg"))
				Expect(fs.Ops()).To(Equal([]string{
					"open 1.seg", "sync 1.seg", "syncdir ", "close 1.seg", "open 2.seg",
				}))
				Expect(first.Sync()).To(Succeed())
				Expect(fs.Ops()).To(HaveLen(5), "a clean handle opens for no sync")
				Expect(second.Close()).To(Succeed())
				Expect(first.Close()).To(Succeed())
			})

			It(
				"Should mark a handle failed when the sync before its close fails",
				func() {
					b := newBudget(1)
					first := MustSucceed(b.Create(fs, "1.seg"))
					write(first, "1.seg")
					fs.FailSync("1.seg", errInjected)
					second := MustSucceed(b.Create(fs, "2.seg"))
					fs.FailSync("1.seg", nil)
					Expect(
						first.WriteAt([]byte("x"), 0),
					).Error().
						To(MatchError(errInjected))
					Expect(first.Sync()).To(MatchError(ContainSubstring(
						"segment file 1.seg failed, and it is never synced or " +
							"written again",
					)))
					Expect(read(first, 5)).To(Equal("1.seg"))
					Expect(fs.Ops()).ToNot(ContainElement("sync 1.seg"))
					Expect(second.Close()).To(Succeed())
					Expect(first.Close()).To(Succeed())
				},
			)

			It(
				"Should never sync or write a handle again after its own sync fails",
				func() {
					h := MustSucceed(newBudget(1).Create(fs, "1.seg"))
					write(h, "1.seg")
					fs.FailSync("1.seg", errInjected)
					Expect(h.Sync()).To(MatchError(errInjected))
					fs.FailSync("1.seg", nil)
					Expect(h.Sync()).To(MatchError(errInjected))
					Expect(
						h.WriteAt([]byte("x"), 0),
					).Error().
						To(MatchError(errInjected))
					Expect(fs.Ops()).To(Equal([]string{"open 1.seg"}))
					Expect(h.Close()).To(Succeed())
				},
			)

			It(
				"Should sync the directory of a created file at its first sync only",
				func() {
					b := newBudget(2)
					created := MustSucceed(b.Create(fs, "12/1.seg"))
					write(created, "12/1.s")
					Expect(created.Sync()).To(Succeed())
					write(created, "12/1.s")
					Expect(created.Sync()).To(Succeed())
					opened := b.Open(fs, "12/1.seg")
					Expect(read(opened, 6)).To(Equal("12/1.s"))
					Expect(opened.WriteAt([]byte("x"), 0)).To(Equal(1))
					Expect(opened.Sync()).To(Succeed())
					Expect(fs.Ops()).To(Equal([]string{
						"open 12/1.seg",
						"sync 12/1.seg",
						"syncdir 12",
						"sync 12/1.seg",
						"open 12/1.seg",
						"sync 12/1.seg",
					}))
					Expect(created.Close()).To(Succeed())
					Expect(opened.Close()).To(Succeed())
				},
			)

			It(
				"Should mark a handle failed when the sync of its directory fails",
				func() {
					h := MustSucceed(newBudget(1).Create(fs, "12/1.seg"))
					fs.FailSyncDir(errInjected)
					err := h.Sync()
					Expect(errors.Is(err, segment.ErrDirSync)).To(BeTrue())
					Expect(errors.Is(err, errInjected)).To(BeTrue())
					fs.FailSyncDir(nil)
					Expect(errors.Is(h.Sync(), segment.ErrDirSync)).To(BeTrue())
					Expect(h.Close()).To(Succeed())
				},
			)

			It("Should hold a write back until a sync of the handle returns", func() {
				h := MustSucceed(newBudget(1).Create(fs, "1.seg"))
				write(h, "1.seg")
				release := fs.Hold()
				synced := make(chan error, 1)
				go func() { synced <- h.Sync() }()
				Eventually(fs.held).Should(Receive(Equal("1.seg")))
				wrote := make(chan error, 1)
				go func() {
					_, err := h.WriteAt([]byte("2.seg"), 0)
					wrote <- err
				}()
				Consistently(wrote).ShouldNot(Receive())
				fs.Unhold()
				close(release)
				Eventually(synced).Should(Receive(BeNil()))
				Eventually(wrote).Should(Receive(BeNil()))
				Expect(h.Sync()).To(Succeed())
				Expect(fs.Ops()).To(Equal([]string{
					"open 1.seg", "sync 1.seg", "syncdir ", "sync 1.seg",
				}))
				Expect(read(h, 5)).To(Equal("2.seg"))
				Expect(h.Close()).To(Succeed())
			})

			It("Should fail a sync that waits for a failed sync of the same handle",
				func() {
					h := MustSucceed(newBudget(1).Create(fs, "1.seg"))
					write(h, "1.seg")
					fs.FailSync("1.seg", errInjected)
					release := fs.Hold()
					first := make(chan error, 1)
					go func() { first <- h.Sync() }()
					Eventually(fs.held).Should(Receive(Equal("1.seg")))
					second := make(chan error, 1)
					go func() { second <- h.Sync() }()
					wrote := make(chan error, 1)
					go func() {
						_, err := h.WriteAt([]byte("x"), 0)
						wrote <- err
					}()
					Consistently(fs.held).ShouldNot(Receive())
					fs.FailSync("1.seg", nil)
					fs.Unhold()
					close(release)
					Eventually(first).Should(Receive(MatchError(errInjected)))
					Eventually(second).Should(Receive(MatchError(errInjected)))
					Eventually(wrote).Should(Receive(MatchError(errInjected)))
					Expect(fs.Ops()).ToNot(ContainElement("sync 1.seg"))
					Expect(h.Close()).To(Succeed())
				},
			)

			It("Should not sync a file again after the sync before its close failed",
				func() {
					b := newBudget(1)
					first := MustSucceed(b.Create(fs, "1.seg"))
					write(first, "1.seg")
					fs.FailSync("1.seg", errInjected)
					release := fs.Hold()
					created := make(chan *segment.Handle, 1)
					go func() { created <- MustSucceed(b.Create(fs, "2.seg")) }()
					Eventually(fs.held).Should(Receive(Equal("1.seg")))
					synced := make(chan error, 1)
					go func() { synced <- first.Sync() }()
					read := make(chan error, 1)
					go func() {
						_, err := first.ReadAt(make([]byte, 5), 0)
						read <- err
					}()
					Consistently(synced).ShouldNot(Receive())
					Consistently(read).ShouldNot(Receive())
					fs.FailSync("1.seg", nil)
					fs.Unhold()
					close(release)
					Eventually(synced).Should(Receive(MatchError(errInjected)))
					var second *segment.Handle
					Eventually(created).Should(Receive(&second))
					Expect(second.Close()).To(Succeed())
					Eventually(read).Should(Receive(BeNil()))
					Expect(fs.Ops()).ToNot(ContainElement("sync 1.seg"))
					Expect(fs.Ops()).ToNot(ContainElement("syncdir "))
					Expect(first.Close()).To(Succeed())
				},
			)

			It("Should sync the directory of a created file once when two callers sync",
				func() {
					h := MustSucceed(newBudget(2).Create(fs, "12/1.seg"))
					release := fs.Hold()
					done := make(chan error, 2)
					go func() { done <- h.Sync() }()
					Eventually(fs.held).Should(Receive(Equal("12")))
					go func() { done <- h.Sync() }()
					Consistently(fs.held).ShouldNot(Receive())
					fs.Unhold()
					close(release)
					Eventually(done).Should(Receive(BeNil()))
					Eventually(done).Should(Receive(BeNil()))
					Expect(fs.Ops()).To(Equal([]string{"open 12/1.seg", "syncdir 12"}))
					Expect(h.Close()).To(Succeed())
				},
			)

			It("Should close a handle only once its calls in flight return", func() {
				h := MustSucceed(newBudget(1).Create(fs, "1.seg"))
				write(h, "1.seg")
				release := fs.Hold()
				synced := make(chan error, 1)
				go func() { synced <- h.Sync() }()
				Eventually(fs.held).Should(Receive(Equal("1.seg")))
				closed := make(chan error, 1)
				go func() { closed <- h.Close() }()
				Consistently(closed).ShouldNot(Receive())
				fs.Unhold()
				close(release)
				Eventually(synced).Should(Receive(BeNil()))
				Eventually(closed).Should(Receive(BeNil()))
				Expect(fs.Ops()).To(ContainElement("close 1.seg"))
			})

			It(
				"Should wait for an idle handle when every open handle is in use",
				func() {
					b := newBudget(1)
					first := MustSucceed(b.Create(fs, "1.seg"))
					write(first, "1.seg")
					release := fs.Hold()
					synced := make(chan error, 1)
					go func() { synced <- first.Sync() }()
					Eventually(fs.held).Should(Receive(Equal("1.seg")))
					created := make(chan *segment.Handle, 1)
					go func() { created <- MustSucceed(b.Create(fs, "2.seg")) }()
					Consistently(created).ShouldNot(Receive())
					fs.Unhold()
					close(release)
					Eventually(synced).Should(Receive(BeNil()))
					var second *segment.Handle
					Eventually(created).Should(Receive(&second))
					Expect(fs.MaxOpen()).To(Equal(1))
					Expect(second.Close()).To(Succeed())
					Expect(first.Close()).To(Succeed())
				},
			)

			It("Should refuse every call after Close", func() {
				h := MustSucceed(newBudget(1).Create(fs, "1.seg"))
				Expect(h.Close()).To(Succeed())
				Expect(h.Sync()).To(MatchError(segment.ErrClosed))
				Expect(h.ReadAt(make([]byte, 1), 0)).Error().
					To(MatchError(segment.ErrClosed))
				Expect(h.WriteAt([]byte("x"), 0)).Error().
					To(MatchError(segment.ErrClosed))
				Expect(h.Close()).To(Succeed())
			})
		})
	}
})
