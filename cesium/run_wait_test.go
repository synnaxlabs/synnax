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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calls on a pointer-log DB that does not run", func() {
	var db *cesium.DB
	BeforeEach(func(ctx SpecContext) {
		db = MustOpen(
			cesium.Open(ctx, "", cesium.WithFS(xfs.NewMem()), cesium.WithPointerLog()),
		)
		DeferCleanup(func() { Expect(db.Close()).To(Succeed()) })
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: 1, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		})).To(Succeed())
	})

	DescribeTable(
		"Should return the error of the caller's context at its deadline",
		func(call func(ctx context.Context) error) {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				50*time.Millisecond,
			)
			defer cancel()
			Expect(call(ctx)).To(MatchError(context.DeadlineExceeded))
		},
		Entry("a Sweep", func(ctx context.Context) error {
			_, _, err := db.Sweep(ctx, nil)
			return err
		}),
		Entry("a Drop", func(ctx context.Context) error {
			return db.Drop(ctx, []cesium.ChannelKey{1})
		}),
		Entry("a delete of a time range", func(ctx context.Context) error {
			return db.DeleteTimeRange(ctx, []cesium.ChannelKey{1}, telem.TimeRangeMax)
		}),
		Entry("a writer open", func(ctx context.Context) error {
			_, err := db.OpenWriter(ctx, cesium.WriterConfig{
				Channels: []cesium.ChannelKey{1},
				Clock:    1,
				Start:    telem.SecondTS,
			})
			return err
		}),
	)
})
