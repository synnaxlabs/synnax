package testutil

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium"
	"sync"
)

var _ = Describe("Test Util Test", func() {
	Describe("Cesium GenerateChannelKey", func() {
		It("Should generate a unique channel key every time it is called", func() {
			var (
				keys = make([]cesium.ChannelKey, 1000)
				wg   = sync.WaitGroup{}
			)
			wg.Add(1000)
			for i := 0; i < 1000; i++ {
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
})
