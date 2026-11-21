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
	"math/rand/v2"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/place"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Checked reads on the pointer log", func() {
	It("Should refuse a read of a block whose bytes changed on disk, and name its key",
		func(ctx SpecContext) {
			fs := xfs.NewMem()
			db := MustOpen(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
			))
			StartRun(db.Run)
			idx, data := GenerateChannelKey(), GenerateChannelKey()
			keys := []cesium.ChannelKey{idx, data}
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      data,
					Name:     "data",
					Index:    idx,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: keys,
				Start:    10 * telem.SecondTS,
			}))
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12),
				telem.NewSeriesV[int64](1, 2, 3),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
			Expect(w.Close()).To(Succeed())
			Expect(
				MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).Get(data).Series,
			).
				To(HaveLen(1))
			dir := strconv.FormatUint(uint64(data), 10)
			var segs []string
			for _, e := range MustSucceed(fs.List(dir)) {
				if strings.HasSuffix(e.Name(), ".seg") {
					segs = append(segs, e.Name())
				}
			}
			Expect(segs).To(HaveLen(1))
			f := MustSucceed(fs.Open(dir+"/"+segs[0], os.O_RDWR))
			b := make([]byte, 1)
			MustSucceed(f.ReadAt(b, 8))
			b[0] ^= 1
			MustSucceed(f.WriteAt(b, 8))
			Expect(f.Close()).To(Succeed())
			Expect(db.Read(ctx, telem.TimeRangeMax, data)).Error().To(SatisfyAll(
				MatchError(place.ErrCorrupt),
				MatchError(ContainSubstring("of key "+dir+" fails two reads")),
			))
			Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, idx)).Get(idx).Series).
				To(HaveLen(1), "another key reads on")
		})
})
