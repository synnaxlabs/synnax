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
	"context"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Pointer log delete check", func() {
	var (
		fs        xfs.FS
		db        *cesium.DB
		stop      func()
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	open := func(ctx context.Context) {
		GinkgoHelper()
		db = MustOpen(cesium.Open(
			ctx,
			"",
			cesium.WithFS(fs),
			cesium.WithPointerLog(),
			cesium.WithRandom(rand.NewPCG(1, 2)),
			pcgIdentity(1, 2),
		))
		stop = StartRun(db.Run)
	}
	BeforeEach(func(ctx SpecContext) {
		fs = xfs.NewMem()
		open(ctx)
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		keys = []cesium.ChannelKey{idx, data}
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "data", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
	})
	write := func(ctx context.Context, seconds ...telem.TimeStamp) {
		GinkgoHelper()
		values := make([]int64, len(seconds))
		for i, s := range seconds {
			values[i] = int64(s)
		}
		w := MustSucceed(db.OpenWriter(ctx, cesium.WriterConfig{
			Channels: keys,
			Clock:    idx,
			Start:    seconds[0] * telem.SecondTS,
		}))
		MustSucceed2(w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeriesSecondsTSV(seconds...),
			telem.NewSeries(values),
		}), nil))
		MustSucceed2(w.Commit(ctx, nil))
		Expect(w.Close()).To(Succeed())
	}
	segments := func(key cesium.ChannelKey) int {
		GinkgoHelper()
		n := 0
		for _, info := range MustSucceed(fs.List(strconv.Itoa(int(key)))) {
			if strings.HasSuffix(info.Name(), ".seg") {
				n++
			}
		}
		return n
	}

	It("Should unlink a segment that a delete empties once a switch seals it", func(
		ctx SpecContext,
	) {
		write(ctx, 10, 11, 12)
		Expect(db.DeleteTimeRange(ctx, keys, telem.TimeRange{
			Start: 10 * telem.SecondTS,
			End:   13 * telem.SecondTS,
		})).To(Succeed())
		// Each Intend takes one block of the log: 12,400 of them pass three quarters,
		// and the switch seals every open segment that no writer holds.
		for range 12_400 {
			Expect(db.Intend([]cesium.ChannelKey{999})).To(Succeed())
		}
		// The log switches on the next entry after log 2 is prepared.
		Eventually(func() (bool, error) {
			if err := db.Intend([]cesium.ChannelKey{999}); err != nil {
				return false, err
			}
			return fs.Exists("0000000000000002.head")
		}).WithTimeout(10 * time.Second).Should(BeTrue())
		// The close of a writer takes the segments that left the state, and each is
		// unlinked once the log covers the record that took it out.
		write(ctx, 20, 21)
		for _, k := range keys {
			Eventually(func() int { return segments(k) }).
				Should(Equal(1), "segment files of key %d", k)
		}
		stop()
		Expect(db.Close()).To(Succeed())
		open(ctx)
		for _, k := range keys {
			Expect(segments(k)).To(Equal(1), "segment files of key %d after a "+
				"reopen", k)
		}
	})
})
