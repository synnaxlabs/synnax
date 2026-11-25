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
	. "github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("History handlers that call the DB", func() {
	var (
		db         *cesium.DB
		key, other cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		db = mustOpenDBOnFS(ctx, xfs.NewMem())
		key, other = GenerateChannelKey(), GenerateChannelKey()
		for _, k := range []cesium.ChannelKey{key, other} {
			Expect(db.CreateChannel(ctx, cesium.Channel{
				Key: k, Name: "Scott", IsIndex: true, DataType: telem.TimestampT,
			})).To(Succeed())
			Expect(db.WriteSeries(
				ctx,
				k,
				10*telem.SecondTS,
				telem.NewSeriesSecondsTSV(10, 11),
			)).To(Succeed())
		}
	})
	written := (10 * telem.SecondTS).Range(11 * telem.SecondTS)
	calls := map[string]func(context.Context, cesium.HistoryChange) error{
		"a read of the changed range": func(
			ctx context.Context,
			c cesium.HistoryChange,
		) error {
			_, err := db.Read(ctx, c.TimeRange, c.Key)
			return err
		},
		"a channel create": func(ctx context.Context, _ cesium.HistoryChange) error {
			return db.CreateChannel(ctx, cesium.Channel{
				Key:      GenerateChannelKey(),
				Name:     "Amundsen",
				Virtual:  true,
				DataType: telem.Float32T,
			})
		},
		"a delete of another channel": func(
			ctx context.Context,
			_ cesium.HistoryChange,
		) error {
			return db.DeleteTimeRange(ctx, []cesium.ChannelKey{other}, written)
		},
	}
	changes := map[string]func(context.Context) error{
		"a backfill write": func(ctx context.Context) error {
			return db.WriteSeries(
				ctx,
				key,
				2*telem.SecondTS,
				telem.NewSeriesSecondsTSV(2, 3),
			)
		},
		"a delete": func(ctx context.Context) error {
			return db.DeleteTimeRange(ctx, []cesium.ChannelKey{key}, written)
		},
	}
	for changeName, change := range changes {
		for callName, call := range calls {
			It("Should finish "+changeName+" whose handler makes "+callName, func(
				ctx SpecContext,
			) {
				called := make(chan error, 1)
				DeferCleanup(db.OnHistoryChange(
					func(ctx context.Context, c cesium.HistoryChange) {
						if c.Key != key {
							return
						}
						select {
						case called <- call(ctx, c):
						default:
						}
					},
				))
				done := make(chan error, 1)
				go func() { done <- change(ctx) }()
				Eventually(done, 5*time.Second).Should(Receive(Succeed()))
				Eventually(called).Should(Receive(Succeed()))
			})
		}
	}
})
