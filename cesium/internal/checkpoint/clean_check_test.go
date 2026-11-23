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
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// crashAtSyncDir stops before each SyncDir, as a crash there does.
type crashAtSyncDir struct{ xfs.FS }

func (crashAtSyncDir) SyncDir(string) error { return errors.New("crash") }

var _ = Describe("Check: a crash in the first clean of a checkpointer", func() {
	// clean runs a checkpointer over fs whose recovery opened head 10, until its first
	// step cleans the root, and returns the error of Close.
	clean := func(fs xfs.FS) error {
		GinkgoHelper()
		c := MustSucceed(checkpoint.New(checkpoint.Config{
			FS:       fs,
			DB:       db,
			LogSize:  logSize,
			Prepared: func(uint64, error) {},
			Executor: goExec,
			Live:     10,
		}))
		return running(c)()
	}
	for key := range uint64(32) {
		It("Should leave the same files for one run key, run key "+itoa(int(key)),
			func() {
				images := make([][]string, 0, 8)
				for range 8 {
					disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
					for g := uint64(1); g <= 10; g++ {
						for _, name := range []string{checkpoint.Name(g), log.Name(g)} {
							f := MustSucceed(disk.Open(name, os.O_CREATE|os.O_RDWR))
							Expect(f.Close()).To(Succeed())
						}
					}
					Expect(disk.SyncDir("")).To(Succeed())
					Expect(clean(crashAtSyncDir{disk})).
						To(MatchError(ContainSubstring("crash")))
					image := MustSucceed(disk.Crash(fault.CrashConfig{
						RunKey:            key,
						EntrySurvivalRate: 0.5,
						EntriesOrdered:    true,
					}))
					entries := MustSucceed(image.List(""))
					names := make([]string, 0, len(entries))
					for _, e := range entries {
						names = append(names, e.Name())
					}
					images = append(images, names)
				}
				for _, names := range images[1:] {
					Expect(names).To(Equal(images[0]))
				}
			})

		It("Should leave the same files for one run key when the same removes run "+
			"in one order, run key "+itoa(int(key)), func() {
			images := make([][]string, 0, 8)
			for range 8 {
				disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
				for g := uint64(1); g <= 10; g++ {
					for _, name := range []string{checkpoint.Name(g), log.Name(g)} {
						f := MustSucceed(disk.Open(name, os.O_CREATE|os.O_RDWR))
						Expect(f.Close()).To(Succeed())
					}
				}
				Expect(disk.SyncDir("")).To(Succeed())
				for g := uint64(1); g < 9; g++ {
					Expect(disk.Remove(checkpoint.Name(g))).To(Succeed())
				}
				for g := uint64(1); g < 9; g++ {
					Expect(disk.Remove(log.Name(g))).To(Succeed())
				}
				image := MustSucceed(disk.Crash(fault.CrashConfig{
					RunKey:            key,
					EntrySurvivalRate: 0.5,
					EntriesOrdered:    true,
				}))
				entries := MustSucceed(image.List(""))
				names := make([]string, 0, len(entries))
				for _, e := range entries {
					names = append(names, e.Name())
				}
				images = append(images, names)
			}
			for _, names := range images[1:] {
				Expect(names).To(Equal(images[0]))
			}
		})
	}
})
