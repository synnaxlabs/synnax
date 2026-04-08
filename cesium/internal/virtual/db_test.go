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
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB Metadata Operations", func() {
	for fsName, makeFS := range fileSystems {
		var (
			fs      fs.FS
			codec   = json.Codec
			cleanUp func() error
			dbKey   channel.Key
			db      *virtual.DB
		)

		Context("FS: "+fsName, func() {
			BeforeEach(func(ctx SpecContext) {
				fs, cleanUp = makeFS()
				dbKey = testutil.GenerateChannelKey()
				db = MustSucceed(virtual.Open(ctx, virtual.Config{
					FS:        fs,
					MetaCodec: codec,
					Channel: channel.Channel{
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
				It("Should rename the channel and persist it", func(ctx SpecContext) {
					Expect(db.RenameChannel(ctx, "new_name")).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, fs, codec))
					Expect(ch.Name).To(Equal("new_name"))
				})

				It("Should be a no-op when the name is the same", func(ctx SpecContext) {
					Expect(db.RenameChannel(ctx, "test")).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, fs, codec))
					Expect(ch.Name).To(Equal("test"))
				})
			})

			Describe("SetChannelKeyInMeta", func() {
				It("Should change the channel key and persist it", func(ctx SpecContext) {
					newKey := testutil.GenerateChannelKey()
					Expect(db.SetChannelKeyInMeta(ctx, newKey)).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, fs, codec))
					Expect(ch.Key).To(Equal(newKey))
				})

				It("Should be a no-op when the key is the same", func(ctx SpecContext) {
					Expect(db.SetChannelKeyInMeta(ctx, dbKey)).To(Succeed())
					ch := MustSucceed(meta.Read(ctx, fs, codec))
					Expect(ch.Key).To(Equal(dbKey))
				})
			})

			Describe("LeadingControlState", func() {
				It("Should return nil when there are no writers open on the DB", func() {
					Expect(db.LeadingControlState()).To(BeNil())
				})

				It("Should return the leading control state when there are writers open on the DB", func(ctx SpecContext) {
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
		BeforeEach(func(ctx SpecContext) {
			db = MustSucceed(virtual.Open(ctx, virtual.Config{
				FS:        fs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
		})

		It("Should return an error when methods are called on a closed DB", func(ctx SpecContext) {
			Expect(db.Close()).To(Succeed())
			Expect(db.RenameChannel(ctx, "new_name")).To(MatchError(virtual.ErrDBClosed))
			Expect(db.SetChannelKeyInMeta(ctx, testutil.GenerateChannelKey())).To(MatchError(virtual.ErrDBClosed))
		})

		It("Should return an error when a DB is closed while writers are still accessing it", func(ctx SpecContext) {
			db := MustSucceed(virtual.Open(ctx, virtual.Config{
				FS:        fs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Key:      testutil.GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
			writer, _ := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
				Subject: control.Subject{Key: "string"},
			}))
			Expect(db.Close()).To(HaveOccurredAs(resource.ErrOpen))
			_ = MustSucceed(writer.Close())
			Expect(db.Close()).To(Succeed())
		})
	})
})
