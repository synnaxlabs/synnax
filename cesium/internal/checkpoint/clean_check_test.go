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

var _ = Describe("Check: a crash in Clean", func() {
	for key := range uint64(32) {
		It("Should leave the same files for one run key, run key "+itoa(int(key)),
			func() {
				var images [][]string
				for range 8 {
					disk := MustSucceed(fault.New(fault.Config{RunKey: key}))
					for g := uint64(1); g <= 10; g++ {
						for _, name := range []string{checkpoint.Name(g), log.Name(g)} {
							f := MustSucceed(disk.Open(name, os.O_CREATE|os.O_RDWR))
							Expect(f.Close()).To(Succeed())
						}
					}
					Expect(disk.SyncDir("")).To(Succeed())
					Expect(checkpoint.Clean(crashAtSyncDir{disk}, 10)).
						To(MatchError(ContainSubstring("crash")))
					image := MustSucceed(disk.Crash(fault.CrashConfig{
						RunKey:            key,
						EntrySurvivalRate: 0.5,
						EntriesOrdered:    true,
					}))
					var names []string
					for _, e := range MustSucceed(image.List("")) {
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
			var images [][]string
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
				var names []string
				for _, e := range MustSucceed(image.List("")) {
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
