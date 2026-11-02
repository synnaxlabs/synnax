// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("OnHistoryChange", func() {
	It("Should report changes until the handler disconnects", func(ctx SpecContext) {
		dist := testutil.NewNode(ctx)
		chs := MustSucceed(dist.Channel.Create(ctx, []channel.Channel{{
			Name:     "Hopper",
			DataType: telem.TimestampT,
			IsIndex:  true,
		}}))
		key := chs[0].Key()
		write := func(stamps ...telem.TimeStamp) {
			GinkgoHelper()
			w := MustSucceed(dist.Framer.OpenWriter(ctx, framer.WriterConfig{
				Keys:  channel.Keys{key},
				Start: stamps[0] * telem.SecondTS,
			}))
			series := telem.NewSeriesSecondsTSV(stamps...)
			Expect(w.Write(frame.NewUnary(key, series))).To(BeTrue())
			MustSucceed(w.Commit())
			Expect(w.Close()).To(Succeed())
		}
		var changes []framer.HistoryChange
		disconnect := dist.Framer.OnHistoryChange(
			func(_ context.Context, c framer.HistoryChange) {
				changes = append(changes, c)
			},
		)
		write(20, 21)
		write(10, 11)
		disconnect()
		write(5, 6)
		Expect(changes).To(Equal([]framer.HistoryChange{{
			Key:       key,
			TimeRange: (10 * telem.SecondTS).Range(11*telem.SecondTS + 1),
		}}))
	})
})
