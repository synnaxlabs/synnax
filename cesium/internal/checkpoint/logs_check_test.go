// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint_test

import (
	"context"
	"strings"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

// headFault fails the first Sync of the file named failSync and the first Remove of the
// name failRemove.
type headFault struct {
	xfs.FS
	mu                   sync.Mutex
	failSync, failRemove string
}

func (f *headFault) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	return &syncFault{File: file, fs: f, name: name}, nil
}

func (f *headFault) Remove(name string) error {
	f.mu.Lock()
	fail := name == f.failRemove
	if fail {
		f.failRemove = ""
	}
	f.mu.Unlock()
	if fail {
		return errors.New("remove failed")
	}
	return f.FS.Remove(name)
}

type syncFault struct {
	xfs.File
	fs   *headFault
	name string
}

func (f *syncFault) Sync() error {
	f.fs.mu.Lock()
	fail := f.name == f.fs.failSync
	if fail {
		f.fs.failSync = ""
	}
	f.fs.mu.Unlock()
	if fail {
		return errSync
	}
	return f.File.Sync()
}

var _ = Describe("Check: logs while heads fail", func() {
	It("Should keep a complete older head and its logs after a failed head whose "+
		"removal failed", func() {
		fs := &headFault{
			FS:         xfs.NewMem(),
			failSync:   checkpoint.Name(2),
			failRemove: checkpoint.Name(2),
		}
		for _, g := range []uint64{1, 2, 3} {
			Expect(log.Prepare(fs, g, logSize)).To(Succeed())
		}
		MustSucceed(checkpoint.Create(fs, db, 1, &state.Snapshot{
			Sequence: 1,
		}, 0))
		// head runs a checkpointer over fs for the head of generation, and returns the
		// error of Close, which is the error of that head.
		head := func(generation uint64) error {
			GinkgoHelper()
			c := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       db,
				LogSize:  logSize,
				Prepared: func(uint64, error) {},
				Executor: goExec,
			}))
			ran := make(chan error, 1)
			go func() { ran <- c.Run(context.Background()) }()
			c.Head(generation, sample(2))
			err := c.Close()
			Eventually(ran).Should(Receive(BeNil()))
			return err
		}
		Expect(head(2)).To(MatchError(errSync))
		Expect(head(3)).To(Succeed())
		Expect(readErr(fs, 3)).To(Succeed())
		// Recovery falls back to the head before the newest when the newest fails on
		// two reads. One older head must pass, with every log from it on.
		var passed []uint64
		for _, g := range []uint64{1, 2} {
			if readErr(fs, g) == nil {
				passed = append(passed, g)
			}
		}
		Expect(passed).ToNot(BeEmpty())
		for g := passed[len(passed)-1]; g <= 3; g++ {
			Expect(MustSucceed(fs.Exists(log.Name(g)))).To(BeTrue())
		}
	})

	It("Should keep at most 16 logs in the root while heads fail", func() {
		const (
			size    = 64 << 10
			maxLogs = 16
		)
		fs := xfs.NewMem()
		Expect(log.Prepare(fs, 1, size)).To(Succeed())
		MustSucceed(checkpoint.Create(fs, db, 1, &state.Snapshot{
			Sequence: 1,
		}, 0))
		p := &prepared{}
		c := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  size,
			Prepared: p.on,
			Executor: goExec,
		}))
		ran := make(chan error, 1)
		go func() { ran <- c.Run(context.Background()) }()
		DeferCleanup(func() {
			Expect(c.Close()).To(Succeed())
			Eventually(ran).Should(Receive(BeNil()))
		})
		// The log asks for log 2 at its start. At each switch it asks for the log after
		// it, and no head is written, as when every head fails. It switches only to a
		// file that is ready.
		c.Prepare(2)
		for g := uint64(2); g <= maxLogs+4; g++ {
			Eventually(p.count).Should(Equal(int(g - 1)))
			p.mu.Lock()
			err := p.errs[g-2]
			p.mu.Unlock()
			if err != nil {
				break
			}
			c.Prepare(g + 1)
		}
		logs := 0
		for _, e := range MustSucceed(fs.List("")) {
			if strings.HasSuffix(e.Name(), ".log") {
				logs++
			}
		}
		Expect(logs).To(BeNumerically("<=", maxLogs))
		p.mu.Lock()
		defer p.mu.Unlock()
		Expect(p.errs).To(ContainElement(
			MatchError("16 logs since the oldest head: disk full"),
		))
	})
})
