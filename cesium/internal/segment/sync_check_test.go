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
	"os"
	"sync"
	"syscall"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	fstestutil "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// gateFS holds the first file sync that fails after it returns, until resume closes.
type gateFS struct {
	xfs.FS
	once   sync.Once
	failed chan struct{}
	resume chan struct{}
}

func newGateFS(fs xfs.FS) *gateFS {
	return &gateFS{FS: fs, failed: make(chan struct{}), resume: make(chan struct{})}
}

func (g *gateFS) Open(name string, flag int) (xfs.File, error) {
	f, err := g.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return &gateFile{File: f, g: g}, nil
}

type gateFile struct {
	xfs.File
	g *gateFS
}

func (f *gateFile) Sync() error {
	err := f.File.Sync()
	if err != nil {
		hold := false
		f.g.once.Do(func() { hold = true })
		if hold {
			close(f.g.failed)
			<-f.g.resume
		}
	}
	return err
}

func durableBytes(ffs *fault.FS, name string, runKey uint64) []byte {
	GinkgoHelper()
	img := MustSucceed(ffs.Crash(fault.CrashConfig{RunKey: runKey}))
	f := MustSucceed(img.Open(name, os.O_RDONLY))
	defer func() { Expect(f.Close()).To(Succeed()) }()
	return MustSucceed(io.ReadAll(f))
}

var _ = Describe("Sync check", func() {
	It(
		"Should fail a later Sync of a handle whose sync by another caller failed",
		func() {
			var tried, lost []uint64
			for runKey := range uint64(128) {
				ffs := MustSucceed(fault.New(fault.Config{
					RunKey:        runKey,
					SyncErrorRate: 0.5,
				}))
				g := newGateFS(ffs)
				b := MustSucceed(segment.NewBudget(4, 4, segment.GoExecutor))
				h := MustSucceed(b.Create(g, "1.seg"))
				if b.Sync([]*segment.Handle{h}) != nil {
					Expect(h.Close()).To(Succeed())
					continue
				}
				var (
					data []byte
					bErr = make(chan error, 1)
				)
				for {
					chunk := fmt.Appendf(
						nil,
						"write %d of run key %d;",
						len(data),
						runKey,
					)
					Expect(h.WriteAt(chunk, int64(len(data)))).To(Equal(len(chunk)))
					data = append(data, chunk...)
					go func() { bErr <- b.Sync([]*segment.Handle{h}) }()
					select {
					case err := <-bErr:
						Expect(err).ToNot(HaveOccurred())
						continue
					case <-g.failed:
					}
					break
				}
				// Caller B's sync failed and has not returned. Caller A syncs now. A
				// sync
				// that does not wait for B returns inside the window.
				aDone := callWithin(
					func() error { return b.Sync([]*segment.Handle{h}) },
				)
				close(g.resume)
				Expect(<-bErr).To(MatchError(syscall.EIO))
				aErr := <-aDone
				tried = append(tried, runKey)
				if aErr == nil {
					if string(durableBytes(ffs, "1.seg", runKey)) != string(data) {
						lost = append(lost, runKey)
					}
				}
				Expect(h.Close()).To(Succeed())
			}
			Expect(len(tried)).To(BeNumerically(">=", 32))
			Expect(lost).To(BeEmpty(), fmt.Sprintf(
				"of %d run keys, Sync returned nil after a failed sync and the "+
					"bytes it acked are not durable after a crash",
				len(tried),
			))
		},
	)

	for fsName, makeFS := range fstestutil.FileSystems {
		Context(fsName, func() {
			var fs *opFS
			BeforeEach(func() {
				inner := makeFS()
				MustSucceed(inner.Sub("12"))
				fs = newOpFS(inner)
			})

			It(
				"Should never sync a file again after the budget's sync of it failed",
				func() {
					b := MustSucceed(segment.NewBudget(1, 1, segment.GoExecutor))
					first := MustSucceed(b.Create(fs, "1.seg"))
					Expect(first.WriteAt([]byte("1.seg"), 0)).To(Equal(5))
					fs.FailSync("1.seg", errInjected)
					release := fs.Hold()
					created := make(chan *segment.Handle, 1)
					go func() { created <- MustSucceed(b.Create(fs, "2.seg")) }()
					Eventually(fs.held).Should(Receive(Equal("1.seg")))
					fs.FailSync("1.seg", nil)
					fs.Unhold()
					synced := make(chan error, 1)
					go func() { synced <- b.Sync([]*segment.Handle{first}) }()
					Consistently(synced, 200*time.Millisecond).ShouldNot(Receive())
					close(release)
					Expect(<-synced).To(MatchError(ContainSubstring(
						"segment file 1.seg failed, and it is never synced or " +
							"written again",
					)))
					second := <-created
					Expect(second.Close()).To(Succeed())
					Expect(first.Close()).To(Succeed())
					Expect(fs.Ops()).ToNot(ContainElement("sync 1.seg"))
				},
			)

			It(
				"Should run the sync before a budget close inside the width of the "+
					"pool",
				func() {
					b := MustSucceed(segment.NewBudget(2, 1, segment.GoExecutor))
					first := MustSucceed(b.Create(fs, "1.seg"))
					second := MustSucceed(b.Create(fs, "2.seg"))
					Expect(first.WriteAt([]byte("1.seg"), 0)).To(Equal(5))
					Expect(second.WriteAt([]byte("2.seg"), 0)).To(Equal(5))
					release := fs.Hold()
					pooled := make(chan error, 1)
					go func() { pooled <- b.Sync([]*segment.Handle{first}) }()
					Eventually(fs.held).Should(Receive(Equal("1.seg")))
					created := make(chan *segment.Handle, 1)
					go func() { created <- MustSucceed(b.Create(fs, "3.seg")) }()
					var outside string
					select {
					case outside = <-fs.held:
					case <-time.After(300 * time.Millisecond):
					}
					fs.Unhold()
					close(release)
					Expect(<-pooled).To(Succeed())
					third := <-created
					for _, h := range []*segment.Handle{first, second, third} {
						Expect(h.Close()).To(Succeed())
					}
					Expect(outside).To(BeEmpty(), "a sync ran outside the pool")
					Expect(fs.MaxSyncs()).To(Equal(1))
				},
			)

			It(
				"Should sync the directory of a new file once when two callers sync it",
				func() {
					budget := MustSucceed(segment.NewBudget(2, 2, segment.GoExecutor))
					h := MustSucceed(budget.Create(fs, "12/1.seg"))
					release := fs.Hold()
					done := make(chan error, 2)
					go func() { done <- budget.Sync([]*segment.Handle{h}) }()
					Eventually(fs.held).Should(Receive(Equal("12")))
					go func() { done <- budget.Sync([]*segment.Handle{h}) }()
					select {
					case <-fs.held:
					case <-time.After(300 * time.Millisecond):
					}
					fs.Unhold()
					close(release)
					Expect(<-done).To(Succeed())
					Expect(<-done).To(Succeed())
					Expect(h.Close()).To(Succeed())
					dirSyncs := 0
					for _, op := range fs.Ops() {
						if op == "syncdir 12" {
							dirSyncs++
						}
					}
					Expect(dirSyncs).To(Equal(1))
				},
			)
		})
	}
})

// callWithin runs f on its own goroutine, and returns once f returns or 100 ms pass,
// with the channel that gets the error of f.
func callWithin(f func() error) chan error {
	done := make(chan error, 1)
	go func() { done <- f() }()
	select {
	case err := <-done:
		done <- err
	case <-time.After(100 * time.Millisecond):
	}
	return done
}
