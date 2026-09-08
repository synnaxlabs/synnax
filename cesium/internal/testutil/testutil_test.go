// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
)

var _ = Describe("Test Util Test", func() {
	Describe("GenerateChannelKey", func() {
		It("Should generate a unique channel key every time it is called", func() {
			var (
				keys = make([]cesium.ChannelKey, 1000)
				wg   = sync.WaitGroup{}
			)
			wg.Add(1000)
			for i := range 1000 {
				go func() {
					defer wg.Done()
					keys[i] = GenerateChannelKey()
				}()
			}

			wg.Wait()

			Expect(keys).To(HaveLen(1000))
			Expect(lo.Uniq(keys)).To(HaveLen(1000))
		})
	})

	DescribeTable(
		"GenerateFrameAndChannels",
		func(numIndex, numData, samplesPerDomain int) {
			data, chs, keys := GenerateDataAndChannels(
				numIndex,
				numData,
				samplesPerDomain,
			)

			Expect(chs).To(HaveLen(numIndex + numData))
			for i := range numIndex {
				Expect(chs[i].IsIndex).To(BeTrue())
				Expect(keys[i]).To(Equal(cesium.ChannelKey(i + 1)))
			}
			for i := numIndex; i < numIndex+numData; i++ {
				Expect(chs[i].Index).To(Equal(cesium.ChannelKey((i+1)%numIndex + 1)))
				Expect(keys[i]).To(Equal(cesium.ChannelKey(i + 1)))
			}

			// Assert that the data channel has the right length
			Expect(data.Len()).To(Equal(int64(samplesPerDomain)))
		},
		Entry("normal", 1, 2, 2),
		Entry("many indices", 3, 5, 3),
		Entry("more indices than data", 10, 5, 15),
		Entry("big", 10, 2342, 400),
	)
})
