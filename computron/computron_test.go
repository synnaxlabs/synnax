package computron_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/computron"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Computron", func() {
	It("Should correctly evaluate an expression", func() {
		s1 := MustSucceed(computron.New(telem.NewSeriesV[int64](1, 2, 3)))
		s2 := MustSucceed(computron.New(telem.NewSeriesV[int64](4, 5, 6)))
		s3 := MustSucceed(computron.Exec("result = s1 + s2", map[string]interface{}{"s1": s1, "s2": s2}))
		Expect(s3).To(Equal(telem.NewSeriesV[int64](5, 7, 9)))
	})
})
