// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Bench Util Test", func() {
	DescribeTable("GenerateFrameAndChannels", func(numIndex, numData, numRate, samplesPerDomain int) {
		data, chs, keys := testutil.GenerateDataAndChannels(numIndex, numData, numRate, samplesPerDomain)

		Expect(chs).To(HaveLen(numIndex + numData + numRate))
		for i := 0; i < numIndex; i++ {
			Expect(chs[i].IsIndex).To(BeTrue())
			Expect(keys[i]).To(Equal(cesium.ChannelKey(i + 1)))
		}
		for i := numIndex; i < numIndex+numData; i++ {
			Expect(chs[i].Index).To(Equal(cesium.ChannelKey((i+1)%numIndex + 1)))
			Expect(keys[i]).To(Equal(cesium.ChannelKey(i + 1)))
		}
		for i := numIndex + numData; i < numIndex+numData+numRate; i++ {
			Expect(chs[i].Rate).To(Equal(1 * telem.Hz))
			Expect(keys[i]).To(Equal(cesium.ChannelKey(i + 1)))
		}

		// Assert that the data channel has the right length
		Expect(data.Len()).To(Equal(int64(samplesPerDomain)))
	},
		Entry("normal", 1, 2, 1, 2),
		Entry("many indices", 3, 5, 0, 3),
		Entry("more indices than data", 10, 5, 3, 15),
		Entry("big", 10, 2342, 123, 400),
	)
})
