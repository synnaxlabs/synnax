// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OnHistoryChange", func() {
	It("Should report a write into the past and a delete", func(ctx SpecContext) {
		var changes []telem.TimeRange
		db := MustOpen(unary.Open(ctx, unary.Config{
			FS:        fs.NewMem(),
			MetaCodec: json.Codec,
			Channel: channel.Channel{
				Name:     "Curie",
				Key:      1,
				DataType: telem.TimestampT,
				IsIndex:  true,
				Index:    1,
			},
			Instrumentation: PanicLogger(),
			OnHistoryChange: func(_ context.Context, tr telem.TimeRange) {
				changes = append(changes, tr)
			},
		}))
		write := func(stamps ...telem.TimeStamp) {
			GinkgoHelper()
			start := stamps[0] * telem.SecondTS
			series := telem.NewSeriesSecondsTSV(stamps...)
			Expect(unary.Write(ctx, db, start, series)).To(Succeed())
		}
		write(20, 21)
		write(10, 11)
		deleted := (20 * telem.SecondTS).Range(21 * telem.SecondTS)
		Expect(db.Delete(ctx, deleted)).To(Succeed())
		Expect(changes).To(Equal([]telem.TimeRange{
			(10 * telem.SecondTS).Range(11*telem.SecondTS + 1),
			deleted,
		}))
	})
})
