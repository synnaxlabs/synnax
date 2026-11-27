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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Chunked read failure check", func() {
	It("Should read and log a channel whose read failed no more in later chunks of "+
		"the same read", func(ctx SpecContext) {
		mem := fs.NewMem()
		ins, logs := ObservedInstrumentation(zapcore.ErrorLevel)
		// Two channels on one index read one sample each per chunk.
		db := mustOpenDBOnFS(
			ctx,
			mem,
			cesium.WithInstrumentation(ins),
			cesium.WithChunkSize(2),
		)
		idx, good := GenerateChannelKey(), GenerateChannelKey()
		bad := GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: good, Name: "good", Index: idx, DataType: telem.Int64T},
			cesium.Channel{Key: bad, Name: "bad", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		Expect(db.Write(ctx, 10*telem.SecondTS, telem.MultiFrame(
			[]cesium.ChannelKey{idx, good, bad},
			[]telem.Series{
				telem.NewSeriesSecondsTSV(10, 11, 12),
				telem.NewSeriesV[int64](1, 2, 3),
				telem.NewSeriesV[int64](4, 5, 6),
			},
		))).To(Succeed())
		Expect(mem.Remove(channelKeyToPath(bad) + "/1.domain")).To(Succeed())
		i := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
			Channels: []cesium.ChannelKey{good, bad},
			Bounds:   telem.TimeRangeMax,
		}))
		Expect(i.SeekFirst()).To(BeTrue())
		Expect(i.Next(10 * telem.Second)).To(BeTrue())
		Expect(i.Value().Get(bad).Series).To(BeEmpty())
		Expect(i.Value().Get(good).Len()).To(Equal(int64(3)))
		Expect(logs.FilterMessage("read of a channel failed").All()).To(HaveLen(1))
		Expect(i.Close()).To(MatchError(ContainSubstring(
			"open " + channelKeyToPath(bad) + "/1.domain: file does not exist",
		)))
	})
})
