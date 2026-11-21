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
	"github.com/synnaxlabs/cesium/internal/convert"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Split check", func() {
	// Index key 1 holds two stretches of two writers: 1 to 3 s in file 1, and 20 to
	// 22 s in file 2. The second writer wrote no sample of data key 2.
	DescribeTable(
		"Should refuse data samples with no stamps in the range of their pointer when a "+
			"later index pointer exists",
		func(first old, samples []int64, msg string) {
			fs := xfs.NewMem()
			root := oldRoot{fs: fs}
			root.channel(idx(1))
			root.file(1, 1, stamps(1, 2, 3))
			root.file(1, 2, stamps(20, 21, 22))
			root.pointers(1, old{1, 4, 1, 0, 24}, old{20, 23, 2, 0, 24})
			root.channel(data(telem.Int64T))
			root.file(2, 1, telem.NewSeriesV(samples...).Data)
			root.pointers(2, first)
			Expect(convert.Read(convert.Config{FS: fs, Codec: codec})).Error().
				To(MatchError(msg))
		},
		Entry(
			"more samples than stamps in its range",
			old{1, 4, 1, 0, 40},
			[]int64{1, 2, 3, 4, 5},
			"bytes 24 to 40 of 2/1.domain have no stamps in key 1",
		),
		Entry(
			"a range with no stamp",
			old{10, 13, 1, 0, 24},
			[]int64{10, 11, 12},
			"bytes 0 to 24 of 2/1.domain have no stamps in key 1",
		),
	)
})
