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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/cesium/internal/testutil"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Delete intents check, read faults", func() {
	It("Should keep an intent that one bad read of the intents file damaged",
		func(ctx SpecContext) {
			// The first read of each block flips one bit. A second read is good.
			disk := MustSucceed(fault.New(fault.Config{ReadCorruptionRate: 1}))
			db := openDBOnFS(ctx, disk)
			Expect(db.Intend([]cesium.ChannelKey{GenerateChannelKey()})).To(Succeed())
			Expect(db.Close()).To(Succeed())
			db = openDBOnFS(ctx, disk)
			Expect(db.Close()).To(Succeed())
			Expect(MustSucceed(disk.Stat("delete-intents")).Size()).To(Equal(int64(16)))
		},
	)
})
