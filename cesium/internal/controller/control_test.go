package controller_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
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

	Describe("RegisterAndOpenGate", func() {
		It("Should register a region and open a gate at the same time", func() {
			c := controller.New[testEntity](control.Exclusive)
			g, t := MustSucceed2(c.RegisterAndOpenGate(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, testEntity{}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(g).ToNot(BeNil())
		})
	})

	Describe("LeadingState", func() {
		It("Should return the leading State of the controller", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			g, t := MustSucceed2(c.RegisterAndOpenGate(controller.GateConfig{
				Subject: control.Subject{
					Key:  "test",
					Name: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, testEntity{}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(g).ToNot(BeNil())
			lead := c.LeadingState()
			Expect(lead).ToNot(BeNil())
			Expect(lead.Subject).To(Equal(control.Subject{
				Key:  "test",
				Name: "test",
			}))
		})
	})

	Describe("OpenGate", func() {
		It("Should return an error if the gate overlaps with multiple regions", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRange{Start: 0, End: 10}, testEntity{})).To(Succeed())
			Expect(c.Register(telem.TimeRange{Start: 10, End: 20}, testEntity{})).To(Succeed())
			_, _, _, err := c.OpenGate(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRange{Start: 5, End: 15},
			})
			Expect(err).To(HaveOccurred())
		})
		It("Should return false if no region exists for the given time range", func() {
			c := controller.New[testEntity](control.Exclusive)
			g, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}))
			Expect(t.Occurred()).To(BeFalse())
			Expect(exists).To(BeFalse())
			Expect(g).To(BeNil())
		})
		It("Should open a control gate for the given time range", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			_, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(exists).To(BeTrue())
		})
	})

	Context("Single Gate", func() {
		It("Should always return true", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 10})).To(Succeed())
			g, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(exists).To(BeTrue())
			v, exists := g.Authorize()
			Expect(exists).To(BeTrue())
			Expect(v.value).To(Equal(10))
		})
		It("Should return true when the gate is released", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 11})).To(Succeed())
			g, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
				Subject:   control.Subject{Key: "test"},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}))
			Expect(t.Occurred()).To(BeTrue())
			Expect(exists).To(BeTrue())
			v, t := g.Release()
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.IsRelease()).To(BeTrue())
			Expect(v.value).To(Equal(11))
			By("Returning false when opening a new gate")
			_, t, exists = MustSucceed3(c.OpenGate(controller.GateConfig{
				Subject:   control.Subject{Key: "test2"},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}))
			Expect(t.Occurred()).To(BeFalse())
			Expect(exists).To(BeFalse())
		})
	})

	Context("Multiple Gates", func() {
		Context("Exclusive Control", func() {
			It("Should authorize the gate with the highest authority", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.From).To(BeNil())
				Expect(t.To).ToNot(BeNil())
				Expect(t.To.Subject).To(Equal(control.Subject{Key: "g1"}))
				Expect(exists).To(BeTrue())
				g2, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}))
				Expect(t.Occurred()).To(BeFalse())
				Expect(exists).To(BeTrue())
				_, exists = g1.Authorize()
				Expect(exists).To(BeTrue())
				_, exists = g2.Authorize()
				Expect(exists).To(BeFalse())
			})
			It("Should authorize the most recently opened gate if gates are equal", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(exists).To(BeTrue())
				g2, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}))
				Expect(t.Occurred()).To(BeFalse())
				Expect(exists).To(BeTrue())
				_, exists = g1.Authorize()
				Expect(exists).To(BeTrue())
				_, exists = g2.Authorize()
				Expect(exists).To(BeFalse())
			})
			It("Should return control to the next highest authority when the highest authority gate is released", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(exists).To(BeTrue())
				g2, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}))
				Expect(t.Occurred()).To(BeFalse())
				Expect(exists).To(BeTrue())
				_, exists = g1.Authorize()
				Expect(exists).To(BeTrue())
				By("Returning false that the region is released")
				_, t = g1.Release()
				Expect(t.IsRelease()).To(BeFalse())
				Expect(t.From).ToNot(BeNil())
				Expect(t.To).ToNot(BeNil())
				_, exists = g2.Authorize()
				Expect(exists).To(BeTrue())
			})
			Describe("SetAuthority", func() {
				Context("To higher authority than all other gates", func() {
					It("Should transfer authority to the gate that called SetAuthority", func() {
						c := controller.New[testEntity](control.Exclusive)
						Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
						g1, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
							Subject:   control.Subject{Key: "g1"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						}))
						Expect(t.Occurred()).To(BeTrue())
						Expect(exists).To(BeTrue())
						g2, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
							Subject:   control.Subject{Key: "g2"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 2,
						}))
						Expect(t.Occurred()).To(BeFalse())
						Expect(exists).To(BeTrue())
						_, exists = g1.Authorize()
						Expect(exists).To(BeTrue())
						_, exists = g2.Authorize()
						Expect(exists).To(BeFalse())
						t = g2.SetAuthority(control.Absolute)
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From.Subject).To(Equal(control.Subject{Key: "g1"}))
						Expect(t.To.Subject).To(Equal(control.Subject{Key: "g2"}))
						_, exists = g2.Authorize()
						Expect(exists).To(BeTrue())
					})
				})
				Context("To the same authority as highest gate", func() {
					Context("Where the next highest gate has a less precedent position", func() {
						It("Should not transfer authority", func() {
							c := controller.New[testEntity](control.Exclusive)
							Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
							g1, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
								Subject:   control.Subject{Key: "g1"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							}))
							Expect(t.Occurred()).To(BeTrue())
							Expect(exists).To(BeTrue())
							g2, t, exists := MustSucceed3(c.OpenGate(controller.GateConfig{
								Subject:   control.Subject{Key: "g2"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							}))
							Expect(t.Occurred()).To(BeFalse())
							Expect(exists).To(BeTrue())
							_, exists = g1.Authorize()
							Expect(exists).To(BeTrue())
							_, exists = g2.Authorize()
							Expect(exists).To(BeFalse())
							t = g1.SetAuthority(control.Absolute - 1)
							Expect(t.Occurred()).To(BeFalse())
							_, exists = g2.Authorize()
							Expect(exists).To(BeFalse())
							_, exists = g1.Authorize()
							Expect(exists).To(BeTrue())
						})
					})
					Context("Where the next highest gate has a more precedent position", func() {
						It("Should transfer authority to the next highest gate", func() {
							c := controller.New[testEntity](control.Exclusive)
							Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
							g1, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
								Subject:   control.Subject{Key: "g1"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							}))
							Expect(t.Occurred()).To(BeTrue())
							Expect(ok).To(BeTrue())
							g2, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
								Subject:   control.Subject{Key: "g2"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							}))
							Expect(t.Occurred()).To(BeTrue())
							Expect(ok).To(BeTrue())
							_, ok = g1.Authorize()
							Expect(ok).To(BeFalse())
							_, ok = g2.Authorize()
							Expect(ok).To(BeTrue())
							t = g2.SetAuthority(control.Absolute - 1)
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From.Subject).To(Equal(control.Subject{Key: "g2"}))
							Expect(t.To.Subject).To(Equal(control.Subject{Key: "g1"}))
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
						g1, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
							Subject:   control.Subject{Key: "g1"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute,
						}))
						Expect(t.Occurred()).To(BeTrue())
						g2, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
							Subject:   control.Subject{Key: "g2"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						}))
						Expect(t.Occurred()).To(BeFalse())
						Expect(ok).To(BeTrue())
						_, ok = g1.Authorize()
						Expect(ok).To(BeTrue())
						_, ok = g2.Authorize()
						Expect(ok).To(BeFalse())
						t = g1.SetAuthority(control.Absolute - 2)
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From.Subject).To(Equal(control.Subject{Key: "g1"}))
						Expect(t.To.Subject).To(Equal(control.Subject{Key: "g2"}))
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
				g1, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(ok).To(BeTrue())
				g2, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}))
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
				g1, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}))
				Expect(t.Occurred()).To(BeTrue())
				Expect(ok).To(BeTrue())
				g2, t, ok := MustSucceed3(c.OpenGate(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}))
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
