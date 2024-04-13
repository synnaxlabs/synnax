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
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Iterator Behavior", Ordered, func() {
	var db *cesium.DB
	BeforeAll(func() { db = openMemDB() })
	AfterAll(func() { Expect(db.Close()).To(Succeed()) })

	Describe("Close", func() {
		It("Should not allow operations on a closed iterator", func() {
			Expect(db.CreateChannel(ctx, cesium.Channel{Key: 1, DataType: telem.Int64T, Rate: 1 * telem.Hz})).To(Succeed())
			var (
				i = MustSucceed(db.OpenIterator(cesium.IteratorConfig{Bounds: telem.TimeRangeMax, Channels: []core.ChannelKey{1}}))
				e = core.EntityClosed("cesium.iterator")
			)
			Expect(i.Close()).To(Succeed())
			Expect(i.Valid()).To(BeFalse())
			Expect(i.SeekFirst()).To(BeFalse())
			Expect(i.Valid()).To(BeFalse())
			Expect(i.Error()).To(MatchError(e))
			Expect(i.Close()).To(Succeed())
		})
	})
})
