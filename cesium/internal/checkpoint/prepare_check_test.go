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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: prepare after a failed sync", func() {
	// prepare runs a checkpointer over fs until the spec ends. It returns a function
	// that queues the prepare of log 4 and returns its result.
	prepare := func(fs xfs.FS) func() error {
		GinkgoHelper()
		p := &prepared{}
		c := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Prepared: p.on,
			Executor: goExec,
		}))
		start(c)
		return func() error {
			GinkgoHelper()
			n := p.count()
			c.Prepare(4)
			Eventually(p.count).Should(Equal(n + 1))
			p.mu.Lock()
			defer p.mu.Unlock()
			return p.errs[n]
		}
	}
	// durable expects log 4 at its size in a crash image of disk.
	durable := func(disk *fault.FS, key uint64) {
		GinkgoHelper()
		image := MustSucceed(disk.Crash(fault.CrashConfig{
			RunKey:            key,
			DataSurvivalRate:  0.5,
			TornRate:          0.2,
			EntrySurvivalRate: 0.5,
		}))
		Expect(MustSucceed(image.Exists(log.Name(4)))).To(BeTrue())
		Expect(MustSucceed(image.Stat(log.Name(4))).Size()).To(Equal(int64(logSize)))
	}
	for key := range uint64(32) {
		It("Should leave a durable log file of its size once Prepare succeeds after "+
			"a failed sync, run key "+itoa(int(key)), func() {
			disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
			next := prepare(&headFault{FS: disk, failSync: log.Name(4)})
			Expect(next()).To(MatchError(errSync))
			Expect(next()).To(Succeed())
			durable(disk, key)
		})

		It("Should leave a durable log file of its size once Prepare succeeds, run "+
			"key "+itoa(int(key)), func() {
			disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
			Expect(prepare(disk)()).To(Succeed())
			durable(disk, key)
		})

		It("Should leave a durable log file of its size once Prepare succeeds on the "+
			"file of a Prepare that a crash cut short, run key "+itoa(int(key)),
			func() {
				disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
				// All of the zeros of the file, with no sync of the file or its entry.
				f := MustSucceed(disk.Open(log.Name(4), os.O_CREATE|os.O_RDWR))
				MustSucceed(f.WriteAt(make([]byte, logSize), 0))
				Expect(f.Close()).To(Succeed())
				Expect(prepare(disk)()).To(Succeed())
				durable(disk, key)
			})
	}
})
