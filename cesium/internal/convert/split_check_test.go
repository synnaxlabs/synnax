// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package convert_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium/internal/convert"
	"github.com/synnaxlabs/cesium/internal/state"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap/zapcore"
)

var _ = Describe("Split check", func() {
	// Index key 1 holds two stretches of two writers: 1 to 3 s in file 1, and 20 to
	// 22 s in file 2. The second writer wrote no sample of data key 2.
	DescribeTable(
		"Should cut the data samples with no stamps in the range of their pointer "+
			"when a later index pointer exists, and log the cut once",
		func(first old, samples []int64, kept []state.Pointer, offset, bytes uint32) {
			fs := xfs.NewMem()
			root := oldRoot{fs: fs}
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3))
			root.file(1, 2, stamps(20, 21, 22))
			root.pointers(1, old{1, 4, 1, 0, 24}, old{20, 23, 2, 0, 24})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV(samples...).Data)
			root.pointers(2, first)
			ins, logs := ObservedInstrumentation(zapcore.WarnLevel)
			p := MustSucceed(convert.Read(convert.Config{
				Instrumentation: ins,
				FS:              fs,
				Codec:           codec,
			}))
			Expect(p.Channels).To(ContainElement(HaveField("Key", uint32(2))))
			var got []state.Pointer
			for _, ch := range p.Snapshot.Channels {
				if ch.Key == 2 {
					for _, seg := range ch.Segments {
						got = append(got, seg.Pointers...)
					}
				}
			}
			Expect(got).To(Equal(kept))
			Expect(logs.Len()).To(Equal(1))
			e := logs.All()[0]
			Expect(e.Message).To(Equal(
				"cut the samples of a data pointer that its index has no stamps for",
			))
			Expect(e.ContextMap()).To(Equal(map[string]any{
				"key":    uint32(2),
				"file":   "2/1.domain",
				"offset": offset,
				"bytes":  bytes,
				"index":  uint32(1),
			}))
		},
		Entry(
			"more samples than stamps in its range",
			old{1, 4, 1, 0, 40},
			[]int64{1, 2, 3, 4, 5},
			[]state.Pointer{ptr(1, 3, 1, 0, 0, 24, 3)},
			uint32(24),
			uint32(16),
		),
		Entry(
			"a range with no stamp",
			old{10, 13, 1, 0, 24},
			[]int64{10, 11, 12},
			[]state.Pointer(nil),
			uint32(0),
			uint32(24),
		),
	)
})
