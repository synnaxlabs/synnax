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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// sessionSegments returns the names of the segment files of key on fs.
func sessionSegments(fs xfs.FS, key cesium.ChannelKey) []string {
	GinkgoHelper()
	var names []string
	for _, info := range MustSucceed(fs.List(strconv.FormatUint(uint64(key), 10))) {
		if strings.HasSuffix(info.Name(), ".seg") {
			names = append(names, info.Name())
		}
	}
	return names
}

var _ = Describe("Check: writer sessions on one clock on the pointer log", func() {
	It("Should append each session to the open segments that the one before left",
		func(ctx SpecContext) {
			const idx, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
			keys := []cesium.ChannelKey{idx, data}
			fs := xfs.NewMem()
			db := MustSucceed(cesium.Open(
				ctx,
				"",
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
			))
			stop := StartRun(db.Run)
			DeferCleanup(func() {
				stop()
				Expect(db.Close()).To(Succeed())
			})
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key:      idx,
					Name:     "time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key:      data,
					Name:     "data",
					Index:    idx,
					DataType: telem.Int64T,
				},
			)).To(Succeed())
			for _, s := range []telem.TimeStamp{10, 20, 30} {
				w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
					Channels: keys,
					Start:    s * telem.SecondTS,
				}))
				MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(s, s+1),
					telem.NewSeriesV[int64](int64(s), int64(s+1)),
				}), nil))
				MustSucceed2(w.Commit(ctx, nil))
				Expect(w.Close()).To(Succeed())
			}
			Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).Get(data).Len()).
				To(Equal(int64(6)))
			Expect(sessionSegments(fs, idx)).To(HaveLen(1))
			Expect(sessionSegments(fs, data)).To(HaveLen(1))
		})
})
