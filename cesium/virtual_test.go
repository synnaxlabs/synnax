// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	jsonv2 "encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/meta"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/encoding/json"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Virtual channel directories", func() {
	var (
		mem *xfs.MemFS
		key cesium.ChannelKey
	)
	// metaPath is the path of the metadata file of the directory of key.
	metaPath := func() string { return channelKeyToPath(key) + "/" + meta.File }
	BeforeEach(func(ctx SpecContext) {
		mem = xfs.NewMem()
		key = GenerateChannelKey()
		// A Core older than this Cesium wrote a directory with a metadata file for each
		// leased virtual channel.
		dir := MustSucceed(mem.Sub(channelKeyToPath(key)))
		Expect(meta.Create(
			ctx,
			dir,
			json.NewCodec(jsonv2.MatchCaseInsensitiveNames(true)),
			cesium.Channel{
				Key:      key,
				Name:     "sy_task_cmd",
				Virtual:  true,
				DataType: telem.StringT,
			},
		)).To(Succeed())
	})

	It(
		"Should leave the metadata of a virtual channel as it was, so an undo reads it",
		func(ctx SpecContext) {
			before := MustSucceed(xfs.ReadFile(mem, metaPath()))
			db := openDBOnFS(ctx, mem)
			Expect(db.RetrieveChannel(ctx, key)).Error().
				To(MatchError(cesium.ErrChannelNotFound))
			Expect(db.Sweep(ctx, []cesium.ChannelKey{key})).To(BeEmpty())
			Expect(db.Close()).To(Succeed())
			Expect(xfs.ReadFile(mem, metaPath())).To(Equal(before))
		},
	)

	It("Should remove the directory of a virtual channel that Drop drops", func(
		ctx SpecContext,
	) {
		db := mustOpenDBOnFS(ctx, mem)
		Expect(db.Drop([]cesium.ChannelKey{key})).To(Succeed())
		Expect(mem.Exists(channelKeyToPath(key))).To(BeFalse())
	})

	It("Should remove the directory of a virtual channel with no record", func(
		ctx SpecContext,
	) {
		db := mustOpenDBOnFS(ctx, mem)
		Expect(db.Sweep(ctx, nil)).To(BeEmpty())
		Expect(mem.Exists(channelKeyToPath(key))).To(BeFalse())
	})

	It("Should refuse a channel with the key of a virtual channel directory", func(
		ctx SpecContext,
	) {
		db := mustOpenDBOnFS(ctx, mem)
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key: key, Name: "time", IsIndex: true, DataType: telem.TimestampT,
		})).To(MatchError(ContainSubstring("it already exists")))
	})

	It("Should refuse to create a virtual channel", func(ctx SpecContext) {
		db := mustOpenDBOnFS(ctx, xfs.NewMem())
		Expect(db.CreateChannel(ctx, cesium.Channel{
			Key:      GenerateChannelKey(),
			Name:     "command",
			Virtual:  true,
			DataType: telem.Int64T,
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot create a virtual channel")),
		))
	})
})
