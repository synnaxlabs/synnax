// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"encoding/binary"
	"os"
	"strconv"
	"sync/atomic"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: second read of a log in blocks", func() {
	It("Should refuse a log whose valid frame after its last one only a second read "+
		"finds, in a later block of that read", func(ctx SpecContext) {
		// block is the size of the blocks of the second read.
		const block = 4 << 20
		fs := xfs.NewMem()
		db, closeDB := MustSucceed2(openLog(ctx, fs))
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
		Expect(closeDB()).To(Succeed())
		f := MustSucceed(fs.Open(log.Name(1), os.O_RDWR))
		b := make([]byte, 1<<20)
		MustSucceed(f.ReadAt(b, 0))
		end := int64(0)
		for off := int64(log.FramesStart); off+log.FrameHeaderSize <= int64(
			len(b),
		); off += 8 {
			if n, ok := log.FrameSize(b[off:]); ok {
				end = off + int64(n)
			}
		}
		Expect(end).ToNot(BeZero())
		// The frame starts in the second block from the end of the last frame.
		at := (end + block + 64) &^ 7
		frame := log.Frame{Count: 1, Sequence: 1, Entries: make([]byte, 256)}
		MustSucceed(f.WriteAt(
			frame.Append(nil, binary.LittleEndian.Uint64(b[16:]), 1, uint64(at)),
			at,
		))
		Expect(f.Close()).To(Succeed())
		Expect(openLog(
			ctx,
			tailFS{FS: fs, from: at, done: &atomic.Bool{}},
		)).Error().To(And(
			MatchError(place.ErrCorrupt),
			MatchError(ContainSubstring(
				"holds a valid frame at "+strconv.FormatInt(at, 10),
			)),
		))
	})
})
