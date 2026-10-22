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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/resource"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB Metadata Operations", func() {
	for fsName, openFS := range FileSystems {
		var (
			fs         xfs.FS
			indexDBfs  xfs.FS
			indexDBKey channel.Key
			indexDB    *unary.DB
			dataDBfs   xfs.FS
			dataDBKey  channel.Key
			dataDB     *unary.DB
		)
		Context("FS: "+fsName, func() {
			BeforeEach(func(ctx SpecContext) {
				fs = openFS()
				indexDBKey = GenerateChannelKey()
				indexDBfs = MustSucceed(fs.Sub("index"))
				indexDB = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        indexDBfs,
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      indexDBKey,
						Name:     "test",
						DataType: telem.TimestampT,
						IsIndex:  true,
					},
				}))
				dataDBKey = GenerateChannelKey()
				dataDBfs = MustSucceed(fs.Sub("data"))
				dataDB = MustSucceed(unary.Open(ctx, unary.Config{
					FS:        dataDBfs,
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      dataDBKey,
						Name:     "test",
						DataType: telem.Int64T,
						IsIndex:  false,
						Index:    indexDBKey,
					},
				}))
			})

			AfterEach(func() {
				Expect(indexDB.Close()).To(Succeed())
				Expect(dataDB.Close()).To(Succeed())
			})

			Describe("RenameChannelInMeta", func() {
				It("Should rename the channel and persist it", func(ctx SpecContext) {
					Expect(dataDB.RenameChannelInMeta(ctx, "new_name")).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, dataDBfs, json.Codec))
					Expect(ch.Name).To(Equal("new_name"))
				})

				It(
					"Should be a no-op when the name is the same",
					func(ctx SpecContext) {
						Expect(dataDB.RenameChannelInMeta(ctx, "test")).To(Succeed())
						ch := MustSucceed(meta.Read(ctx, dataDBfs, json.Codec))
						Expect(ch.Name).To(Equal("test"))
					},
				)
			})

			Describe("Size", func() {
				It("Should return zero for an empty database", func() {
					Expect(indexDB.Size()).To(Equal(telem.Size(0)))
					Expect(dataDB.Size()).To(Equal(telem.Size(0)))
				})

				It(
					"Should return the correct size after writing data",
					func(ctx SpecContext) {
						w, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{
							Start:   telem.TimeStamp(0),
							Subject: control.Subject{Key: "size_test"},
						}))
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(0, 1, 2, 3, 4)))
						MustSucceed(w.Commit(ctx))
						MustSucceed(w.Close())

						expectedSize := telem.Size(5 * telem.TimestampT.Density())
						Expect(indexDB.Size()).To(Equal(expectedSize))
					},
				)

				It(
					"Should accumulate size across multiple writes",
					func(ctx SpecContext) {
						w, _ := MustSucceed2(indexDB.OpenWriter(ctx, unary.WriterConfig{
							Start:   telem.TimeStamp(0),
							Subject: control.Subject{Key: "size_test"},
						}))
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(0, 1, 2)))
						MustSucceed(w.Commit(ctx))
						MustSucceed(w.Write(telem.NewSeriesSecondsTSV(3, 4)))
						MustSucceed(w.Commit(ctx))
						MustSucceed(w.Close())

						expectedSize := telem.Size(5 * telem.TimestampT.Density())
						Expect(indexDB.Size()).To(Equal(expectedSize))
					},
				)
			})
		})
	}

	Describe("Close", func() {
		var db *unary.DB
		BeforeEach(func(ctx SpecContext) {
			db = MustSucceed(unary.Open(ctx, unary.Config{
				FS:        xfs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     "test",
					DataType: telem.TimestampT,
					IsIndex:  true,
				},
			}))
		})

		It(
			"Should return an error when methods are called on a closed DB",
			func(ctx SpecContext) {
				Expect(db.Close()).To(Succeed())
				Expect(
					db.RenameChannelInMeta(ctx, "new_name"),
				).To(MatchError(unary.ErrDBClosed))
				Expect(
					db.Delete(ctx, telem.TimeRange{}),
				).To(MatchError(unary.ErrDBClosed))
				Expect(db.GarbageCollect(ctx)).To(MatchError(unary.ErrDBClosed))
				Expect(
					db.HasDataFor(ctx, telem.TimeRangeMax),
				).Error().
					To(MatchError(unary.ErrDBClosed))
				Expect(
					db.OpenWriter(ctx, unary.WriterConfig{}),
				).Error().
					To(MatchError(unary.ErrDBClosed))
				Expect(
					db.OpenIterator(unary.IteratorConfig{}),
				).Error().
					To(MatchError(unary.ErrDBClosed))
			},
		)

		It(
			"Should return an error when a DB is closed while writers are still accessing it",
			func(ctx SpecContext) {
				db := MustSucceed(unary.Open(ctx, unary.Config{
					FS:        xfs.NewMem(),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "test",
						DataType: telem.TimestampT,
						IsIndex:  true,
					},
				}))
				writer, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
					Subject: control.Subject{Key: "string"},
				}))
				Expect(db.Close()).To(MatchError(resource.ErrOpen))
				_ = MustSucceed(writer.Close())
				Expect(db.Close()).To(Succeed())
			},
		)
	})
})
