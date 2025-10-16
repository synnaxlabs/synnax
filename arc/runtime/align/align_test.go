package align_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/align"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Align", func() {
	It("should produce correct outputs for normal data", func() {
		a := align.NewAligner([]string{ir.LHSInputParam, ir.RHSInputParam})
		firstLhs := telem.NewSeriesV[float32](1)
		firstTime := telem.NewSeriesSecondsTSV(1)
		Expect(a.Add(ir.LHSInputParam, firstLhs, firstTime)).To(Succeed())
		_, ready := a.Next()
		Expect(ready).To(BeFalse())
		firstRhs := telem.NewSeriesV[float32](1)
		firstTime = telem.NewSeriesSecondsTSV(1)
		Expect(a.Add(ir.RHSInputParam, firstRhs, firstTime)).To(Succeed())
		ops, ready := a.Next()
		Expect(ready).To(BeTrue())
		Expect(ops.Inputs).To(HaveLen(2))
	})

	It("should produce correct outputs for normal data", func() {
		a := align.NewAligner([]string{ir.LHSInputParam, ir.RHSInputParam})
		firstLhs := telem.NewSeriesV[float32](20)
		firstLhs.Alignment = telem.NewAlignment(4293967296, 0)
		firstTime := telem.NewSeriesSecondsTSV(1)
		firstTime.Alignment = firstLhs.Alignment
		Expect(a.Add(ir.LHSInputParam, firstLhs, firstTime)).To(Succeed())
		_, ready := a.Next()
		Expect(ready).To(BeFalse())
		firstRhs := telem.NewSeriesV[float32](10)
		firstRHSTime := telem.NewSeriesSecondsTSV(1)
		Expect(a.Add(ir.RHSInputParam, firstRhs, firstRHSTime)).To(Succeed())
		ops, ready := a.Next()
		Expect(ready).To(BeTrue())
		Expect(ops.Inputs).To(HaveLen(2))
	})

	It("Should handle data with different timestamps", func() {
		a := align.NewAligner([]string{ir.LHSInputParam, ir.RHSInputParam})
		firstLhs := telem.NewSeriesV[float32](1)
		firstTime := telem.NewSeriesSecondsTSV(1)
		Expect(a.Add(ir.LHSInputParam, firstLhs, firstTime)).To(Succeed())
		_, ready := a.Next()
		Expect(ready).To(BeFalse())
		firstRhs := telem.NewSeriesV[float32](1)
		firstTime = telem.NewSeriesSecondsTSV(2)
		Expect(a.Add(ir.RHSInputParam, firstRhs, firstTime)).To(Succeed())
		ops, ready := a.Next()
		Expect(ready).To(BeTrue())
		Expect(ops.Inputs).To(HaveLen(2))
	})

	It("Should handle multiple series arrivals", func() {
		a := align.NewAligner([]string{ir.LHSInputParam, ir.RHSInputParam})
		firstLhs := telem.NewSeriesV[float32](1)
		firstTime := telem.NewSeriesSecondsTSV(1)
		Expect(a.Add(ir.LHSInputParam, firstLhs, firstTime)).To(Succeed())
		_, ready := a.Next()
		Expect(ready).To(BeFalse())
		secondLhs := telem.NewSeriesV[float32](1)
		secondTime := telem.NewSeriesSecondsTSV(2)
		Expect(a.Add(ir.LHSInputParam, secondLhs, secondTime)).To(Succeed())
		_, ready = a.Next()
		Expect(ready).To(BeFalse())
		firstRhs := telem.NewSeriesV[float32](1)
		firstTime = telem.NewSeriesSecondsTSV(2)
		Expect(a.Add(ir.RHSInputParam, firstRhs, firstTime)).To(Succeed())
		ops, ready := a.Next()
		Expect(ready).To(BeTrue())
		Expect(ops.Inputs).To(HaveLen(2))
		ops, ready = a.Next()
		Expect(ready).To(BeTrue())
		Expect(ops.Inputs).To(HaveLen(2))
	})
})
