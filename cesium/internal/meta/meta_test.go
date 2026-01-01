// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package meta_test

import (
	"context"
	"io"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Meta", func() {
	for fsName, makeFS := range FileSystems {
		var (
			ctx     context.Context
			fs      fs.FS
			cleanUp func() error
			codec   binary.Codec
		)
		BeforeEach(func() {
			ctx = context.Background()
			fs, cleanUp = makeFS()
			codec = &binary.JSONCodec{}
		})
		AfterEach(func() { Expect(cleanUp()).To(Succeed()) })
		Context("FS: "+fsName, func() {
			Describe("Corrupted Meta file", func() {
				Specify("Corrupted meta.json", func() {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					ch := MustSucceed(meta.Open(
						ctx,
						subFs,
						channel.Channel{
							Key:      key,
							Name:     "Faraday",
							Virtual:  true,
							DataType: telem.Int64T,
						},
						codec,
					))
					Expect(ch.Key).To(Equal(key))

					f := MustSucceed(subFs.Open("meta.json", os.O_WRONLY))
					Expect(f.Write([]byte("heheheha"))).To(Equal(8))
					Expect(f.Close()).To(Succeed())

					Expect(meta.Open(ctx, subFs, ch, codec)).Error().
						To(MatchError(ContainSubstring(
							"error decoding meta in folder for channel %d",
							key,
						)))
				})
			})

			Describe("Impossible meta configurations", func() {
				DescribeTable("meta configs", func(ch channel.Channel, badField string) {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					createdChannel := MustSucceed(
						meta.Open(
							ctx,
							subFs,
							channel.Channel{
								Key:      key,
								Name:     "John",
								Virtual:  true,
								DataType: telem.Int64T,
							},
							codec),
					)
					Expect(createdChannel.Key).To(Equal(key))

					f := MustSucceed(subFs.Open("meta.json", os.O_WRONLY))
					Expect(codec.EncodeStream(ctx, f, ch)).To(Succeed())
					Expect(f.Close()).To(Succeed())

					Expect(meta.Open(ctx, subFs, ch, codec)).Error().
						To(MatchError(ContainSubstring(badField)))
				},
					Entry(
						"datatype not set",
						channel.Channel{
							Key: GenerateChannelKey(), Name: "Wick", Virtual: true,
						},
						"data_type",
					),
					Entry(
						"virtual indexed",
						channel.Channel{
							Key:      GenerateChannelKey(),
							Virtual:  true,
							Name:     "Snow?",
							Index:    500000000,
							DataType: telem.Int64T,
						},
						"virtual channel cannot be indexed",
					),
					Entry(
						"index not type timestamp",
						channel.Channel{
							Key:      GenerateChannelKey(),
							Name:     "Mulaney?",
							IsIndex:  true,
							DataType: telem.Float32T,
						},
						"index channel must be of type timestamp",
					),
				)
			})

			It("Should not delete the original file if an error occurs while encoding", func() {
				key := GenerateChannelKey()
				subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
				ch := MustSucceed(meta.Open(
					ctx,
					subFs,
					channel.Channel{
						Key:      key,
						Name:     "Faraday",
						Virtual:  true,
						DataType: telem.Int64T,
					},
					codec,
				))
				Expect(ch.Key).To(Equal(key))

				Expect(meta.Create(ctx, subFs, &brokenCodec{}, ch)).Error().
					To(MatchError(encodingError))
				Expect(subFs.Exists("meta.json")).To(BeTrue())

				Expect(meta.Read(ctx, subFs, &brokenCodec{})).Error().
					To(MatchError(encodingError))
				Expect(subFs.Exists("meta.json")).To(BeTrue())
				Expect(subFs.Exists("meta.json.tmp")).To(BeFalse())

				ch2 := MustSucceed(meta.Read(ctx, subFs, codec))
				Expect(ch2.Key).To(Equal(key))
				Expect(ch2.Name).To(Equal("Faraday"))
				Expect(ch2.Virtual).To(BeTrue())
				Expect(ch2.DataType).To(Equal(telem.Int64T))

			})
		})
	}
})

type brokenCodec struct{}

var _ binary.Codec = (*brokenCodec)(nil)

var encodingError = errors.New("broken codec")

func (b *brokenCodec) Encode(context.Context, any) ([]byte, error) {
	return nil, encodingError
}

func (b *brokenCodec) EncodeStream(context.Context, io.Writer, any) error {
	return encodingError
}

func (b *brokenCodec) Decode(context.Context, []byte, any) error {
	return encodingError
}

func (b *brokenCodec) DecodeStream(context.Context, io.Reader, any) error {
	return encodingError
}
