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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

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
