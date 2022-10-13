package seeker_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/synnaxlabs/cesium/internal/seeker"
	"time"
)

var _ = Describe("RollingMemoryIndex", func() {
	Describe("Seek", func() {
		It("Should return the root position of the first alignment that is less than or equal to the position", func() {
			index := seeker.NewMemoryIndex(10)
			index.Add(2, 10)
			index.Add(4, 20)
			index.Add(6, 30)
			index.Add(8, 40)
			index.Add(10, 50)
			index.Add(12, 60)
			index.Add(14, 70)
			index.Add(16, 80)
			index.Add(18, 90)
			index.Add(20, 100)
			n, _ := index.Seek(3)
			Expect(n).To(Equal(10))
			n, _ = index.Seek(1)
			Expect(n).To(Equal(0))
			n, _ = index.Seek(5)
			Expect(n).To(Equal(20))
			n, _ = index.Seek(7)
			Expect(n).To(Equal(30))
			t0 := time.Now()
			n, _ = index.Seek(22)
			t1 := time.Now()
			logrus.Info("time: ", t1.Sub(t0))

		})
	})
	Describe("Performance", func() {
		It("Should be able to seek 1000000 times in less than 1 second", func() {
			index := seeker.NewMemoryIndex(1000000)
			for i := 0; i < 100; i++ {
				index.Add(i*5, i)
			}
			t0 := time.Now()
			index.Seek(50)
			t1 := time.Now()
			logrus.Info("time: ", t1.Sub(t0))
		})
	})

})
