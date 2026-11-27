// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package crash_test

import (
	"cmp"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"slices"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/io/fs/testutil/crash"
)

// filePerCommit writes each commit to a file of its own, after its length and
// checksum, and syncs the file but never the directory. Only the order of the directory
// changes keeps its commits a prefix. It skips a file that a crash tore.
type filePerCommit struct{ fs xfs.FS }

func openFilePerCommit(_ context.Context, fs xfs.FS) (crash.Engine, error) {
	return &filePerCommit{fs: fs}, nil
}

func (e *filePerCommit) Commit(
	_ context.Context,
	r crash.Record,
	_ crash.Ack,
) (crash.Ack, error) {
	f, err := e.fs.Open(fmt.Sprintf("%06d", r.ID), os.O_CREATE|os.O_RDWR)
	if err != nil {
		return 0, err
	}
	b := make([]byte, 8+len(r.Payload))
	binary.LittleEndian.PutUint32(b, uint32(len(r.Payload)))
	binary.LittleEndian.PutUint32(b[4:], crc32.ChecksumIEEE(r.Payload))
	copy(b[8:], r.Payload)
	if _, err = f.Write(b); err != nil {
		return 0, errors.Combine(err, f.Close())
	}
	return crash.Visible, errors.Combine(f.Sync(), f.Close())
}

func (e *filePerCommit) Recovered(context.Context) ([]crash.Record, error) {
	infos, err := e.fs.List("")
	if err != nil {
		return nil, err
	}
	records := make([]crash.Record, 0, len(infos))
	for _, info := range infos {
		id, err := strconv.ParseUint(info.Name(), 10, 64)
		if err != nil {
			return nil, err
		}
		b, err := xfs.ReadFile(e.fs, info.Name())
		if err != nil {
			return nil, err
		}
		if len(b) < 8 || int(binary.LittleEndian.Uint32(b)) != len(b)-8 ||
			binary.LittleEndian.Uint32(b[4:]) != crc32.ChecksumIEEE(b[8:]) {
			continue
		}
		records = append(records, crash.Record{ID: id, Payload: b[8:]})
	}
	slices.SortFunc(records, func(a, b crash.Record) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return records, nil
}

func (e *filePerCommit) Close() error { return nil }

var _ = Describe("Directory change order", func() {
	// violations returns the error of each of 16 run keys on which a Partial run of a
	// filePerCommit fails.
	violations := func(ctx context.Context, ordered bool) []string {
		GinkgoHelper()
		var out []string
		for runKey := range uint64(16) {
			err := violation(ctx, crash.Config{
				RunKey:         runKey,
				Open:           openFilePerCommit,
				Commits:        8,
				Stride:         1,
				Profile:        crash.Partial,
				EntriesOrdered: ordered,
			})
			if err != nil {
				out = append(out, err.Error())
			}
		}
		return out
	}

	It("Should pass an engine that needs ordered directory changes when a crash "+
		"keeps them in order", func(ctx SpecContext) {
		Expect(violations(ctx, true)).To(BeEmpty())
	})

	It("Should catch the hole that a crash leaves when it keeps directory changes out "+
		"of order", func(ctx SpecContext) {
		Expect(violations(ctx, false)).To(ContainElement(ContainSubstring("R3 broken")))
	})
})
