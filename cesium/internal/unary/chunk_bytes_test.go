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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/encoding/json"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Chunk bytes", func() {
	for fsName, openFS := range FileSystems {
		Context("FS: "+fsName, func() {
			var idx, data *unary.DB
			BeforeEach(func(ctx SpecContext) {
				fs := openFS()
				idx = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("idx")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "idx",
						DataType: telem.TimestampT,
						IsIndex:  true,
					},
				}))
				data = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        MustSucceed(fs.Sub("data")),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "data",
						DataType: telem.StringT,
						Index:    idx.Channel().Key,
					},
				}))
				data.SetIndex(idx.Index())
				DeferCleanup(func() {
					Expect(data.Close()).To(Succeed())
					Expect(idx.Close()).To(Succeed())
				})
			})

			// write writes count 2-character strings, prefixed by prefix, to a new
			// domain at one sample per second from start. Each sample takes 6 bytes.
			write := func(ctx SpecContext, start telem.TimeStamp, prefix string, count int) {
				stamps := make([]telem.TimeStamp, count)
				values := make([]string, count)
				for i := range count {
					stamps[i] = start + telem.TimeStamp(i)*telem.SecondTS
					values[i] = fmt.Sprintf("%s%d", prefix, i)
				}
				Expect(unary.Write(ctx, idx, start, telem.NewSeriesV(stamps...))).
					To(Succeed())
				Expect(unary.Write(ctx, data, start, telem.NewSeriesV(values...))).
					To(Succeed())
			}

			// chunks reads every chunk of an automatic span, forward or backward, with
			// the given chunk bytes.
			chunks := func(ctx SpecContext, forward bool, chunkBytes telem.Size) [][]string {
				iter := MustSucceed(data.OpenIterator(unary.IteratorConfig{
					Bounds:     telem.TimeRangeMax,
					ChunkBytes: chunkBytes,
				}))
				defer func() { Expect(iter.Close()).To(Succeed()) }()
				var out [][]string
				seek, move := iter.SeekLast, iter.Prev
				if forward {
					seek, move = iter.SeekFirst, iter.Next
				}
				Expect(seek(ctx)).To(BeTrue())
				for move(ctx, unary.AutoSpan) {
					var chunk []string
					for _, s := range iter.Value().RawSeries() {
						chunk = append(chunk, s.Unmarshal[string]()...)
					}
					out = append(out, chunk)
				}
				Expect(iter.Error()).ToNot(HaveOccurred())
				return out
			}

			It(
				"should end each forward chunk at the chunk bytes",
				func(ctx SpecContext) {
					write(ctx, 10*telem.SecondTS, "a", 10)
					Expect(chunks(ctx, true, 18)).To(Equal([][]string{
						{"a0", "a1", "a2"},
						{"a3", "a4", "a5"},
						{"a6", "a7", "a8"},
						{"a9"},
					}))
				},
			)

			It(
				"should end each backward chunk at the chunk bytes",
				func(ctx SpecContext) {
					write(ctx, 10*telem.SecondTS, "a", 10)
					Expect(chunks(ctx, false, 18)).To(Equal([][]string{
						{"a7", "a8", "a9"},
						{"a4", "a5", "a6"},
						{"a1", "a2", "a3"},
						{"a0"},
					}))
				},
			)

			It("should carry the chunk bytes across domains", func(ctx SpecContext) {
				write(ctx, 10*telem.SecondTS, "a", 5)
				write(ctx, 100*telem.SecondTS, "b", 5)
				Expect(chunks(ctx, true, 24)).To(Equal([][]string{
					{"a0", "a1", "a2", "a3"},
					{"a4", "b0", "b1", "b2"},
					{"b3", "b4"},
				}))
			})

			It(
				"should hold one sample larger than the chunk bytes",
				func(ctx SpecContext) {
					write(ctx, 10*telem.SecondTS, "a", 3)
					Expect(
						chunks(ctx, true, 1),
					).To(Equal([][]string{{"a0"}, {"a1"}, {"a2"}}))
				},
			)
		})
	}
})
