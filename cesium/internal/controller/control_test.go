package controller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

type testEntity struct {
	value int
}

var _ = Describe("Control", func() {
	Describe("Register", func() {
		It("Should correctly register an entity with the controller", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
		})
		It("Should return an error if time range is already registered", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).ToNot(Succeed())
		})
	})
	Describe("OpenGate", func() {
		It("Should return false if no region exists for the given time range", func() {
			c := controller.New[testEntity](control.Exclusive)
			g, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
			Expect(ok).To(BeFalse())
			Expect(g).To(BeNil())
		})
		It("Should open a control gate for the given time range", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			_, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
			Expect(ok).To(BeTrue())
		})
	})
	Context("Single Gate", func() {
		It("Should always return true", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 10})).To(Succeed())
			g, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
			Expect(ok).To(BeTrue())
			v, ok := g.Authorize()
			Expect(ok).To(BeTrue())
			Expect(v.value).To(Equal(10))
		})
		It("Should return true when the gate is released", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 11})).To(Succeed())
			g, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
			Expect(ok).To(BeTrue())
			v, ok := g.Release()
			Expect(ok).To(BeTrue())
			Expect(v.value).To(Equal(11))
			By("Returning false when opening a new gate")
			_, ok = c.OpenGate(telem.TimeRangeMax, control.Absolute)
			Expect(ok).To(BeFalse())
		})
	})
	Context("Multiple Gates", func() {
		Context("Exclusive Control", func() {
			It("Should authorize the gate with the highest authority", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
				Expect(ok).To(BeTrue())
				g2, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute-1)
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeFalse())
			})
			It("Should authorize the most recently opened gate if authorities are equal", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
				Expect(ok).To(BeTrue())
				g2, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeFalse())
			})
			It("Should return control to the next highest authority when the highest authority gate is released", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
				Expect(ok).To(BeTrue())
				g2, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute-1)
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				By("Returning false that the region is released")
				_, regionReleased := g1.Release()
				Expect(regionReleased).To(BeFalse())
				_, ok = g2.Authorize()
				Expect(ok).To(BeTrue())
			})
		})
		Context("Shared Control", func() {
			It("Should authorize gate with the highest authority", func() {
				c := controller.New[testEntity](control.Shared)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
				Expect(ok).To(BeTrue())
				g2, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute-1)
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeFalse())
			})
			It("Should authorize gates with equal authority", func() {
				c := controller.New[testEntity](control.Shared)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
				Expect(ok).To(BeTrue())
				g2, ok := c.OpenGate(telem.TimeRangeMax, control.Absolute)
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeTrue())
			})
		})
	})
})
