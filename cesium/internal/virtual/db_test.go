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
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("DB Metadata Operations", func() {
	Describe("Close", func() {
		var db *virtual.DB
		BeforeEach(func(ctx SpecContext) {
			db = MustOpen(virtual.Open(ctx, virtual.Config{
				FS:        fs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
		})

		It(
			"Should return an error when methods are called on a closed DB",
			func(ctx SpecContext) {
				Expect(db.Close()).To(Succeed())
				Expect(db.OpenWriter(ctx, virtual.WriterConfig{
					Start: 10 * telem.SecondTS,
				})).Error().To(MatchError(virtual.ErrDBClosed))
			},
		)

		It(
			"Should return an error when a DB is closed while writers are still accessing it",
			func(ctx SpecContext) {
				db := MustSucceed(virtual.Open(ctx, virtual.Config{
					FS:        fs.NewMem(),
					MetaCodec: json.Codec,
					Channel: channel.Channel{
						Key:      GenerateChannelKey(),
						Name:     "test",
						DataType: telem.Int64T,
						Virtual:  true,
					},
				}))
				writer := MustSucceed(db.OpenWriter(ctx, virtual.WriterConfig{}))
				Expect(db.Close()).To(MatchError(resource.ErrOpen))
				Expect(writer.Close()).To(Succeed())
				Expect(db.Close()).To(Succeed())
			},
		)
	})

	Describe("MarkClosed", func() {
		var db *virtual.DB
		BeforeEach(func(ctx SpecContext) {
			db = MustOpen(virtual.Open(ctx, virtual.Config{
				FS:        fs.NewMem(),
				MetaCodec: json.Codec,
				Channel: channel.Channel{
					Key:      GenerateChannelKey(),
					Name:     "test",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}))
		})
		cfg := virtual.WriterConfig{}

		It("Should refuse new writers until UnmarkClosed", func(ctx SpecContext) {
			Expect(db.MarkClosed()).To(Succeed())
			Expect(db.OpenWriter(ctx, cfg)).Error().To(MatchError(virtual.ErrDBClosed))
			db.UnmarkClosed()
			w := MustSucceed(db.OpenWriter(ctx, cfg))
			Expect(w.Close()).To(Succeed())
		})

		It("Should leave the DB open if a writer is open", func(ctx SpecContext) {
			w := MustSucceed(db.OpenWriter(ctx, cfg))
			Expect(db.MarkClosed()).To(MatchError(resource.ErrOpen))
			Expect(w.Close()).To(Succeed())
			w = MustSucceed(db.OpenWriter(ctx, cfg))
			Expect(w.Close()).To(Succeed())
		})

		It("Should return ErrDBClosed if the DB is closed", func() {
			Expect(db.MarkClosed()).To(Succeed())
			Expect(db.MarkClosed()).To(MatchError(virtual.ErrDBClosed))
		})
	})
})
