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
	"bytes"
	"io"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// headComplete reports whether the head of generation 2 has its header sector.
func headComplete(fs xfs.FS) (bool, error) {
	f, err := fs.Open("0000000000000002.head", os.O_RDONLY)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	b := make([]byte, 8)
	_, err = f.ReadAt(b, 0)
	if err = errors.Combine(err, f.Close()); errors.Is(err, io.EOF) {
		return false, nil
	}
	return bytes.Equal(b, []byte("CSCHKPT\x00")), err
}

var _ = Describe("Delete on the pointer log, broken by a head", func() {
	It("Should close a writer after a reopen that replays a Tombstone over a sealed "+
		"segment of the head", func(ctx SpecContext) {
		fs := xfs.NewMem()
		db, closeDB := breakOpen(ctx, fs)
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: breakIdx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key: breakData, Name: "data", Index: breakIdx, DataType: telem.Int64T,
			},
		)).To(Succeed())
		breakWrite(ctx, db, 10, 11)
		breakWrite(ctx, db, 20, 21, 22)
		next := cesium.ChannelKey(1_000)
		intend := func(n int) {
			keys := make([]cesium.ChannelKey, n)
			for i := range keys {
				keys[i], next = next, next+1
			}
			Expect(db.Intend(keys)).To(Succeed())
		}
		for range 4 {
			intend(900_000)
		}
		Eventually(func() (bool, error) {
			intend(1)
			return headComplete(fs)
		}).WithTimeout(time.Minute).Should(BeTrue())
		Expect(db.DeleteTimeRange(
			ctx,
			[]cesium.ChannelKey{breakData},
			(0 * telem.SecondTS).Range(15*telem.SecondTS),
		)).To(Succeed())
		closeDB()
		db, closeDB = breakOpen(ctx, fs)
		Expect(breakValues(ctx, db)).To(Equal([]int64{20, 21, 22}))
		keys := []cesium.ChannelKey{breakIdx, breakData}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    breakIdx,
			Start:    30 * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(30, 31),
			telem.NewSeriesV[int64](30, 31),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
		Expect(breakValues(ctx, db)).To(Equal([]int64{20, 21, 22, 30, 31}))
		closeDB()
	})

	It("Should keep the cuts of a Tombstone that the head of the next log covers",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			db, closeDB := breakOpen(ctx, fs)
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: breakIdx, Name: "time", IsIndex: true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: breakData, Name: "data", Index: breakIdx,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			breakWrite(ctx, db, 10, 11, 12, 13, 14, 15, 16)
			breakWrite(ctx, db, 20, 21, 22)
			keys := []cesium.ChannelKey{breakIdx, breakData}
			Expect(db.DeleteTimeRange(
				ctx, keys, (12 * telem.SecondTS).Range(15*telem.SecondTS),
			)).To(Succeed())
			Expect(db.DeleteTimeRange(
				ctx,
				[]cesium.ChannelKey{breakData},
				(21 * telem.SecondTS).Range(22*telem.SecondTS),
			)).To(Succeed())
			next := cesium.ChannelKey(1_000)
			intend := func(n int) {
				ks := make([]cesium.ChannelKey, n)
				for i := range ks {
					ks[i], next = next, next+1
				}
				Expect(db.Intend(ks)).To(Succeed())
			}
			for range 4 {
				intend(900_000)
			}
			Eventually(func() (bool, error) {
				intend(1)
				return headComplete(fs)
			}).WithTimeout(time.Minute).Should(BeTrue())
			closeDB()
			db, closeDB = breakOpen(ctx, fs)
			Expect(breakValues(ctx, db)).
				To(Equal([]int64{10, 11, 15, 16, 20, 22}))
			Expect(db.DeleteTimeRange(
				ctx, keys, (11 * telem.SecondTS).Range(16*telem.SecondTS),
			)).To(Succeed())
			breakWrite(ctx, db, 12, 13)
			Expect(breakValues(ctx, db)).
				To(Equal([]int64{10, 12, 13, 16, 20, 22}))
			closeDB()
			db, closeDB = breakOpen(ctx, fs)
			Expect(breakValues(ctx, db)).
				To(Equal([]int64{10, 12, 13, 16, 20, 22}))
			closeDB()
		})
})
