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
		Expect(checkpoint.Write(fs, 1, checkpoint.Encode(db, 1, &state.Snapshot{
			Sequence: 1,
		}))).To(Succeed())
		c := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Snapshot: func() (*state.Snapshot, error) { return sample(2), nil },
			Prepared: func(uint64, error) {},
			Executor: goExec,
		}))
		start(c)
		c.Switched(2)
		Eventually(c.Err).Should(MatchError(errSync))
		c.Switched(3)
		Eventually(func() error { return readErr(fs, 3) }).Should(Succeed())
		Eventually(c.Err).Should(Succeed())
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

	It("Should keep at most MaxLogs logs in the root while heads fail", func() {
		const size = 64 << 10
		fs := xfs.NewMem()
		Expect(log.Prepare(fs, 1, size)).To(Succeed())
		Expect(checkpoint.Write(fs, 1, checkpoint.Encode(db, 1, &state.Snapshot{
			Sequence: 1,
		}))).To(Succeed())
		p := &prepared{}
		fault := errors.New("no head")
		c := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  size,
			Snapshot: func() (*state.Snapshot, error) { return nil, fault },
			Prepared: p.on,
			Executor: goExec,
		}))
		ran := make(chan error, 1)
		go func() { ran <- c.Run(context.Background()) }()
		DeferCleanup(func() {
			_ = c.Close()
			Eventually(ran).Should(Receive(BeNil()))
		})
		// The log asks for log 2 at its start. At each switch it reports the new log
		// and asks for the one after it. It switches only to a file that is ready.
		c.Prepare(2)
		for g := uint64(2); g <= checkpoint.MaxLogs+4; g++ {
			Eventually(p.count).Should(Equal(int(g - 1)))
			p.mu.Lock()
			err := p.errs[g-2]
			p.mu.Unlock()
			if err != nil {
				break
			}
			c.Switched(g)
			c.Prepare(g + 1)
		}
		Eventually(c.Err).Should(MatchError(fault))
		logs := 0
		for _, e := range MustSucceed(fs.List("")) {
			if strings.HasSuffix(e.Name(), ".log") {
				logs++
			}
		}
		Expect(logs).To(BeNumerically("<=", checkpoint.MaxLogs))
		p.mu.Lock()
		defer p.mu.Unlock()
		Expect(p.errs).To(ContainElement(MatchError(checkpoint.ErrDiskFull)))
	})
})
