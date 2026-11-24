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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check: a read of a checked version 0 segment", func() {
	It("Should fail a read of a block that no longer matches its check file",
		func(ctx SpecContext) {
			keys := []cesium.ChannelKey{oldIdx, oldI32, oldStr}
			disk := MustSucceed(fault.New(fault.Config{}))
			writeOld(ctx, disk)
			name := oldName(oldI32, newestPointer(disk, oldI32))
			db, closeDB := openConverted(ctx, disk, alamos.New("cesium"))
			Eventually(func() (map[cesium.ChannelKey]int, error) {
				return checkFiles(disk, keys...)
			}).WithTimeout(time.Minute).Should(Equal(oldFilesOf(disk, keys...)))
			f := MustSucceed(disk.Open(name, os.O_RDWR))
			b := make([]byte, 1)
			MustSucceed(f.ReadAt(b, 0))
			b[0] ^= 0xFF
			MustSucceed(f.WriteAt(b, 0))
			Expect(f.Close()).To(Succeed())
			Expect(db.Read(ctx, telem.TimeRangeMax, oldI32)).Error().
				To(MatchError(place.ErrCorrupt))
			Expect(closeDB()).To(Succeed())
		})
})
