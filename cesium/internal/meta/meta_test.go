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
	"fmt"
	"io"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Meta", func() {
	for fsName, openFS := range FileSystems {
		var (
			fs fs.FS
		)
		BeforeEach(func() {
			fs = openFS()
		})
		Context("FS: "+fsName, func() {
			Describe("Corrupted Meta file", func() {
				Specify("Corrupted meta.json", func(ctx SpecContext) {
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
						json.Codec,
					))
					Expect(ch.Key).To(Equal(key))

					f := MustSucceed(subFs.Open("meta.json", os.O_WRONLY))
					Expect(f.Write([]byte("heheheha"))).To(Equal(8))
					Expect(f.Close()).To(Succeed())

					Expect(meta.Open(ctx, subFs, ch, json.Codec)).Error().
						To(MatchError(ContainSubstring(
							"error decoding meta in folder for channel %d",
							key,
						)))
				})
			})

			Describe("Impossible meta configurations", func() {
				DescribeTable("meta configs", func(ctx SpecContext, ch channel.Channel, badField string) {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					createdChannel := MustSucceed(
						meta.Open(
							ctx,
							subFs,
							channel.Channel{
								Key:      key,
								Name:     "John",
								IsIndex:  true,
								DataType: telem.TimeStampT,
							},
							json.Codec),
					)
					Expect(createdChannel.Key).To(Equal(key))

					f := MustSucceed(subFs.Open("meta.json", os.O_WRONLY))
					Expect(json.Codec.EncodeStream(ctx, f, ch)).To(Succeed())
					Expect(f.Close()).To(Succeed())

					Expect(meta.Open(ctx, subFs, ch, json.Codec)).Error().
						To(MatchError(ContainSubstring(badField)))
				},
					Entry(
						"datatype not set",
						channel.Channel{
							Key: GenerateChannelKey(), Name: "Wick", IsIndex: true,
						},
						"data_type",
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

			It("Should not delete the original file if an error occurs while encoding", func(ctx SpecContext) {
				key := GenerateChannelKey()
				subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
				ch := MustSucceed(meta.Open(
					ctx,
					subFs,
					channel.Channel{
						Key:      key,
						Name:     "Faraday",
						IsIndex:  true,
						DataType: telem.TimeStampT,
					},
					json.Codec,
				))
				Expect(ch.Key).To(Equal(key))

				Expect(meta.Create(ctx, subFs, &brokenCodec{}, ch)).Error().
					To(MatchError(errEncoding))
				Expect(subFs.Exists("meta.json")).To(BeTrue())

				Expect(meta.Read(ctx, subFs, &brokenCodec{})).Error().
					To(MatchError(errEncoding))
				Expect(subFs.Exists("meta.json")).To(BeTrue())
				Expect(subFs.Exists("meta.json.tmp")).To(BeFalse())

				ch2 := MustSucceed(meta.Read(ctx, subFs, json.Codec))
				Expect(ch2.Key).To(Equal(key))
				Expect(ch2.Name).To(Equal("Faraday"))
				Expect(ch2.IsIndex).To(BeTrue())
				Expect(ch2.DataType).To(Equal(telem.TimeStampT))

			})

			Describe("ReadVirtualFlag", func() {
				It("Should report true for a meta file persisted for a virtual channel by a previous version", func(ctx SpecContext) {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					f := MustSucceed(subFs.Open("meta.json", os.O_CREATE|os.O_WRONLY))
					MustSucceed(f.Write(fmt.Appendf(nil,
						`{"key":%d,"name":"legacy","data_type":"int64","virtual":true,"version":2}`,
						key,
					)))
					Expect(f.Close()).To(Succeed())
					Expect(meta.ReadVirtualFlag(ctx, subFs, json.Codec)).To(BeTrue())
				})

				It("Should report false when the meta file does not exist", func(ctx SpecContext) {
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(GenerateChannelKey()))))
					Expect(meta.ReadVirtualFlag(ctx, subFs, json.Codec)).To(BeFalse())
				})

				It("Should report false for a stored channel's meta file", func(ctx SpecContext) {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					MustSucceed(meta.Open(
						ctx,
						subFs,
						channel.Channel{
							Key:      key,
							Name:     "stored",
							IsIndex:  true,
							DataType: telem.TimeStampT,
						},
						json.Codec,
					))
					Expect(meta.ReadVirtualFlag(ctx, subFs, json.Codec)).To(BeFalse())
				})
			})
		})
	}
})

type brokenCodec struct{}

var _ encoding.Codec = (*brokenCodec)(nil)

var errEncoding = errors.New("broken json.Codec")

func (b *brokenCodec) Encode(context.Context, any) ([]byte, error) {
	return nil, errEncoding
}

func (b *brokenCodec) EncodeStream(context.Context, io.Writer, any) error {
	return errEncoding
}

func (b *brokenCodec) Decode(context.Context, []byte, any) error {
	return errEncoding
}

func (b *brokenCodec) DecodeStream(context.Context, io.Reader, any) error {
	return errEncoding
}
