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
	"os"
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/x/testutil"
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
				i := i
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

	Describe("File Systems", func() {
		It("Should generate factories for os-based FS and memory-based FS", func() {
			fs := FileSystems
			_, ok := fs["memFS"]
			Expect(ok).To(BeTrue())
			_, ok = fs["osFS"]
			Expect(ok).To(BeTrue())
		})
	})

	Describe("CopyFS", func() {
		for fsName, makeFS := range FileSystems {
			Context("FS: "+fsName, func() {
				It("Should copy one fs entirely from one place to another", func() {
					fs, cleanUp := makeFS()
					sub1 := MustSucceed(fs.Sub("sub1"))
					sub2 := MustSucceed(fs.Sub("sub2"))

					By("Creating various files and directories in sub1", func() {
						// sub1
						//   - subsub1
						//       - subsubsub1
						//          - file3
						//       - file2
						//   - subsub2
						//   - file1
						subsub1 := MustSucceed(sub1.Sub("subsub1"))
						MustSucceed(sub1.Sub("subsub2"))
						f1 := MustSucceed(sub1.Open("file1", os.O_CREATE))
						Expect(f1.Close()).To(Succeed())
						subsubsub1 := MustSucceed(subsub1.Sub("subsubsub1"))
						f2 := MustSucceed(subsub1.Open("file2", os.O_CREATE))
						Expect(f2.Close()).To(Succeed())
						f3 := MustSucceed(subsubsub1.Open("file3", os.O_CREATE))
						Expect(f3.Close()).To(Succeed())
					})

					By("Copying the FS")
					Expect(CopyFS(sub1, sub2)).To(Succeed())

					By("Asserting it was an exact replica")
					infoToName := func(i os.FileInfo, _ int) string { return i.Name() }
					sub1list := lo.Map(MustSucceed(sub1.List("")), infoToName)
					sub2list := lo.Map(MustSucceed(sub2.List("")), infoToName)
					Expect(sub1list).To(Equal(sub2list))

					subsub1list := lo.Map(MustSucceed(sub1.List("subsub1")), infoToName)
					subsub2list := lo.Map(MustSucceed(sub2.List("subsub1")), infoToName)
					Expect(subsub1list).To(Equal(subsub2list))

					subsubsub1list := lo.Map(MustSucceed(sub1.List("subsub1/subsubsub1")), infoToName)
					subsubsub2list := lo.Map(MustSucceed(sub2.List("subsub1/subsubsub1")), infoToName)
					Expect(subsubsub1list).To(Equal(subsubsub2list))

					Expect(cleanUp()).To(Succeed())
				})
			})
		}
	})

	DescribeTable("GenerateFrameAndChannels", func(numIndex, numData, samplesPerDomain int) {
		data, chs, keys := GenerateDataAndChannels(numIndex, numData, samplesPerDomain)

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
