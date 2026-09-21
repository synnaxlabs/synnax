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
	"fmt"
	"strings"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// writeSizes records the size of every write it receives.
type writeSizes []int

// Write implements io.Writer.
func (w *writeSizes) Write(p []byte) (int, error) {
	*w = append(*w, len(p))
	return len(p), nil
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
				iter := MustSucceed(framerSvc.OpenIterator(
					ctx,
					svcframer.IteratorConfig{
						Keys: svcchannel.KeysFromChannels(
							append(channels, indexes...),
						),
						Bounds: readBounds,
					},
				))
				b := MustSucceed(framer.CSVEncoder.Encode(ctx, framer.ReadResponse{
					Iterator: iter,
					Channels: channels,
					Indexes:  indexes,
				}))
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
			iter := MustSucceed(framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
				Keys:      svcchannel.KeysFromChannels(append(channels, indexes...)),
				Bounds:    readBounds,
				ChunkSize: 2,
			}))
			b := MustSucceed(framer.CSVEncoder.Encode(ctx, framer.ReadResponse{
				Iterator: iter,
				Channels: channels,
				Indexes:  indexes,
			}))
			rows := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")[1:]
			Expect(rows).To(Equal([]string{
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
			iter := MustSucceed(framerSvc.OpenIterator(ctx, svcframer.IteratorConfig{
				Keys:      svcchannel.KeysFromChannels(append(channels, index)),
				Bounds:    readBounds,
				ChunkSize: 1000,
			}))
			var w writeSizes
			Expect(framer.CSVEncoder.EncodeStream(ctx, &w, framer.ReadResponse{
				Iterator: iter,
				Channels: channels,
				Indexes:  []svcchannel.Channel{index},
			})).To(Succeed())
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
