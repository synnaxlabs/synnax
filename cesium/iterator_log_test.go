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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Reads on the pointer log", func() {
	It("Should open each segment file once across reads of its ranges", func(
		ctx SpecContext,
	) {
		rec := NewRecorder(xfs.NewMem())
		db := mustOpenDBOnFS(ctx, rec, cesium.WithPointerLog())
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		keys := []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Start:    telem.SecondTS,
		}))
		const commits = 10
		for i := range commits {
			s := telem.TimeStamp(i + 1)
			MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(s),
				telem.NewSeries([]int64{int64(s)}),
			}), nil))
			MustSucceed2(w.Commit(ctx, nil))
		}
		Expect(w.Close()).To(Succeed())
		rec.Reset()
		for i := range commits {
			s := telem.TimeStamp(i+1) * telem.SecondTS
			fr := MustSucceed(db.Read(ctx, s.Range(s+1), data))
			Expect(fr.Get(data).Series).To(HaveLen(1))
			Expect(fr.Get(data).Series[0].Len()).To(BeEquivalentTo(1))
		}
		opens := map[string]int{}
		for _, e := range rec.Events() {
			if e.Op == OpOpen && strings.HasSuffix(e.Name, ".seg") {
				opens[e.Name]++
			}
		}
		Expect(opens).To(HaveLen(len(keys)))
		for name, n := range opens {
			Expect(n).To(Equal(1), name)
		}
	})
})
