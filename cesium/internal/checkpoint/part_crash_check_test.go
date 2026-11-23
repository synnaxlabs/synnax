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
	"fmt"
	"os"
	"slices"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

// crashImages takes a crash image of disk after each call that changes it.
type crashImages struct {
	mu     sync.Mutex
	disk   *fault.FS
	runKey uint64
	list   []*fault.FS
}

func (im *crashImages) take() {
	im.mu.Lock()
	defer im.mu.Unlock()
	im.list = append(im.list, MustSucceed(im.disk.Crash(fault.CrashConfig{
		RunKey:            im.runKey*1_000 + uint64(len(im.list)),
		EntrySurvivalRate: 0.5,
		DataSurvivalRate:  0.5,
	})))
}

type imageFS struct {
	xfs.FS
	im *crashImages
}

func (f imageFS) Open(name string, flag int) (xfs.File, error) {
	file, err := f.FS.Open(name, flag)
	if err != nil {
		return nil, err
	}
	if flag&os.O_CREATE != 0 {
		f.im.take()
	}
	return imageFile{File: file, im: f.im}, nil
}

func (f imageFS) Remove(name string) error {
	defer f.im.take()
	return f.FS.Remove(name)
}

func (f imageFS) SyncDir(name string) error {
	defer f.im.take()
	return f.FS.SyncDir(name)
}

type imageFile struct {
	xfs.File
	im *crashImages
}

func (f imageFile) WriteAt(p []byte, off int64) (int, error) {
	defer f.im.take()
	return f.File.WriteAt(p, off)
}

func (f imageFile) Truncate(size int64) error {
	defer f.im.take()
	return f.File.Truncate(size)
}

func (f imageFile) Sync() error {
	defer f.im.take()
	return f.File.Sync()
}

var _ = Describe("Check: a crash in a switch that writes parts", func() {
	for runKey := range uint64(16) {
		It("Should open the state of the newest whole head from each image, run key "+
			itoa(int(runKey)), func() {
			disk := MustSucceed(fault.New(fault.Config{}))
			for g := uint64(1); g <= 4; g++ {
				f := MustSucceed(disk.Open(log.Name(g), os.O_CREATE|os.O_RDWR))
				Expect(f.Close()).To(Succeed())
			}
			Expect(checkpoint.Write(disk, 1, checkpoint.Encode(db, 1, withSealed()))).
				To(Succeed())
			Expect(disk.SyncDir("")).To(Succeed())
			im := &crashImages{disk: disk, runKey: runKey}
			c := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       imageFS{FS: disk, im: im},
				DB:       db,
				LogSize:  logSize,
				Prepared: func(uint64, error) {},
				Executor: goExec,
				PartSize: 1,
			}))
			stop := running(c)
			// Head 2 writes part 1. Head 3 writes part 1 again as part 2, since two of
			// its three segments are dead. Head 4 removes head 2 and part 1.
			want := map[uint64][]uint64{
				1: {100}, 2: {1, 2, 3, 100}, 3: {1, 100}, 4: {1, 100},
			}
			snapshots := map[uint64]*state.Snapshot{
				2: withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30)),
				3: killed(withSealed(sealed(1, 10)), 2, 2, 3),
				4: withSealed(sealed(1, 10)),
			}
			for g := uint64(2); g <= 4; g++ {
				c.Head(g, snapshots[g])
				Eventually(func() error { return readErr(disk, g) }).Should(Succeed())
			}
			Expect(stop()).To(Succeed())
			im.mu.Lock()
			defer im.mu.Unlock()
			var failures []string
			for i, image := range im.list {
				got, err := newestState(image)
				if err != nil {
					failures = append(failures, fmt.Sprintf("image %d: %v", i, err))
					continue
				}
				if !slices.Equal(got.ids, want[got.generation]) {
					failures = append(failures, fmt.Sprintf(
						"image %d: head %d gives segments %v, not %v",
						i, got.generation, got.ids, want[got.generation],
					))
				}
			}
			Expect(failures).To(BeEmpty(), "%d images", len(im.list))
		})
	}
})

type opened struct {
	generation uint64
	ids        []uint64
}

// newestState returns the segments of the newest head in fs that passes its checks,
// with the parts that it names. A head that is missing, not complete, or fails a check
// gives way to the head before it.
func newestState(fs xfs.FS) (opened, error) {
	for g := uint64(4); g >= 1; g-- {
		h, s, err := checkpoint.Read(fs, g)
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, checkpoint.ErrIncomplete) ||
			errors.Is(err, place.ErrCorrupt) {
			continue
		}
		if err != nil {
			return opened{}, errors.Wrapf(err, "head %d", g)
		}
		parts, err := checkpoint.LoadParts(fs, db, h)
		if err != nil {
			return opened{}, errors.Wrapf(err, "head %d", g)
		}
		parts.Merge(s)
		out := opened{generation: g}
		for _, ch := range s.Channels {
			for _, seg := range ch.Segments {
				out.ids = append(out.ids, seg.ID)
			}
		}
		return out, nil
	}
	return opened{}, errors.New("no head passes")
}
