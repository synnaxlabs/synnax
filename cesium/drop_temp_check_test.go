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
	"path"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Check: a Drop of a key whose old file has a compaction file", func() {
	DescribeTable("Should remove the directory of the key by the Sweep after the Drop",
		func(ctx SpecContext, pinned bool) {
			disk := MustSucceed(fault.New(fault.Config{}))
			writeOld(ctx, disk)
			// The old file and its compaction file both exist, so the conversion reads
			// the old file and keeps the other.
			name := oldName(oldI32, newestPointer(disk, oldI32))
			sub := MustSucceed(disk.Sub(path.Dir(name)))
			f := MustSucceed(sub.Open(path.Base(name)+"_temp", os.O_CREATE|os.O_WRONLY))
			MustSucceed(f.WriteAt([]byte{1, 2, 3, 4}, 0))
			Expect(f.Close()).To(Succeed())
			ins, _ := ObservedInstrumentation(zapcore.InfoLevel)
			// The first open converts the root, and the second loads its head.
			_, closeDB := openConverted(ctx, disk, ins)
			Expect(closeDB()).To(Succeed())
			db, closeDB := openConverted(ctx, disk, ins)
			// finish reads to the end of the reader that pins the old file, if any.
			finish := func() {}
			if pinned {
				iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
					Bounds:   telem.TimeRangeMax,
					Channels: []cesium.ChannelKey{oldI32},
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				finish = func() {
					for iter.Next(cesium.AutoSpan) {
					}
					Expect(iter.Error()).ToNot(HaveOccurred())
					Expect(iter.Close()).To(Succeed())
				}
			}
			Expect(db.Drop(ctx, []cesium.ChannelKey{oldI32})).To(Succeed())
			finish()
			Expect(closeDB()).To(Succeed())
			db, closeDB = openConverted(ctx, disk, ins)
			MustSucceed2(db.Sweep(ctx, []cesium.ChannelKey{oldIdx, oldStr}))
			Expect(closeDB()).To(Succeed())
			dir := strconv.Itoa(int(oldI32))
			var left []string
			if MustSucceed(disk.Exists(dir)) {
				for _, info := range MustSucceed(disk.List(dir)) {
					left = append(left, info.Name())
				}
			}
			Expect(left).To(BeEmpty())
			Expect(MustSucceed(disk.Exists(dir))).To(BeFalse())
		},
		Entry("with no reader", false),
		Entry("with a reader that pins the old file across the Drop", true),
	)
})
