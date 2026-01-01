// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/control"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB Metadata Operations", func() {
	for fsName, makeFS := range fileSystems {
		var (
			fs      xfs.FS
			codec   = &binary.JSONCodec{}
			cleanUp func() error
			dbKey   core.ChannelKey
			db      *virtual.DB
		)

		Context("FS: "+fsName, func() {
			BeforeEach(func() {
				fs, cleanUp = makeFS()
				dbKey = testutil.GenerateChannelKey()
				db = MustSucceed(virtual.Open(ctx, virtual.Config{
					FS:        fs,
					MetaCodec: codec,
					Channel: core.Channel{
						Key:      dbKey,
						Name:     "test",
						DataType: telem.Int64T,
						Virtual:  true,
					},
				}))
			})

			AfterEach(func() {
				Expect(cleanUp()).To(Succeed())
				Expect(db.Close()).To(Succeed())
			})

			Describe("RenameChannel", func() {
				It("Should rename the channel and persist it", func() {
					Expect(db.RenameChannel(ctx, "new_name")).To(Succeed())
					ch, err := meta.Read(ctx, fs, codec)
					Expect(err).ToNot(HaveOccurred())
					Expect(ch.Name).To(Equal("new_name"))
				})

				It("Should be a no-op when the name is the same", func() {
					Expect(db.RenameChannel(ctx, "test")).To(Succeed())
					ch, err := meta.Read(ctx, fs, codec)
					Expect(err).ToNot(HaveOccurred())
					Expect(ch.Name).To(Equal("test"))
				})
			})

			Describe("SetChannelKeyInMeta", func() {
				It("Should change the channel key and persist it", func() {
					newKey := testutil.GenerateChannelKey()
					Expect(db.SetChannelKeyInMeta(ctx, newKey)).To(Succeed())
					ch, err := meta.Read(ctx, fs, codec)
					Expect(err).ToNot(HaveOccurred())
					Expect(ch.Key).To(Equal(newKey))
				})

				It("Should be a no-op when the key is the same", func() {
					Expect(db.SetChannelKeyInMeta(ctx, dbKey)).To(Succeed())
					ch, err := meta.Read(ctx, fs, codec)
					Expect(err).ToNot(HaveOccurred())
					Expect(ch.Key).To(Equal(dbKey))
				})
			})

			Describe("LeadingControlState", func() {
				It("Should return nil when there are no writers open on the DB", func() {
					Expect(db.LeadingControlState()).To(BeNil())
				})

				It("Should return the leading control state when there are writers open on the DB", func() {
					w, transfer := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
						Start:     10 * telem.SecondTS,
						Authority: control.AuthorityAbsolute,
						Subject:   control.Subject{Key: "foo"},
					}))
					Expect(transfer.Occurred()).To(BeTrue())
					Expect(db.LeadingControlState()).ToNot(BeNil())
					Expect(db.LeadingControlState().Authority).To(Equal(control.AuthorityAbsolute))
					Expect(db.LeadingControlState().Subject.Key).To(Equal("foo"))
					t, err := w.Close()
					Expect(t.Occurred()).To(BeTrue())
					Expect(err).To(BeNil())
				})
			})
		})
	}

	Describe("Close", func() {
		var db *virtual.DB
		BeforeEach(func() {
			db = MustSucceed(virtual.Open(ctx, virtual.Config{
				FS:        xfs.NewMem(),
				MetaCodec: &binary.JSONCodec{},
				Channel: core.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
		})

		It("Should return an error when methods are called on a closed DB", func() {
			Expect(db.Close()).To(Succeed())
			Expect(db.RenameChannel(ctx, "new_name")).To(MatchError(virtual.ErrDBClosed))
			Expect(db.SetChannelKeyInMeta(ctx, testutil.GenerateChannelKey())).To(MatchError(virtual.ErrDBClosed))
		})

		It("Should return an error when a DB is closed while writers are still accessing it", func() {
			db := MustSucceed(virtual.Open(ctx, virtual.Config{
				FS:        xfs.NewMem(),
				MetaCodec: &binary.JSONCodec{},
				Channel: core.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
			writer, _ := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
				Subject: control.Subject{Key: "string"},
			}))
			Expect(db.Close()).To(HaveOccurredAs(core.ErrOpenResource))
			_ = MustSucceed(writer.Close())
			Expect(db.Close()).To(Succeed())
		})
	})
})
