// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = DescribeTableSubtree(
	"Check: a commit whose index another writer holds",
	func(pointerLog bool) {
		It("Should commit none of the data until the index commits its stamps",
			func(ctx SpecContext) {
				host := newNode(ctx, pointerLog)
				idx := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:     "held_index_time",
					IsIndex:  true,
					DataType: telem.TimestampT,
				}}))[0]
				data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{{
					Name:       "held_index_data",
					DataType:   telem.Int64T,
					LocalIndex: idx.LocalKey,
				}}))[0]
				keys := channel.Keys{idx.Key(), data.Key()}
				open := func(
					key string,
					authority control.Authority,
					keys channel.Keys,
				) *writer.Writer {
					GinkgoHelper()
					return MustSucceed(host.Framer.OpenWriter(ctx, writer.Config{
						ControlSubject:   control.Subject{Key: key},
						Keys:             keys,
						Start:            10 * telem.SecondTS,
						Authorities:      []control.Authority{authority},
						EnableAutoCommit: new(false),
						Sync:             new(true),
					}))
				}
				// lengths returns the number of stored samples of the index and the
				// data.
				lengths := func() (index, values int64) {
					GinkgoHelper()
					iter := MustSucceed(host.Framer.OpenIterator(ctx, iterator.Config{
						Keys:   keys,
						Bounds: telem.TimeRangeMax,
					}))
					for ok := iter.SeekFirst(); ok && iter.Next(telem.TimeSpanMax); {
						index += iter.Value().Get(keys[0]).Len()
						values += iter.Value().Get(keys[1]).Len()
					}
					Expect(iter.Close()).To(Succeed())
					return index, values
				}
				first := open("first", 100, keys)
				Expect(first.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12, 13),
					telem.NewSeries([]int64{1, 2, 3, 4}),
				}))).To(BeTrue())
				second := open("second", 200, channel.Keys{keys[0]})
				_, authorized := MustSucceed2(first.Commit())
				Expect(authorized).To(BeFalse())
				// The data waits for its index, so a read finds no sample without its
				// stamp.
				index, values := lengths()
				Expect(index).To(BeZero())
				Expect(values).To(BeZero())
				Expect(second.Close()).To(Succeed())
				_, authorized = MustSucceed2(first.Commit())
				Expect(authorized).To(BeTrue())
				Expect(first.Close()).To(Succeed())
				index, values = lengths()
				Expect(index).To(Equal(int64(4)))
				Expect(values).To(Equal(int64(4)))
			})
	},
	Entry("old layout", false),
	Entry("pointer log", true),
)
