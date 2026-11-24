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
	"maps"
	"os"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: a crash in a create that writes parts", func() {
	for runKey := range uint64(16) {
		It("Should open every segment of head 1 from each image, or create it again, "+
			"run key "+itoa(int(runKey)), func() {
			disk := MustSucceed(fault.New(fault.Config{}))
			im := &crashImages{disk: disk, runKey: runKey}
			s := withSealed(sealed(1, 10), sealed(2, 20), sealed(3, 30), sealed(4, 40))
			want := []uint64{1, 2, 3, 4, 100}
			MustSucceed(checkpoint.Create(imageFS{FS: disk, im: im}, db, 1, s, 1))
			h, _ := MustSucceed2(checkpoint.Read(disk, 1))
			Expect(h.Parts).To(HaveLen(4))
			Expect(created(disk, s)).To(Equal(want))
			im.mu.Lock()
			defer im.mu.Unlock()
			var failures []string
			for i, image := range im.list {
				got, err := created(image, s)
				if err != nil {
					failures = append(failures, fmt.Sprintf("image %d: %v", i, err))
					continue
				}
				if !slices.Equal(got, want) {
					failures = append(failures, fmt.Sprintf(
						"image %d: head 1 gives segments %v, not %v", i, got, want,
					))
				}
			}
			Expect(failures).To(BeEmpty(), "%d images", len(im.list))
		})
	}
})

var _ = Describe("Check: switches after a create that writes parts", func() {
	It("Should drop the dead parts of a create, and give a new part the next identity",
		func() {
			// Each part holds two sealed segment records of one pointer each.
			const partSize = 2 * (32 + 36)
			fs := xfs.NewMem()
			for g := uint64(1); g <= 3; g++ {
				Expect(log.Prepare(fs, g, logSize)).To(Succeed())
			}
			x := MustSucceed(checkpoint.Create(fs, db, 1, withSealed(
				sealed(1, 10), sealed(2, 20), sealed(3, 30), sealed(4, 40),
			), partSize))
			c := MustSucceed(checkpoint.New(checkpoint.Config{
				FS:       fs,
				DB:       db,
				LogSize:  logSize,
				Prepared: func(uint64, error) {},
				Executor: goExec,
				Parts:    x,
				PartSize: partSize,
				Live:     1,
			}))
			start(c)
			// The state forgets the records that parts hold, so a part record dies only
			// by a kill.
			heads := []struct {
				s     *state.Snapshot
				parts []uint64
				ids   []uint64
			}{
				{
					s:     killed(withSealed(sealed(5, 50)), 1, 1, 2),
					parts: []uint64{2},
					ids:   []uint64{3, 4, 5, 100},
				},
				{
					s: killed(
						withSealed(
							sealed(5, 50),
							sealed(6, 60),
							sealed(7, 70),
						),
						2,
						3,
						4,
					),
					parts: []uint64{3},
					ids:   []uint64{5, 6, 7, 100},
				},
			}
			for i, want := range heads {
				g := uint64(i + 2)
				c.Head(g, want.s)
				Eventually(func() error { return readErr(fs, g) }).Should(Succeed())
				h, got := MustSucceed2(checkpoint.Read(fs, g))
				var parts []uint64
				for _, p := range h.Parts {
					parts = append(parts, p.ID)
				}
				Expect(parts).To(Equal(want.parts), "head %d", g)
				x := MustSucceed(checkpoint.LoadParts(fs, db, h))
				Expect(segmentIDs(x, got)).To(Equal(want.ids), "head %d", g)
			}
		})
})

// created returns the segments of head 1 of fs with the parts that it names. A head 1
// that is missing or not complete starts the root as a new database, so created removes
// it and creates s again first, as the engine does.
func created(fs xfs.FS, s *state.Snapshot) ([]uint64, error) {
	_, _, err := checkpoint.Read(fs, 1)
	if errors.Is(err, os.ErrNotExist) || errors.Is(err, checkpoint.ErrIncomplete) {
		err = createAgain(fs, s)
	}
	if err != nil {
		return nil, errors.Wrap(err, "head 1")
	}
	h, got, err := checkpoint.Read(fs, 1)
	if err != nil {
		return nil, errors.Wrap(err, "head 1")
	}
	x, err := checkpoint.LoadParts(fs, db, h)
	if err != nil {
		return nil, errors.Wrap(err, "head 1")
	}
	return segmentIDs(x, got), nil
}

// segmentIDs returns the identities of the segments of each channel of s, a head whose
// parts x loads: the records of the head and the live records of the parts, in rising
// order for each channel.
func segmentIDs(x *checkpoint.Index, s *state.Snapshot) []uint64 {
	var ids []uint64
	for _, ch := range s.Channels {
		at := len(ids)
		ids = slices.AppendSeq(ids, maps.Keys(x.Live(ch.Key)))
		for _, seg := range ch.Segments {
			ids = append(ids, seg.ID)
		}
		slices.Sort(ids[at:])
	}
	return ids
}

// createAgain removes head 1 of fs when it exists, and creates s.
func createAgain(fs xfs.FS, s *state.Snapshot) error {
	ok, err := fs.Exists(checkpoint.Name(1))
	if err != nil {
		return err
	}
	if ok {
		if err = fs.Remove(checkpoint.Name(1)); err != nil {
			return err
		}
	}
	_, err = checkpoint.Create(fs, db, 1, s, 1)
	return errors.Wrap(err, "create again")
}
