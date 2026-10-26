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
	"bytes"
	"context"
	"fmt"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// writeSizes records the size of every write it receives.
type writeSizes []int

// Write implements io.Writer.
func (w *writeSizes) Write(p []byte) (int, error) {
	*w = append(*w, len(p))
	return len(p), nil
}

// csvRows returns the rows of a CSV body that follow its header.
func csvRows(body []byte) []string {
	return strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")[1:]
}

var _ = Describe("CSV", func() {
	Describe("CSVEncoder", func() {
		It("Should emit the CSV media type", func() {
			Expect(framer.CSVEncoder.ContentType()).To(Equal("text/csv"))
		})

		It("Should write a header and a row per sample", func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.Float32T, telem.Int64T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2, 3),
				channels,
				telem.NewSeriesV[float32](1.5, 2.25, 3),
				telem.NewSeriesV[int64](10, 20, 30),
			)
			b := MustSucceed(
				framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(string(b)).To(Equal(fmt.Sprintf(
				"%s,%s,%s\n1000000000,1.5,10\n2000000000,2.25,20\n3000000000,3,30\n",
				index.Name,
				channels[0].Name,
				channels[1].Name,
			)))
		})

		It("Should quote a value holding a comma", func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.StringT)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2),
				channels,
				telem.NewSeriesV("ok", "a,b"),
			)
			b := MustSucceed(
				framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(string(b)).To(Equal(fmt.Sprintf(
				"%s,%s\n1000000000,ok\n2000000000,\"a,b\"\n",
				index.Name,
				channels[0].Name,
			)))
		})

		It("Should double the quotes inside a quoted value", func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.StringT, telem.Int64T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2),
				channels,
				telem.NewSeriesV(`say "hi"`, "line\nbreak"),
				telem.NewSeriesV[int64](7, 8),
			)
			b := MustSucceed(
				framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(string(b)).To(Equal(fmt.Sprintf(
				"%s,%s,%s\n1000000000,\"say \"\"hi\"\"\",7\n2000000000,\"line\nbreak\",8\n",
				index.Name,
				channels[0].Name,
				channels[1].Name,
			)))
		})

		It("Should quote an empty value to tell it apart from a missing one", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.StringT, telem.BytesT)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1),
				channels,
				telem.NewSeriesV(""),
				telem.NewSeriesV([]byte{}),
			)
			b := MustSucceed(
				framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(string(b)).To(Equal(fmt.Sprintf(
				"%s,%s,%s\n1000000000,\"\",\"\"\n",
				index.Name,
				channels[0].Name,
				channels[1].Name,
			)))
		})

		It("Should write a UUID in canonical form and bytes as base64", func(
			ctx SpecContext,
		) {
			id := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
			index, channels := createIndexed(ctx, telem.UUIDT, telem.BytesT)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1),
				channels,
				telem.NewSeriesV(id),
				telem.NewSeriesV([]byte{0, 255}),
			)
			b := MustSucceed(
				framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(string(b)).To(Equal(fmt.Sprintf(
				"%s,%s,%s\n1000000000,%s,AP8=\n",
				index.Name,
				channels[0].Name,
				channels[1].Name,
				id,
			)))
		})

		It("Should leave a group's cells empty at a timestamp it has no sample for",
			func(ctx SpecContext) {
				firstIndex, first := createIndexed(ctx, telem.Float32T)
				secondIndex, second := createIndexed(ctx, telem.Float32T)
				write(
					ctx,
					firstIndex,
					telem.NewSeriesSecondsTSV(1, 3),
					first,
					telem.NewSeriesV[float32](1, 2),
				)
				write(
					ctx,
					secondIndex,
					telem.NewSeriesSecondsTSV(2),
					second,
					telem.NewSeriesV[float32](3),
				)
				channels := append(first, second...)
				indexes := []svcchannel.Channel{firstIndex, secondIndex}
				b := MustSucceed(
					framer.CSVEncoder.Encode(
						ctx,
						response(channels, indexes, svcframer.IteratorConfig{}),
					),
				)
				Expect(string(b)).To(Equal(fmt.Sprintf(
					"%s,%s,%s,%s\n"+
						"1000000000,1,,\n"+
						",,2000000000,3\n"+
						"3000000000,2,,\n",
					firstIndex.Name,
					first[0].Name,
					secondIndex.Name,
					second[0].Name,
				)))
			},
		)

		It("Should fail a read that leaves out the index of a channel", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1),
				channels,
				telem.NewSeriesV[float32](1),
			)
			Expect(
				framer.CSVEncoder.Encode(
					ctx,
					response(channels, nil, svcframer.IteratorConfig{}),
				),
			).Error().
				To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(channels[0].Name)),
				))
		})

		It("Should fail a limited read before the first byte", func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				index,
				telem.NewSeriesSecondsTSV(1, 2, 3, 4),
				channels,
				telem.NewSeriesV[float32](1, 2, 3, 4),
			)
			res := response(
				channels,
				[]svcchannel.Channel{index},
				svcframer.IteratorConfig{
					Reduction: telem.Reduction{
						Variant: telem.LimitReduction{
							Aggregation: telem.AggregationDecimate,
							PointLimit:  2,
						},
					},
				},
			)
			var buf bytes.Buffer
			Expect(framer.CSVEncoder.EncodeStream(ctx, &buf, res)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("cannot take a limit reduction")),
			))
			Expect(buf.Len()).To(BeZero())
		})

		It("Should read each column through its own iterator", func(ctx SpecContext) {
			firstIndex, first := createIndexed(ctx, telem.Float32T)
			secondIndex, second := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				firstIndex,
				telem.NewSeriesSecondsTSV(1, 2),
				first,
				telem.NewSeriesV[float32](1, 2),
			)
			write(
				ctx,
				secondIndex,
				telem.NewSeriesSecondsTSV(10, 11),
				second,
				telem.NewSeriesV[float32](10, 11),
			)
			res := response(
				append(first, second...),
				[]svcchannel.Channel{firstIndex, secondIndex},
				svcframer.IteratorConfig{ChunkSize: 1},
			)
			var opened []svcchannel.Keys
			open := res.OpenIterator
			res.OpenIterator = func(
				ctx context.Context,
				keys svcchannel.Keys,
			) (*svcframer.Iterator, error) {
				opened = append(opened, keys)
				return open(ctx, keys)
			}
			b := MustSucceed(framer.CSVEncoder.Encode(ctx, res))
			Expect(opened).To(ConsistOf(
				svcchannel.Keys{firstIndex.Key()},
				svcchannel.Keys{first[0].Key()},
				svcchannel.Keys{secondIndex.Key()},
				svcchannel.Keys{second[0].Key()},
			))
			Expect(string(b)).To(Equal(fmt.Sprintf(
				"%s,%s,%s,%s\n"+
					"1000000000,1,,\n"+
					"2000000000,2,,\n"+
					",,10000000000,10\n"+
					",,11000000000,11\n",
				firstIndex.Name,
				first[0].Name,
				secondIndex.Name,
				second[0].Name,
			)))
		})

		It("Should order the rows of groups that advance at different rates", func(
			ctx SpecContext,
		) {
			var (
				channels []svcchannel.Channel
				indexes  []svcchannel.Channel
			)
			for _, seconds := range [][]float64{
				{1, 2, 3, 4, 5, 6},
				{1.5, 3.5, 5.5},
				{0.5, 6.5},
			} {
				index, group := createIndexed(ctx, telem.Float32T)
				timestamps := make([]telem.TimeStamp, len(seconds))
				values := make([]float32, len(seconds))
				for i, s := range seconds {
					timestamps[i] = telem.TimeStamp(s * float64(telem.Second))
					values[i] = float32(s)
				}
				write(
					ctx,
					index,
					telem.NewSeriesV(timestamps...),
					group,
					telem.NewSeriesV(values...),
				)
				channels = append(channels, group...)
				indexes = append(indexes, index)
			}
			b := MustSucceed(
				framer.CSVEncoder.Encode(
					ctx,
					response(channels, indexes, svcframer.IteratorConfig{ChunkSize: 2}),
				),
			)
			Expect(csvRows(b)).To(Equal([]string{
				",,,,500000000,0.5",
				"1000000000,1,,,,",
				",,1500000000,1.5,,",
				"2000000000,2,,,,",
				"3000000000,3,,,,",
				",,3500000000,3.5,,",
				"4000000000,4,,,,",
				"5000000000,5,,,,",
				",,5500000000,5.5,,",
				"6000000000,6,,,,",
				",,,,6500000000,6.5",
			}))
		})

		It("Should leave a cell empty where its channel has no sample", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T, telem.StringT)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3, 4), nil)
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*3,
				telem.NewSeriesV[float32](1.5, 2.5),
			)
			writeData(ctx, channels[1], telem.SecondTS, telem.NewSeriesV("a", "b"))
			b := MustSucceed(
				framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(string(b)).To(Equal(fmt.Sprintf(
				"%s,%s,%s\n"+
					"1000000000,,a\n"+
					"2000000000,,b\n"+
					"3000000000,1.5,\n"+
					"4000000000,2.5,\n",
				index.Name,
				channels[0].Name,
				channels[1].Name,
			)))
		})

		It("Should hold a channel's samples until its index reaches them", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6), nil)
			writeData(ctx, channels[0], telem.SecondTS, telem.NewSeriesV[float32](1, 2))
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*5,
				telem.NewSeriesV[float32](5, 6),
			)
			b := MustSucceed(
				framer.CSVEncoder.Encode(
					ctx,
					response(
						channels,
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{ChunkSize: 2},
					),
				),
			)
			Expect(csvRows(b)).To(Equal([]string{
				"1000000000,1",
				"2000000000,2",
				"3000000000,",
				"4000000000,",
				"5000000000,5",
				"6000000000,6",
			}))
		})

		It("Should pair a channel's samples with each index domain that holds them",
			func(ctx SpecContext) {
				index, channels := createIndexed(ctx, telem.Float32T)
				write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3), nil)
				write(ctx, index, telem.NewSeriesSecondsTSV(10, 11, 12), nil)
				writeData(
					ctx,
					channels[0],
					telem.SecondTS*2,
					telem.NewSeriesV[float32](2, 3),
				)
				writeData(
					ctx,
					channels[0],
					telem.SecondTS*10,
					telem.NewSeriesV[float32](10, 11),
				)
				b := MustSucceed(
					framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
				)
				Expect(csvRows(b)).To(Equal([]string{
					"1000000000,",
					"2000000000,2",
					"3000000000,3",
					"10000000000,10",
					"11000000000,11",
					"12000000000,",
				}))
			},
		)

		It("Should pair each group's samples with its own index", func(
			ctx SpecContext,
		) {
			firstIndex, first := createIndexed(ctx, telem.Float32T)
			secondIndex, second := createIndexed(ctx, telem.Float32T)
			write(ctx, firstIndex, telem.NewSeriesSecondsTSV(1, 3, 5), nil)
			write(ctx, secondIndex, telem.NewSeriesSecondsTSV(2, 4, 6), nil)
			writeData(ctx, first[0], telem.SecondTS*3, telem.NewSeriesV[float32](3, 5))
			writeData(ctx, second[0], telem.SecondTS*2, telem.NewSeriesV[float32](2))
			channels := append(first, second...)
			indexes := []svcchannel.Channel{firstIndex, secondIndex}
			b := MustSucceed(
				framer.CSVEncoder.Encode(
					ctx,
					response(channels, indexes, svcframer.IteratorConfig{ChunkSize: 1}),
				),
			)
			Expect(csvRows(b)).To(Equal([]string{
				"1000000000,,,",
				",,2000000000,2",
				"3000000000,3,,",
				",,4000000000,",
				"5000000000,5,,",
				",,6000000000,",
			}))
		})

		It("Should pair a downsampled channel's samples with its index", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6), nil)
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*2,
				telem.NewSeriesV[float32](2, 3, 4, 5, 6),
			)
			b := MustSucceed(
				framer.CSVEncoder.Encode(
					ctx,
					response(
						channels,
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{
							Reduction: telem.Reduction{
								Variant: telem.StrideReduction{Factor: 2},
							},
						},
					),
				),
			)
			Expect(csvRows(b)).To(Equal([]string{
				"1000000000,",
				"3000000000,3",
				"5000000000,5",
			}))
		})

		It("Should pair a calculated channel's samples with its index", func(
			ctx SpecContext,
		) {
			index, channels := createIndexed(ctx, telem.Float32T)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3, 4), nil)
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*3,
				telem.NewSeriesV[float32](3, 4),
			)
			calculated := svcchannel.Channel{
				Name:       UniqueChannelName(),
				DataType:   telem.Float32T,
				Expression: "return " + channels[0].Name + " * 2",
			}
			Expect(channelWriter.Create(ctx, &calculated)).To(Succeed())
			var calculatedIndex svcchannel.Channel
			Expect(channelSvc.NewRetrieve().
				Entry(&calculatedIndex).
				Where(svcchannel.MatchKeys(calculated.Index())).
				Exec(ctx, nil)).To(Succeed())
			b := MustSucceed(framer.CSVEncoder.Encode(
				ctx,
				read(ctx, calculatedIndex, calculated),
			))
			Expect(csvRows(b)).To(Equal([]string{"3000000000,6", "4000000000,8"}))
		})

		It("Should return an error when a sample has no index sample", func(
			ctx SpecContext,
		) {
			index, _ := createIndexed(ctx)
			write(ctx, index, telem.NewSeriesSecondsTSV(1, 2, 3), nil)
			write(ctx, index, telem.NewSeriesSecondsTSV(10, 11, 12), nil)
			otherIndex, channels := createIndexed(ctx, telem.Float32T)
			write(
				ctx,
				otherIndex,
				telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8),
				nil,
			)
			writeData(
				ctx,
				channels[0],
				telem.SecondTS*6,
				telem.NewSeriesV[float32](6, 7, 8),
			)
			misindexed := channels[0]
			misindexed.LocalIndex = index.LocalKey
			Expect(
				framer.CSVEncoder.Encode(
					ctx,
					response(
						[]svcchannel.Channel{misindexed},
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{},
					),
				),
			).Error().
				To(MatchError(fmt.Sprintf(
					"[api.Read] channel %s has a sample at alignment 0-5 with no index sample",
					misindexed.Name,
				),
				))
		})

		It("Should write a large read in bounded pieces", func(ctx SpecContext) {
			const samples = 20_000
			index, channels := createIndexed(ctx, telem.Int64T)
			timestamps := make([]telem.TimeStamp, samples)
			values := make([]int64, samples)
			for i := range samples {
				timestamps[i] = telem.TimeStamp(i+1) * telem.MillisecondTS
				values[i] = int64(i)
			}
			write(
				ctx,
				index,
				telem.NewSeriesV(timestamps...),
				channels,
				telem.NewSeriesV(values...),
			)
			var w writeSizes
			Expect(
				framer.CSVEncoder.EncodeStream(
					ctx,
					&w,
					response(
						channels,
						[]svcchannel.Channel{index},
						svcframer.IteratorConfig{ChunkSize: 1000},
					),
				),
			).To(Succeed())
			Expect(len(w)).To(BeNumerically(">", 2))
			Expect(w).To(HaveEach(BeNumerically("<", 70<<10)))
		})

		It("Should write only a header for a read with no data", func(ctx SpecContext) {
			index, channels := createIndexed(ctx, telem.Float32T)
			b := MustSucceed(
				framer.CSVEncoder.Encode(ctx, read(ctx, index, channels...)),
			)
			Expect(string(b)).
				To(Equal(fmt.Sprintf("%s,%s\n", index.Name, channels[0].Name)))
		})

		It("Should return an error when the value is not a read response", func(
			ctx SpecContext,
		) {
			var buf bytes.Buffer
			Expect(framer.CSVEncoder.EncodeStream(ctx, &buf, 42)).
				To(MatchError(ContainSubstring("cannot encode a value of type int")))
		})
	})
})
