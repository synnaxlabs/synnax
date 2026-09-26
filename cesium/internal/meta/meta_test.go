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
	jsonv2 "encoding/json/v2"
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
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Meta", func() {
	for fsName, openFS := range FileSystems {
		var fs fs.FS
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

			Describe("Legacy meta names", func() {
				// A database written before IsIndex, Virtual and Concurrency carried
				// json tags stored them under their Go field names. The meta codec
				// matches names case-insensitively so those databases still open.
				Specify("Should clear the index flag on a virtual channel", func(
					ctx SpecContext,
				) {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					f := MustSucceed(subFs.Open("meta.json", os.O_CREATE|os.O_WRONLY))
					// Validate rejects this pair, so the record only stays readable
					// because the migration runs first.
					legacy := []byte(`{"name":"Faraday","data_type":"timestamp",` +
						`"key":` + strconv.Itoa(int(key)) +
						`,"index":0,"IsIndex":true,"Virtual":true,"version":2}`)
					Expect(f.Write(legacy)).To(Equal(len(legacy)))
					Expect(f.Close()).To(Succeed())

					ch := MustSucceed(meta.Open(
						ctx,
						subFs,
						channel.Channel{Key: key},
						json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true)),
					))
					Expect(ch.Virtual).To(BeTrue())
					Expect(ch.IsIndex).To(BeFalse())
				})

				Specify("Should rewrite a legacy file under the current names", func(
					ctx SpecContext,
				) {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					f := MustSucceed(subFs.Open("meta.json", os.O_CREATE|os.O_WRONLY))
					legacy := []byte(`{"name":"Faraday","data_type":"timestamp",` +
						`"key":` + strconv.Itoa(int(key)) +
						`,"index":0,"IsIndex":true,"version":2}`)
					Expect(f.Write(legacy)).To(Equal(len(legacy)))
					Expect(f.Close()).To(Succeed())

					codec := json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true))
					ch := MustSucceed(meta.Open(
						ctx, subFs, channel.Channel{Key: key}, codec,
					))
					Expect(ch.IsIndex).To(BeTrue())
					Expect(ch.Version).To(Equal(channel.VersionCurrent))

					r := MustSucceed(subFs.Open("meta.json", os.O_RDONLY))
					stored := make([]byte, 512)
					n := MustSucceed(r.Read(stored))
					Expect(r.Close()).To(Succeed())
					Expect(string(stored[:n])).To(ContainSubstring(`"is_index":true`))
					Expect(string(stored[:n])).NotTo(ContainSubstring(`"IsIndex"`))
				})

				Specify("Should read a meta.json written under Go field names", func(
					ctx SpecContext,
				) {
					key := GenerateChannelKey()
					subFs := MustSucceed(fs.Sub(strconv.Itoa(int(key))))
					f := MustSucceed(subFs.Open("meta.json", os.O_CREATE|os.O_WRONLY))
					legacy := []byte(`{"name":"Faraday","data_type":"int64",` +
						`"key":` + strconv.Itoa(int(key)) +
						`,"index":0,"IsIndex":true,"Virtual":false,` +
						`"Concurrency":1,"version":1}`)
					Expect(f.Write(legacy)).To(Equal(len(legacy)))
					Expect(f.Close()).To(Succeed())

					ch := MustSucceed(meta.Read(
						ctx,
						subFs,
						json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true)),
					))
					Expect(ch.Key).To(Equal(key))
					Expect(ch.IsIndex).To(BeTrue())
					Expect(ch.Virtual).To(BeFalse())
				})
			})

			Describe("Impossible meta configurations", func() {
				DescribeTable(
					"meta configs",
					func(ctx SpecContext, ch channel.Channel, badField string) {
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
								json.Codec,
							),
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

			It(
				"Should not delete the original file if an error occurs while encoding",
				func(ctx SpecContext) {
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
					Expect(ch2.Virtual).To(BeTrue())
					Expect(ch2.DataType).To(Equal(telem.Int64T))
				},
			)
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
