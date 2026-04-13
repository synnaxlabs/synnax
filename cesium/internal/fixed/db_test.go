// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fixed_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/fixed"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB Metadata Operations", func() {
	for fsName, makeFS := range fileSystems {
		var (
			fs         xfs.FS
			codec      = json.Codec
			cleanUp    func() error
			indexDBfs  xfs.FS
			indexDBKey channel.Key
			indexDB    *fixed.DB
			dataDBfs   xfs.FS
			dataDBKey  channel.Key
			dataDB     *fixed.DB
		)
		Context("FS: "+fsName, func() {
			BeforeEach(func(ctx SpecContext) {
				fs, cleanUp = makeFS()
				indexDBKey = testutil.GenerateChannelKey()
				indexDBfs = MustSucceed(fs.Sub("index"))
				indexDB = MustSucceed(fixed.Open(ctx, fixed.Config{
					FS:        indexDBfs,
					MetaCodec: codec,
					Channel: channel.Channel{
						Key:      indexDBKey,
						Name:     "test",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				}))
				dataDBKey = testutil.GenerateChannelKey()
				dataDBfs = MustSucceed(fs.Sub("data"))
				dataDB = MustSucceed(fixed.Open(ctx, fixed.Config{
					FS:        dataDBfs,
					MetaCodec: codec,
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
				Expect(cleanUp()).To(Succeed())
			})

			Describe("SetChannelKeyInMeta", func() {
				It("Should change both key and index when channel is an index", func(ctx SpecContext) {
					newKey := testutil.GenerateChannelKey()
					Expect(indexDB.SetChannelKeyInMeta(ctx, newKey)).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, indexDBfs, codec))
					Expect(ch.Key).To(Equal(newKey))
					Expect(ch.Index).To(Equal(newKey))
				})

				It("Should change only the key when channel is not an index", func(ctx SpecContext) {
					newKey := testutil.GenerateChannelKey()
					Expect(dataDB.SetChannelKeyInMeta(ctx, newKey)).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, dataDBfs, codec))
					Expect(ch.Key).To(Equal(newKey))
					Expect(ch.Index).To(Equal(indexDBKey))
				})
			})

			Describe("SetIndexKeyInMeta", func() {

				AfterEach(func() {
					Expect(indexDB.Close()).To(Succeed())
					Expect(dataDB.Close()).To(Succeed())
				})

				Describe("Index Channel", func() {
					It("Should set the index channel to a new key", func(ctx SpecContext) {
						newIndexKey := testutil.GenerateChannelKey()
						Expect(indexDB.Channel().Key).ToNot(Equal(newIndexKey))
						Expect(indexDB.SetChannelKeyInMeta(ctx, newIndexKey))
						Expect(indexDB.SetIndexKeyInMeta(ctx, newIndexKey)).To(Succeed())
						Expect(indexDB.Channel().Key).To(Equal(newIndexKey))
						Expect(indexDB.Channel().Index).To(Equal(newIndexKey))
					})

					It("Should return an error when attempting to set an index key that is different than the channel key", func(ctx SpecContext) {
						newIndexKey := testutil.GenerateChannelKey()
						Expect(indexDB.SetIndexKeyInMeta(ctx, newIndexKey)).To(MatchError(ContainSubstring("index: index channel cannot be indexed by another channel")))
					})
				})

				Describe("Data Channel", func() {
					It("Should set the data channel to a new key", func(ctx SpecContext) {
						newIndexKey := testutil.GenerateChannelKey()
						Expect(dataDB.SetIndexKeyInMeta(ctx, newIndexKey)).To(Succeed())
						Expect(dataDB.Channel().Index).To(Equal(newIndexKey))
					})
				})
			})

			Describe("RenameChannelInMeta", func() {
				It("Should rename the channel and persist it", func(ctx SpecContext) {
					Expect(dataDB.RenameChannelInMeta(ctx, "new_name")).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, dataDBfs, codec))
					Expect(ch.Name).To(Equal("new_name"))
				})

				It("Should be a no-op when the name is the same", func(ctx SpecContext) {
					Expect(dataDB.RenameChannelInMeta(ctx, "test")).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, dataDBfs, codec))
					Expect(ch.Name).To(Equal("test"))
				})
			})

			Describe("Size", func() {
				It("Should return zero for an empty database", func() {
					Expect(indexDB.Size()).To(Equal(telem.Size(0)))
					Expect(dataDB.Size()).To(Equal(telem.Size(0)))
				})

				It("Should return the correct size after writing data", func(ctx SpecContext) {
					w, _ := MustSucceed2(indexDB.OpenWriter(ctx, fixed.WriterConfig{
						Start:   telem.TimeStamp(0),
						Subject: control.Subject{Key: "size_test"},
					}))
					MustSucceed(w.Write(telem.NewSeriesSecondsTSV(0, 1, 2, 3, 4)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())

					expectedSize := telem.Size(5 * telem.TimeStampT.Density())
					Expect(indexDB.Size()).To(Equal(expectedSize))
				})

				It("Should accumulate size across multiple writes", func(ctx SpecContext) {
					w, _ := MustSucceed2(indexDB.OpenWriter(ctx, fixed.WriterConfig{
						Start:   telem.TimeStamp(0),
						Subject: control.Subject{Key: "size_test"},
					}))
					MustSucceed(w.Write(telem.NewSeriesSecondsTSV(0, 1, 2)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Write(telem.NewSeriesSecondsTSV(3, 4)))
					MustSucceed(w.Commit(ctx))
					MustSucceed(w.Close())

					expectedSize := telem.Size(5 * telem.TimeStampT.Density())
					Expect(indexDB.Size()).To(Equal(expectedSize))
				})
			})

		})
	}

	Describe("Close", func() {
		var db *fixed.DB
		BeforeEach(func(ctx SpecContext) {
			db = MustSucceed(fixed.Open(ctx, fixed.Config{
				FS:        xfs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}))
		})

		It("Should return an error when methods are called on a closed DB", func(ctx SpecContext) {

			Expect(db.Close()).To(Succeed())
			Expect(db.RenameChannelInMeta(ctx, "new_name")).To(HaveOccurredAs(fixed.ErrDBClosed))
			Expect(db.SetChannelKeyInMeta(ctx, testutil.GenerateChannelKey())).To(HaveOccurredAs(fixed.ErrDBClosed))
			Expect(db.SetIndexKeyInMeta(ctx, testutil.GenerateChannelKey())).To(HaveOccurredAs(fixed.ErrDBClosed))
			Expect(db.SetChannelKeyInMeta(ctx, testutil.GenerateChannelKey())).To(HaveOccurredAs(fixed.ErrDBClosed))
			Expect(db.Delete(ctx, telem.TimeRange{})).To(HaveOccurredAs(fixed.ErrDBClosed))
			Expect(db.GarbageCollect(ctx)).To(HaveOccurredAs(fixed.ErrDBClosed))
			_, err := db.HasDataFor(ctx, telem.TimeRangeMax)
			Expect(err).To(HaveOccurredAs(fixed.ErrDBClosed))
			_, _, err = db.OpenWriter(ctx, fixed.WriterConfig{})
			Expect(err).To(HaveOccurredAs(fixed.ErrDBClosed))
			_, err = db.OpenIterator(fixed.IteratorConfig{})
			Expect(err).To(HaveOccurredAs(fixed.ErrDBClosed))
		})

		It("Should return an error when a DB is closed while writers are still accessing it", func(ctx SpecContext) {
			db := MustSucceed(fixed.Open(ctx, fixed.Config{
				FS:        xfs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}))
			writer, _ := MustSucceed2(db.OpenWriter(ctx, fixed.WriterConfig{
				Subject: control.Subject{Key: "string"},
			}))
			Expect(db.Close()).To(MatchError(resource.ErrOpen))
			_ = MustSucceed(writer.Close())
			Expect(db.Close()).To(Succeed())
		})
	})
})
