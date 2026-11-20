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
	"os"
	"runtime"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/segment"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

// holdFS counts the syncs of each file, fails the syncs of a file, and holds one sync
// or close until the test opens its gate.
type holdFS struct {
	xfs.FS
	mu      sync.Mutex
	syncs   map[string]int
	syncErr map[string]error
	gates   map[string]chan struct{}
	entered chan string
}

func newHoldFS(fs xfs.FS) *holdFS {
	return &holdFS{
		FS:      fs,
		syncs:   map[string]int{},
		syncErr: map[string]error{},
		gates:   map[string]chan struct{}{},
		entered: make(chan string, 16),
	}
}

// Gate holds the next op until the returned channel closes.
func (f *holdFS) Gate(op string) chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	g := make(chan struct{})
	f.gates[op] = g
	return g
}

func (f *holdFS) Syncs(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.syncs[name]
}

func (f *holdFS) wait(op string) {
	f.mu.Lock()
	g := f.gates[op]
	delete(f.gates, op)
	f.mu.Unlock()
	if g != nil {
		f.entered <- op
		<-g
	}
}

func (f *holdFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return &holdFile{File: file, fs: f, name: name}, nil
}

type holdFile struct {
	xfs.File
	fs   *holdFS
	name string
}

func (f *holdFile) Sync() error {
	f.fs.mu.Lock()
	f.fs.syncs[f.name]++
	err := f.fs.syncErr[f.name]
	f.fs.mu.Unlock()
	f.fs.wait("sync " + f.name)
	if err != nil {
		return err
	}
	return f.File.Sync()
}

func (f *holdFile) Close() error {
	f.fs.wait("close " + f.name)
	return f.File.Close()
}

var _ = Describe("Round 2 check", func() {
	It("Should finish a budget sync of a handle whose close is in flight, when "+
		"the executor runs one job at a time", func() {
		fs := newOpFS(xfs.NewMem())
		jobs := make(chan func(), 16)
		worker := make(chan struct{})
		go func() {
			defer close(worker)
			for f := range jobs {
				f()
			}
		}()
		exec := func(f func()) { jobs <- f }
		b := MustSucceed(segment.NewBudget(1, 4, exec))
		h := MustSucceed(b.Create(fs, "0.seg"))
		Expect(h.Sync()).To(Succeed())
		Expect(h.WriteAt([]byte("0.seg"), 0)).To(Equal(5))
		// An earlier job keeps the worker busy while the next two queue.
		gate, started := make(chan struct{}), make(chan struct{})
		exec(func() { close(started); <-gate })
		Eventually(started).Should(BeClosed())
		synced := make(chan error, 1)
		go func() { synced <- b.Sync([]*segment.Handle{h}) }()
		Eventually(func() int { return len(jobs) }).Should(Equal(1))
		created := make(chan *segment.Handle, 1)
		go func() {
			defer GinkgoRecover()
			created <- MustSucceed(b.Create(fs, "1.seg"))
		}()
		Eventually(func() int { return len(jobs) }).Should(Equal(2))
		DeferCleanup(func() {
			for len(jobs) > 0 {
				(<-jobs)()
			}
			Eventually(synced).Should(Receive())
			var second *segment.Handle
			Eventually(created).Should(Receive(&second))
			close(jobs)
			Eventually(worker).Should(BeClosed())
			Expect(second.Close()).To(Succeed())
			Expect(h.Close()).To(Succeed())
		})
		close(gate)
		Eventually(synced).Should(Receive(BeNil()))
	})

	for attempt := range 8 {
		It(fmt.Sprintf("Should not sync a file again after its own sync failed, when "+
			"a close of the budget takes the handle before the mark, attempt %d",
			attempt), func() {
			prev := runtime.GOMAXPROCS(1)
			DeferCleanup(func() { runtime.GOMAXPROCS(prev) })
			mem := xfs.NewMem()
			for _, name := range []string{"d1.seg", "d2.seg", "w.seg"} {
				f := MustSucceed(mem.Open(name, os.O_CREATE|os.O_RDWR))
				Expect(f.Close()).To(Succeed())
			}
			fs := newHoldFS(mem)
			b := MustSucceed(segment.NewBudget(3, 4, segment.GoExecutor))
			h1 := MustSucceed(b.Create(fs, "1.seg"))
			Expect(h1.Sync()).To(Succeed())
			Expect(h1.WriteAt([]byte("1"), 0)).To(Equal(1))
			c1 := MustSucceed(b.Create(fs, "c1.seg"))
			c2 := MustSucceed(b.Create(fs, "c2.seg"))
			d1, d2, w := b.Open(fs, "d1.seg"), b.Open(fs, "d2.seg"), b.Open(fs, "w.seg")

			fs.mu.Lock()
			fs.syncErr["1.seg"] = errInjected
			fs.mu.Unlock()
			syncGate := fs.Gate("sync 1.seg")
			synced := make(chan error, 1)
			go func() { synced <- h1.Sync() }()
			Eventually(fs.entered).Should(Receive(Equal("sync 1.seg")))

			// Two closes hold the lock of the budget, one after the other, while the
			// failed sync and three opens wait for it.
			close1, close2 := fs.Gate("close c1.seg"), fs.Gate("close c2.seg")
			closed := make(chan error, 2)
			go func() {
				closed <- c1.Close()
				closed <- c2.Close()
			}()
			Eventually(fs.entered).Should(Receive(Equal("close c1.seg")))
			close(syncGate)
			time.Sleep(5 * time.Millisecond)
			wrote := make(chan error, 3)
			for _, h := range []*segment.Handle{d1, d2, w} {
				go func() {
					_, err := h.WriteAt([]byte("x"), 0)
					wrote <- err
				}()
				time.Sleep(5 * time.Millisecond)
			}
			close(close1)
			Eventually(fs.entered).Should(Receive(Equal("close c2.seg")))
			time.Sleep(5 * time.Millisecond)
			close(close2)

			Eventually(synced).Should(Receive(MatchError(errInjected)))
			for range 2 {
				Eventually(closed).Should(Receive(BeNil()))
			}
			for range 3 {
				Eventually(wrote).Should(Receive(BeNil()))
			}
			for _, h := range []*segment.Handle{h1, d1, d2, w} {
				Expect(h.Close()).To(Succeed())
			}
			Expect(fs.Syncs("1.seg")).To(
				Equal(1),
				"a file whose sync failed is never synced again",
			)
		})
	}
})
