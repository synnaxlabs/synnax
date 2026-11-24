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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// The spec holds over 2 GiB of segments in memory.
var _ = Describe("Check: the adopt of a Pending that sealed a segment", func() {
	It("Should make the sealed segment durable at the next sync of the adopter",
		func(ctx SpecContext) {
			const (
				idx, data = cesium.ChannelKey(1), cesium.ChannelKey(2)
				// n values of 64 KiB pass the 1 GiB of a segment.
				n, chunk = 16384, 512
			)
			keys := []cesium.ChannelKey{idx, data}
			disk := MustSucceed(fault.New(fault.Config{}))
			open := func(fs *fault.FS) *cesium.DB {
				GinkgoHelper()
				return MustSucceed(cesium.Open(
					ctx,
					"",
					cesium.WithFS(fs),
					cesium.WithPointerLog(),
					cesium.WithRandom(rand.NewPCG(1, 2)),
				))
			}
			db := open(disk)
			stop := StartRun(db.Run)
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
					DataType: telem.StringT,
				},
			)).To(Succeed())
			cfg := func(keys ...cesium.ChannelKey) cesium.WriterConfig {
				return cesium.WriterConfig{
					Channels: keys,
					Clock:    idx,
					Start:    telem.SecondTS,
				}
			}
			w0 := MustSucceed(db.OpenWriter(ctx, cfg(keys...)))
			value := strings.Repeat("x", 64<<10)
			for from := 0; from < n; from += chunk {
				stamps := make([]telem.TimeStamp, chunk)
				values := make([]string, chunk)
				for i := range chunk {
					stamps[i] = telem.TimeStamp(from+i+1) * telem.SecondTS
					values[i] = value
				}
				MustSucceed2(w0.Write(ctx, telem.MultiFrame(keys, []telem.Series{
					telem.NewSeries(stamps),
					telem.NewSeries(values),
				}), nil))
			}
			// A handoff to the clock alone leaves the data as a Pending. Its commit
			// seals the full segment of data, then waits on the stamps that the clock
			// has not made durable.
			w1, pending := MustSucceed2(db.Handoff(ctx, w0, cfg(idx), nil))
			Expect(pending).ToNot(BeNil())
			MustSucceed2(w1.Commit(ctx, nil))
			_, released := MustSucceed2(pending.Commit(ctx))
			Expect(released).To(BeFalse())
			w2, rest := MustSucceed2(db.Handoff(ctx, w1, cfg(keys...), pending))
			Expect(rest).To(BeNil())
			Expect(w2.Sync(ctx)).To(Succeed())
			image := MustSucceed(disk.Crash(fault.CrashConfig{}))
			Expect(w2.Close()).To(Succeed())
			stop()
			Expect(db.Close()).To(Succeed())
			db = open(image)
			stop = StartRun(db.Run)
			Expect(MustSucceed(db.Read(ctx, telem.TimeRangeMax, data)).Get(data).Len()).
				To(Equal(int64(n)))
			stop()
			Expect(db.Close()).To(Succeed())
		})
})
