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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/resource"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB Metadata Operations", func() {
	var (
		dbKey channel.Key
		db    *virtual.DB
	)

	BeforeEach(func(ctx SpecContext) {
		dbKey = GenerateChannelKey()
		db = MustSucceed(virtual.Open(ctx, virtual.Config{
			Channel: channel.Channel{
				Key:      dbKey,
				Name:     "test",
				DataType: telem.Int64T,
				Virtual:  true,
			},
		}))
	})

	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})

	Describe("RenameChannel", func() {
		It("Should rename the channel in memory", func(ctx SpecContext) {
			Expect(db.RenameChannel(ctx, "new_name")).To(Succeed())
			Expect(db.Channel().Name).To(Equal("new_name"))
		})
	})

	Describe("SetChannelKey", func() {
		It("Should change the channel key in memory", func(ctx SpecContext) {
			newKey := GenerateChannelKey()
			Expect(db.SetChannelKey(ctx, newKey)).To(Succeed())
			Expect(db.Channel().Key).To(Equal(newKey))
		})
	})

	Describe("Metadata Persistence", func() {
		var (
			fs xfs.FS
			ch channel.Channel
		)

		BeforeEach(func() {
			fs = MustSucceed(xfs.NewMem().Sub("virtual"))
			ch = channel.Channel{
				Key:      GenerateChannelKey(),
				Name:     "persisted",
				DataType: telem.Int64T,
				Virtual:  true,
			}
		})

		openWithFS := func(ctx context.Context) *virtual.DB {
			return MustSucceed(virtual.Open(ctx, virtual.Config{
				Channel:   ch,
				FS:        fs,
				MetaCodec: json.Codec,
			}))
		}

		It("Should create the metadata file on open and read it back on reopen", func(ctx SpecContext) {
			db := openWithFS(ctx)
			Expect(db.Close()).To(Succeed())
			reopened := MustSucceed(virtual.Open(ctx, virtual.Config{
				Channel:   channel.Channel{Key: ch.Key},
				FS:        fs,
				MetaCodec: json.Codec,
			}))
			Expect(reopened.Channel().Name).To(Equal("persisted"))
			Expect(reopened.Channel().Virtual).To(BeTrue())
			Expect(reopened.Close()).To(Succeed())
		})

		It("Should persist a rename across reopens", func(ctx SpecContext) {
			db := openWithFS(ctx)
			Expect(db.RenameChannel(ctx, "renamed")).To(Succeed())
			Expect(db.Close()).To(Succeed())
			reopened := openWithFS(ctx)
			Expect(reopened.Channel().Name).To(Equal("renamed"))
			Expect(reopened.Close()).To(Succeed())
		})

		It("Should persist a key change across reopens", func(ctx SpecContext) {
			db := openWithFS(ctx)
			newKey := GenerateChannelKey()
			Expect(db.SetChannelKey(ctx, newKey)).To(Succeed())
			Expect(db.Close()).To(Succeed())
			reopened := openWithFS(ctx)
			Expect(reopened.Channel().Key).To(Equal(newKey))
			Expect(reopened.Close()).To(Succeed())
		})

		It("Should reject a file system without a meta codec", func(ctx SpecContext) {
			Expect(virtual.Open(ctx, virtual.Config{Channel: ch, FS: fs})).Error().
				To(MatchError(ContainSubstring("meta_codec")))
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
			t := MustSucceed(w.Close())
			Expect(t.Occurred()).To(BeTrue())
		})
	})

	Describe("Close", func() {
		It("Should return an error when methods are called on a closed DB", func(ctx SpecContext) {
			db := MustSucceed(virtual.Open(ctx, virtual.Config{
				Channel: channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
			Expect(db.Close()).To(Succeed())
			Expect(db.RenameChannel(ctx, "new_name")).To(MatchError(virtual.ErrDBClosed))
			Expect(db.SetChannelKey(ctx, GenerateChannelKey())).To(MatchError(virtual.ErrDBClosed))
		})

		It("Should return an error when a DB is closed while writers are still accessing it", func(ctx SpecContext) {
			db := MustSucceed(virtual.Open(ctx, virtual.Config{
				Channel: channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
			writer, _ := MustSucceed2(db.OpenWriter(ctx, virtual.WriterConfig{
				Subject: control.Subject{Key: "string"},
			}))
			Expect(db.Close()).To(MatchError(resource.ErrOpen))
			_ = MustSucceed(writer.Close())
			Expect(db.Close()).To(Succeed())
		})
	})
})
