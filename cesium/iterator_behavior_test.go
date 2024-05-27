// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", func() {
	for fsName, makeFS := range fileSystems {
		Context("FS: "+fsName, Ordered, func() {
			var (
				db      *cesium.DB
				fs      xfs.FS
				cleanUp func() error
			)
			BeforeAll(func() {
				fs, cleanUp = makeFS()
				db = openDBOnFS(fs)
			})
			AfterAll(func() {
				Expect(db.Close()).To(Succeed())
				Expect(cleanUp()).To(Succeed())
			})

			Describe("Close", func() {
				It("Should not allow operations on a closed iterator", func() {
					key := testutil.GenerateChannelKey()
					Expect(db.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					var (
						i = MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{key}}))
						e = core.EntityClosed("cesium.iterator")
					)
					Expect(i.Close()).To(Succeed())
					Expect(i.Valid()).To(BeFalse())
					Expect(i.SeekFirst()).To(BeFalse())
					Expect(i.Valid()).To(BeFalse())
					Expect(i.Error()).To(HaveOccurredAs(e))
					Expect(i.Close()).To(Succeed())
				})

				It("Should not allow opening an iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow opening a stream iterator on a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.NewStreamIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []cesium.ChannelKey{key}})
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})

				It("Should not allow reading from a closed db", func() {
					sub := MustSucceed(fs.Sub("closed-fs"))
					key := cesium.ChannelKey(1)
					subDB := openDBOnFS(sub)
					Expect(subDB.CreateChannel(ctx, cesium.Channel{Key: key, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
					Expect(subDB.Close()).To(Succeed())
					_, err := subDB.Read(ctx, telem.TimeRangeMax, key)
					Expect(err).To(HaveOccurredAs(core.EntityClosed("cesium.db")))

					Expect(fs.Remove("closed-fs")).To(Succeed())
				})
			})
		})
	}
})
