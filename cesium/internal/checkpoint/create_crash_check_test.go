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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
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
			MustSucceed(checkpoint.Create(imageFS{FS: disk, im: im}, db, s, 1))
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
	parts, err := checkpoint.LoadParts(fs, db, h)
	if err != nil {
		return nil, errors.Wrap(err, "head 1")
	}
	parts.Merge(got)
	var ids []uint64
	for _, ch := range got.Channels {
		for _, seg := range ch.Segments {
			ids = append(ids, seg.ID)
		}
	}
	return ids, nil
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
	_, err = checkpoint.Create(fs, db, s, 1)
	return errors.Wrap(err, "create again")
}
