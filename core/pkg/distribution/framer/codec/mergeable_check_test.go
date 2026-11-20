// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Encode of a type with no code", func() {
	bad := telem.Series{DataType: "int128", Data: make([]byte, 16)}
	good := telem.NewSeriesV[float32](1)
	DescribeTable(
		"Should fail, not panic, on a type with no code beside a series of its key",
		func(ctx SpecContext, first, second telem.Series) {
			fr := multi(keyed{1, first}, keyed{1, second})
			Expect(float32Codec().Encode(ctx, fr)).Error().
				To(invalid(`data type "int128" of channel 1 has no type code`))
		},
		Entry("before the series", bad, good),
		Entry("after the series", good, bad),
	)
})
