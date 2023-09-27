package controller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

type testEntity struct {
	value int
}

func (t testEntity) ChannelKey() core.ChannelKey { return core.ChannelKey(0) }

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
			g, t, ok := c.OpenGate(controller.Config{
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			})
			Expect(t.Occurred()).To(BeFalse())
			Expect(ok).To(BeFalse())
			Expect(g).To(BeNil())
		})
		It("Should open a control gate for the given time range", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			_, t, ok := c.OpenGate(controller.Config{
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			})
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(ok).To(BeTrue())
		})
	})

	Context("Single Gate", func() {
		It("Should always return true", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 10})).To(Succeed())
			g, t, ok := c.OpenGate(controller.Config{
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			})
			Expect(t.Occurred()).To(BeTrue())
			Expect(ok).To(BeTrue())
			v, ok := g.Authorize()
			Expect(ok).To(BeTrue())
			Expect(v.value).To(Equal(10))
		})
		It("Should return true when the gate is released", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 11})).To(Succeed())
			g, t, ok := c.OpenGate(controller.Config{
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			})
			Expect(t.Occurred()).To(BeTrue())
			Expect(ok).To(BeTrue())
			v, t := g.Release()
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.IsRelease()).To(BeTrue())
			Expect(v.value).To(Equal(11))
			By("Returning false when opening a new gate")
			_, t, ok = c.OpenGate(controller.Config{
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			})
			Expect(t.Occurred()).To(BeFalse())
			Expect(ok).To(BeFalse())
		})
	})

	Context("Multiple Gates", func() {
		Context("Exclusive Control", func() {
			It("Should authorize the gate with the highest authority", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, ok := c.OpenGate(controller.Config{
					Name:      "g1",
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				})
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.From).To(BeNil())
				Expect(t.To).ToNot(BeNil())
				Expect(t.To.Subject).To(Equal("g1"))
				Expect(ok).To(BeTrue())
				g2, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				})
				Expect(t.Occurred()).To(BeFalse())
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeFalse())
			})
			It("Should authorize the most recently opened gate if gates are equal", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, ok := c.OpenGate(controller.Config{
					Name:      "g1",
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				})
				Expect(t.Occurred()).To(BeTrue())
				Expect(ok).To(BeTrue())
				g2, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				})
				Expect(t.Occurred()).To(BeFalse())
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeFalse())
			})
			It("Should return control to the next highest authority when the highest authority gate is released", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				})
				Expect(t.Occurred()).To(BeTrue())
				Expect(ok).To(BeTrue())
				g2, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				})
				Expect(t.Occurred()).To(BeFalse())
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				By("Returning false that the region is released")
				_, t = g1.Release()
				Expect(t.IsRelease()).To(BeFalse())
				Expect(t.From).ToNot(BeNil())
				Expect(t.To).ToNot(BeNil())
				_, ok = g2.Authorize()
				Expect(ok).To(BeTrue())
			})
			Describe("SetAuthority", func() {
				Context("To higher authority than all other gates", func() {
					It("Should transfer authority to the gate that called SetAuthority", func() {
						c := controller.New[testEntity](control.Exclusive)
						Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
						g1, t, ok := c.OpenGate(controller.Config{
							Name:      "g1",
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						})
						Expect(t.Occurred()).To(BeTrue())
						Expect(ok).To(BeTrue())
						g2, t, ok := c.OpenGate(controller.Config{
							Name:      "g2",
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 2,
						})
						Expect(t.Occurred()).To(BeFalse())
						Expect(ok).To(BeTrue())
						_, ok = g1.Authorize()
						Expect(ok).To(BeTrue())
						_, ok = g2.Authorize()
						Expect(ok).To(BeFalse())
						t = g2.SetAuthority(control.Absolute)
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From.Subject).To(Equal("g1"))
						Expect(t.To.Subject).To(Equal("g2"))
						_, ok = g2.Authorize()
						Expect(ok).To(BeTrue())
					})
				})
				Context("To the same authority as highest gate", func() {
					Context("Where the next highest gate has a less precedent position", func() {
						It("Should not transfer authority", func() {
							c := controller.New[testEntity](control.Exclusive)
							Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
							g1, t, ok := c.OpenGate(controller.Config{
								Name:      "g1",
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							})
							Expect(t.Occurred()).To(BeTrue())
							Expect(ok).To(BeTrue())
							g2, t, ok := c.OpenGate(controller.Config{
								Name:      "g2",
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							})
							Expect(t.Occurred()).To(BeFalse())
							Expect(ok).To(BeTrue())
							_, ok = g1.Authorize()
							Expect(ok).To(BeTrue())
							_, ok = g2.Authorize()
							Expect(ok).To(BeFalse())
							t = g1.SetAuthority(control.Absolute - 1)
							Expect(t.Occurred()).To(BeFalse())
							_, ok = g2.Authorize()
							Expect(ok).To(BeFalse())
							_, ok = g1.Authorize()
							Expect(ok).To(BeTrue())
						})
					})
					Context("Where the next highest gate has a more precedent position", func() {
						It("Should transfer authority to the next highest gate", func() {
							c := controller.New[testEntity](control.Exclusive)
							Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
							g1, t, ok := c.OpenGate(controller.Config{
								Name:      "g1",
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							})
							Expect(t.Occurred()).To(BeTrue())
							Expect(ok).To(BeTrue())
							g2, t, ok := c.OpenGate(controller.Config{
								Name:      "g2",
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							})
							Expect(t.Occurred()).To(BeTrue())
							Expect(ok).To(BeTrue())
							_, ok = g1.Authorize()
							Expect(ok).To(BeFalse())
							_, ok = g2.Authorize()
							Expect(ok).To(BeTrue())
							t = g2.SetAuthority(control.Absolute - 1)
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From.Subject).To(Equal("g2"))
							Expect(t.To.Subject).To(Equal("g1"))
							_, ok = g2.Authorize()
							Expect(ok).To(BeFalse())
							_, ok = g1.Authorize()
							Expect(ok).To(BeTrue())
						})
					})
				})
				Context("To a lower authority than the next highest gate", func() {
					It("Should transfer authority to the next highest gate", func() {
						c := controller.New[testEntity](control.Exclusive)
						Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
						g1, t, ok := c.OpenGate(controller.Config{
							Name:      "g1",
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute,
						})
						Expect(t.Occurred()).To(BeTrue())
						g2, t, ok := c.OpenGate(controller.Config{
							Name:      "g2",
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						})
						Expect(t.Occurred()).To(BeFalse())
						Expect(ok).To(BeTrue())
						_, ok = g1.Authorize()
						Expect(ok).To(BeTrue())
						_, ok = g2.Authorize()
						Expect(ok).To(BeFalse())
						t = g1.SetAuthority(control.Absolute - 2)
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From.Subject).To(Equal("g1"))
						Expect(t.To.Subject).To(Equal("g2"))
						_, ok = g2.Authorize()
						Expect(ok).To(BeTrue())
						_, ok = g1.Authorize()
						Expect(ok).To(BeFalse())
					})
				})
			})
		})
		Context("Shared Control", func() {
			It("Should authorize gate with the highest authority", func() {
				c := controller.New[testEntity](control.Shared)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				})
				Expect(t.Occurred()).To(BeTrue())
				Expect(ok).To(BeTrue())
				g2, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				})
				Expect(t.Occurred()).To(BeFalse())
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeFalse())
			})
			It("Should authorize gates with equal authority", func() {
				c := controller.New[testEntity](control.Shared)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				})
				Expect(t.Occurred()).To(BeTrue())
				Expect(ok).To(BeTrue())
				g2, t, ok := c.OpenGate(controller.Config{
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				})
				Expect(t.Occurred()).To(BeFalse())
				Expect(ok).To(BeTrue())
				_, ok = g1.Authorize()
				Expect(ok).To(BeTrue())
				_, ok = g2.Authorize()
				Expect(ok).To(BeTrue())
			})
		})
	})
})
