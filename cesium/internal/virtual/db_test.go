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
	"github.com/synnaxlabs/cesium/internal/resource"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB Metadata Operations", func() {
	var (
		dbKey channel.Key
		db    *virtual.DB
	)

	BeforeEach(func() {
		dbKey = GenerateChannelKey()
		db = MustSucceed(virtual.Open(virtual.Config{
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
		It("Should rename the channel in memory", func() {
			Expect(db.RenameChannel("new_name")).To(Succeed())
			Expect(db.Channel().Name).To(Equal("new_name"))
		})
	})

	Describe("SetChannelKey", func() {
		It("Should change the channel key in memory", func() {
			newKey := GenerateChannelKey()
			Expect(db.SetChannelKey(newKey)).To(Succeed())
			Expect(db.Channel().Key).To(Equal(newKey))
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
		It("Should return an error when methods are called on a closed DB", func() {
			db := MustSucceed(virtual.Open(virtual.Config{
				Channel: channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
			Expect(db.Close()).To(Succeed())
			Expect(db.RenameChannel("new_name")).To(MatchError(virtual.ErrDBClosed))
			Expect(db.SetChannelKey(GenerateChannelKey())).To(MatchError(virtual.ErrDBClosed))
		})

		It("Should return an error when a DB is closed while writers are still accessing it", func(ctx SpecContext) {
			db := MustSucceed(virtual.Open(virtual.Config{
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
