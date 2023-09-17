package control_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/storage/control"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Control", func() {

	It("Should allow one gate to override the authority of another gate", func() {
		svc := control.NewService[int]()
		g1 := svc.OpenGate(telem.TimeRangeMax)
		g1.Set([]int{1}, []control.Authority{control.Authority(1)})
		Expect(g1.Check([]int{1})).To(BeEmpty())
		g2 := svc.OpenGate(telem.TimeRangeMax)
		g2.Set([]int{1}, []control.Authority{control.Authority(2)})
		Expect(g1.Check([]int{1})).To(Equal([]int{1}))
		Expect(g2.Check([]int{1})).To(BeEmpty())
	})

	It("Should allow a gate to remove its authority over a key", func() {
		svc := control.NewService[int]()
		g1 := svc.OpenGate(telem.TimeRangeMax)
		g1.Set([]int{1}, []control.Authority{control.Authority(1)})
		Expect(g1.Check([]int{1})).To(BeEmpty())
		g1.Delete(1)
		Expect(g1.Check([]int{1})).To(Equal([]int{1}))
	})

	It("Should allow two gates to have authority over the same key", func() {
		svc := control.NewService[int]()
		g1 := svc.OpenGate(telem.TimeRangeMax)
		g1.Set([]int{1}, []control.Authority{control.Authority(1)})
		Expect(g1.Check([]int{1})).To(BeEmpty())
		g2 := svc.OpenGate(telem.TimeRangeMax)
		g2.Set([]int{1}, []control.Authority{control.Authority(1)})
		Expect(g1.Check([]int{1})).To(BeEmpty())
		Expect(g2.Check([]int{1})).To(BeEmpty())
	})

	It("Should allow a gate to remain in control over a different time range", func() {
		svc := control.NewService[int]()
		g1 := svc.OpenGate(telem.TimeRange{Start: 2, End: 3})
		g1.Set([]int{1}, []control.Authority{control.Authority(1)})
		Expect(g1.Check([]int{1})).To(BeEmpty())
		g2 := svc.OpenGate(telem.TimeRange{Start: 0, End: 1})
		g2.Set([]int{1}, []control.Authority{control.Authority(2)})
		Expect(g1.Check([]int{1})).To(BeEmpty())
		Expect(g2.Check([]int{1})).To(BeEmpty())
	})

	It("Should remove a gates' authority after it is closed", func() {
		svc := control.NewService[int]()
		g1 := svc.OpenGate(telem.TimeRangeMax)
		g1.Set([]int{1}, []control.Authority{control.Authority(2)})
		g2 := svc.OpenGate(telem.TimeRangeMax)
		g2.Set([]int{1}, []control.Authority{control.Authority(1)})
		Expect(g1.Check([]int{1})).To(BeEmpty())
		Expect(g2.Check([]int{1})).To(Equal([]int{1}))
		g1.Close()
		Expect(g2.Check([]int{1})).To(BeEmpty())
	})

	It("Shou;d panic if an op is attempted on a closed gate", func() {
		svc := control.NewService[int]()
		g1 := svc.OpenGate(telem.TimeRangeMax)
		g1.Close()
		Expect(func() { g1.Close() }).To(Panic())
		Expect(func() { g1.Set([]int{1}, []control.Authority{control.Authority(1)}) }).To(Panic())
		Expect(func() { g1.Delete(1) }).To(Panic())
		Expect(func() { g1.Check([]int{1}) }).To(Panic())

	})
})
