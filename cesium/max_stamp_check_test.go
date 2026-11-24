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
	"fmt"
	"math/rand/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Check: a sample at the largest stamp", func() {
	It("Should refuse a sample that no exclusive end can follow, as the old layout "+
		"does", func(ctx SpecContext) {
		fs := xfs.NewMem()
		open := func(opts ...cesium.Option) (*cesium.DB, func()) {
			GinkgoHelper()
			db := MustSucceed(cesium.Open(ctx, "", append([]cesium.Option{
				cesium.WithFS(fs),
				cesium.WithPointerLog(),
				cesium.WithRandom(rand.NewPCG(1, 2)),
			}, opts...)...))
			stop := StartRun(db.Run)
			return db, func() {
				GinkgoHelper()
				stop()
				Expect(db.Close()).To(Succeed())
			}
		}
		db, closeDB := open()
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
			Start:    telem.TimeStampMax,
		}))
		_, _, werr := w.Write(ctx, telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries([]telem.TimeStamp{telem.TimeStampMax}),
			telem.NewSeries([]int64{7}),
		}), nil)
		var (
			end  telem.TimeStamp
			cerr error
		)
		if werr == nil {
			end, _, cerr = w.Commit(ctx, nil)
		}
		err := errors.Join(werr, cerr)
		if err == nil {
			// The facts of a stored sample at the largest stamp, for the report.
			count := func(db *cesium.DB) int64 {
				return MustSucceed(
					db.Read(ctx, telem.TimeRangeMax, data),
				).Get(data).
					Len()
			}
			before := count(db)
			Expect(w.Close()).To(Succeed())
			id := cesium.WithIdentity(db.Identity())
			closeDB()
			re, closeRe := open(id)
			after := count(re)
			closeRe()
			Fail(fmt.Sprintf(
				"the commit of a sample at %d took it: end %d, %d samples read, %d "+
					"after a reopen",
				telem.TimeStampMax, end, before, after,
			))
		}
		Expect(err).To(MatchError(validate.ErrValidation))
		Expect(w.Close()).To(MatchError(validate.ErrValidation))
		closeDB()
	})
})
