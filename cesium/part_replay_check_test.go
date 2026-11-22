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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/log"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// appendEntry writes a frame of one entry of records after the last frame of the log of
// generation in fs, as the log owner of database db writes its next frame.
func appendEntry(fs xfs.FS, db, generation uint64, records ...log.Record) {
	GinkgoHelper()
	f := MustSucceed(fs.Open(log.Name(generation), os.O_RDWR))
	b := make([]byte, MustSucceed(f.Stat()).Size())
	MustSucceed(f.ReadAt(b, 0))
	var (
		last log.Frame
		end  int64
	)
	for at := int64(log.FramesStart); at+log.FrameHeaderSize <= int64(len(b)); {
		fr, err := log.DecodeFrame(b[at:], db, generation, uint64(at))
		if err != nil {
			// The log writes a mark at the next block boundary.
			next := (at + log.BlockSize - 1) &^ (log.BlockSize - 1)
			if next == at {
				break
			}
			at = next
			if fr, err = log.DecodeFrame(
				b[at:],
				db,
				generation,
				uint64(at),
			); err != nil {
				break
			}
		}
		last, end = fr, at+int64(fr.Size())
		at = end
	}
	Expect(end).ToNot(BeZero(), "log %d holds no frame", generation)
	var e log.Entry
	for _, r := range records {
		e.Add(r)
	}
	next := log.Frame{
		Count:    1,
		Sequence: last.Sequence + uint64(last.Count),
		PrevCRC:  last.CRC,
		Entries:  e.Bytes(),
	}
	MustSucceed(f.WriteAt(next.Append(nil, db, generation, uint64(end)), end))
	Expect(f.Sync()).To(Succeed())
	Expect(f.Close()).To(Succeed())
}

var _ = Describe("Check: replay over parted data", func() {
	It("Should refuse to replay a Tombstone of a clock over parted data that another "+
		"key times", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db := partedDB(ctx, fs)
		stop := StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		writeParted(ctx, db, fs, idx, data, 5, 10)
		stop()
		Expect(db.Close()).To(Succeed())
		Expect(parted(fs, 2, data)).To(Equal([]uint64{1, 1, 1, 1, 0}))
		// The Tombstone takes seconds 1 to 40 of the clock, which data times in its
		// four parted segments alone.
		h, _ := MustSucceed2(checkpoint.Read(fs, 2))
		appendEntry(fs, h.DB, 2, &log.Tombstone{
			Key:   idx,
			Start: int64(telem.SecondTS),
			End:   int64(41 * telem.SecondTS),
		})
		reopened, err := cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithPartSize(1),
			cesium.WithIdentity(h.DB),
		)
		if err == nil {
			DeferCleanup(reopened.Close)
		}
		Expect(err).To(MatchError(ContainSubstring("tombstone of clock")))
	})
})
