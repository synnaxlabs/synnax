package unary_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("DB Metadata Operations", func() {
	for fsName, makeFS := range fileSystems {
		var (
			fs         xfs.FS
			codec      = &binary.JSONCodec{}
			cleanUp    func() error
			indexDBfs  xfs.FS
			indexDBKey core.ChannelKey
			indexDB    *unary.DB
			dataDBfs   xfs.FS
			dataDBKey  core.ChannelKey
			dataDB     *unary.DB
		)
		Context("FS: "+fsName, func() {
			BeforeEach(func() {
				fs, cleanUp = makeFS()
				indexDBKey = testutil.GenerateChannelKey()
				indexDBfs = MustSucceed(fs.Sub("index"))
				indexDB = MustSucceed(unary.Open(unary.Config{
					FS:        indexDBfs,
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      indexDBKey,
						Name:     "test",
						DataType: telem.TimeStampT,
						IsIndex:  true,
					},
				}))
				dataDBKey = testutil.GenerateChannelKey()
				dataDBfs = MustSucceed(fs.Sub("data"))
				dataDB = MustSucceed(unary.Open(unary.Config{
					FS:        dataDBfs,
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      dataDBKey,
						Name:     "test",
						DataType: telem.Int64T,
						IsIndex:  false,
						Index:    indexDBKey,
					},
				}))
			})

			AfterEach(func() {
				Expect(cleanUp()).To(Succeed())
				Expect(indexDB.Close()).To(Succeed())
				Expect(dataDB.Close()).To(Succeed())
			})

			Describe("SetChannelKeyInMeta", func() {
				It("Should change both key and index when channel is an index", func() {
					newKey := testutil.GenerateChannelKey()
					Expect(indexDB.SetChannelKeyInMeta(newKey)).To(Succeed())
					ch, err := meta.Read(indexDBfs, codec)
					Expect(err).ToNot(HaveOccurred())
					Expect(ch.Key).To(Equal(newKey))
					Expect(ch.Index).To(Equal(newKey))
				})

				It("Should change only the key when channel is not an index", func() {
					newKey := testutil.GenerateChannelKey()
					Expect(dataDB.SetChannelKeyInMeta(newKey)).To(Succeed())
					ch, err := meta.Read(dataDBfs, codec)
					Expect(err).ToNot(HaveOccurred())
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
					It("Should set the index channel to a new key", func() {
						newIndexKey := testutil.GenerateChannelKey()
						Expect(indexDB.Channel().Key).ToNot(Equal(newIndexKey))
						Expect(indexDB.SetChannelKeyInMeta(newIndexKey))
						Expect(indexDB.SetIndexKeyInMeta(newIndexKey)).To(Succeed())
						Expect(indexDB.Channel().Key).To(Equal(newIndexKey))
						Expect(indexDB.Channel().Index).To(Equal(newIndexKey))
					})

					It("Should return an error when attempting to set an index key that is different than the channel key", func() {
						newIndexKey := testutil.GenerateChannelKey()
						Expect(indexDB.SetIndexKeyInMeta(newIndexKey)).To(MatchError(validate.FieldError{
							Field:   "index",
							Message: "index channel cannot be indexed by another channel",
						}))
					})
				})

				Describe("Data Channel", func() {
					It("Should set the data channel to a new key", func() {
						newIndexKey := testutil.GenerateChannelKey()
						Expect(dataDB.SetIndexKeyInMeta(newIndexKey)).To(Succeed())
						Expect(dataDB.Channel().Index).To(Equal(newIndexKey))
					})
				})
			})

			Describe("RenameChannelInMeta", func() {
				It("Should rename the channel and persist it", func() {
					Expect(dataDB.RenameChannelInMeta("new_name")).To(Succeed())
					ch, err := meta.Read(dataDBfs, codec)
					Expect(err).ToNot(HaveOccurred())
					Expect(ch.Name).To(Equal("new_name"))
				})

				It("Should be a no-op when the name is the same", func() {
					Expect(dataDB.RenameChannelInMeta("test")).To(Succeed())
					ch, err := meta.Read(dataDBfs, codec)
					Expect(err).ToNot(HaveOccurred())
					Expect(ch.Name).To(Equal("test"))
				})
			})

		})
	}

	Describe("Close", func() {
		var db *unary.DB
		BeforeEach(func() {
			db = MustSucceed(unary.Open(unary.Config{
				FS:        xfs.NewMem(),
				MetaCodec: &binary.JSONCodec{},
				Channel: core.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}))
		})

		It("Should return an error when methods are called on a closed DB", func() {

			Expect(db.Close()).To(Succeed())
			Expect(db.RenameChannelInMeta("new_name")).To(HaveOccurredAs(unary.ErrDBClosed))
			Expect(db.SetChannelKeyInMeta(testutil.GenerateChannelKey())).To(HaveOccurredAs(unary.ErrDBClosed))
			Expect(db.SetIndexKeyInMeta(testutil.GenerateChannelKey())).To(HaveOccurredAs(unary.ErrDBClosed))
			Expect(db.SetChannelKeyInMeta(testutil.GenerateChannelKey())).To(HaveOccurredAs(unary.ErrDBClosed))
			Expect(db.Delete(ctx, telem.TimeRange{})).To(HaveOccurredAs(unary.ErrDBClosed))
			Expect(db.GarbageCollect(ctx)).To(HaveOccurredAs(unary.ErrDBClosed))
			_, err := db.HasDataFor(ctx, telem.TimeRangeMax)
			Expect(err).To(HaveOccurredAs(unary.ErrDBClosed))
			_, _, err = db.OpenWriter(ctx, unary.WriterConfig{})
			Expect(err).To(HaveOccurredAs(unary.ErrDBClosed))
			_, err = db.OpenIterator(unary.IteratorConfig{})
			Expect(err).To(HaveOccurredAs(unary.ErrDBClosed))
		})

		It("Should return an error when a DB is closed while writers are still accessing it", func() {
			db := MustSucceed(unary.Open(unary.Config{
				FS:        xfs.NewMem(),
				MetaCodec: &binary.JSONCodec{},
				Channel: core.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}))
			writer, _ := MustSucceed2(db.OpenWriter(ctx, unary.WriterConfig{
				Subject: control.Subject{Key: "string"},
			}))
			Expect(db.Close()).To(MatchError(core.ErrOpenEntity))
			_ = MustSucceed(writer.Close())
			Expect(db.Close()).To(Succeed())
		})
	})
})
