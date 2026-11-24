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
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/io/fs/testutil/fault"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Check: the last unpin of a converted segment that a delete buries",
	func() {
		It(
			"Should remove the old file that an old compaction left under its "+
				"temporary name",
			func(ctx SpecContext) {
				const spare cesium.ChannelKey = 100
				disk := MustSucceed(fault.New(fault.Config{}))
				writeOld(ctx, disk)
				p := newestPointer(disk, oldStr)
				temp := oldName(oldStr, p) + "_temp"
				Expect(disk.Rename(oldName(oldStr, p), temp)).To(Succeed())
				// The open that converts the root reads the old file from its temporary
				// name, and the next open moves it into place.
				ins, _ := ObservedInstrumentation(zapcore.InfoLevel)
				db, closeDB := openConverted(ctx, disk, ins)
				Expect(db.CreateChannel(ctx, cesium.Channel{
					Key:      spare,
					Name:     "spare",
					IsIndex:  true,
					DataType: telem.TimestampT,
				})).To(Succeed())
				iter := MustSucceed(db.OpenIterator(cesium.IteratorConfig{
					Bounds:   telem.TimeRangeMax,
					Channels: []cesium.ChannelKey{oldStr},
				}))
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(db.DeleteTimeRange(
					ctx,
					[]cesium.ChannelKey{oldStr},
					telem.TimeRange{Start: p.start, End: p.end},
				)).To(Succeed())
				// A Drop returns once the log covers it, and its reap then hands the
				// buried segment to the pin of the iterator.
				Expect(db.Drop(ctx, []cesium.ChannelKey{spare})).To(Succeed())
				Expect(disk.Exists(temp)).To(BeTrue(), "a file that a reader pins")
				Expect(iter.Close()).To(Succeed())
				Expect(disk.Exists(temp)).To(BeFalse())
				Expect(disk.Exists(oldName(oldStr, p))).To(BeFalse())
				Expect(closeDB()).To(Succeed())
			},
		)
	})
