// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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

func createEntityAndNoError() (t testEntity, err error) {
	return testEntity{}, nil
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

	Describe("RegisterRegionAndOpenGate", func() {
		It("Should register a region and open a gate at the same time", func() {
			c := controller.New[testEntity](control.Exclusive)
			g, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(g).ToNot(BeNil())
		})
	})

	Describe("LeadingState", func() {
		It("Should return the leading State of the controller", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			g, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key:  "test",
					Name: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(createdRegion).To(BeFalse())
			Expect(g).ToNot(BeNil())
			lead := c.LeadingState()
			Expect(lead).ToNot(BeNil())
			Expect(lead.Subject).To(Equal(control.Subject{
				Key:  "test",
				Name: "test",
			}))
		})
	})

	Describe("OpenGateAndMaybeRegister", func() {
		It("Should return an error if the gate overlaps with multiple regions", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRange{Start: 0, End: 10}, testEntity{})).To(Succeed())
			Expect(c.Register(telem.TimeRange{Start: 10, End: 20}, testEntity{})).To(Succeed())
			_, _, _, err := c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRange{Start: 5, End: 15},
			}, createEntityAndNoError)
			Expect(err).To(HaveOccurred())
		})
		It("Should return true if a new region was created", func() {
			c := controller.New[testEntity](control.Exclusive)
			g, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeTrue())
			Expect(g).ToNot(BeNil())
		})
		It("Should return false if a new region was not created", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 12})).To(Succeed())
			g, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeFalse())
			Expect(g).ToNot(BeNil())
		})
		It("Should open a control gate for the given time range", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			_, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(createdRegion).To(BeFalse())
		})
	})

	Context("Single Gate", func() {
		It("Already existing region", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 10})).To(Succeed())
			g, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeFalse())
			v, authorized := g.Authorize()
			Expect(authorized).To(BeTrue())
			Expect(v.value).To(Equal(10))
		})
		It("Should delete a region when all gates from that region are removed", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 11})).To(Succeed())
			g, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject:   control.Subject{Key: "test"},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeFalse())
			v, t := g.Release()
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.IsRelease()).To(BeTrue())
			Expect(v.value).To(Equal(11))
			By("Returning false when opening a new gate")
			_, t, createdRegion = MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject:   control.Subject{Key: "test2"},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeTrue())
		})
	})

	Context("Multiple Gates", func() {
		Context("Exclusive Control", func() {
			It("Should authorize the gate with the highest authority", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.From).To(BeNil())
				Expect(t.To).ToNot(BeNil())
				Expect(t.To.Subject).To(Equal(control.Subject{Key: "g1"}))
				Expect(createdRegion).To(BeFalse())
				g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				Expect(createdRegion).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeFalse())
			})
			It("Should authorize the most recently opened gate if gates are equal", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				Expect(createdRegion).To(BeFalse())
				g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				Expect(createdRegion).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeFalse())
			})
			It("Should return control to the next highest authority when the highest authority gate is released", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				Expect(createdRegion).To(BeFalse())
				g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				Expect(createdRegion).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				By("Returning false that the region is released")
				_, t = g1.Release()
				Expect(t.IsRelease()).To(BeFalse())
				Expect(t.From).ToNot(BeNil())
				Expect(t.To).ToNot(BeNil())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeTrue())
			})
			Describe("SetAuthority", func() {
				Context("To higher authority than all other gates", func() {
					It("Should transfer authority to the gate that called SetAuthority", func() {
						c := controller.New[testEntity](control.Exclusive)
						Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
						g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g1"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeTrue())
						Expect(createdRegion).To(BeFalse())
						g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g2"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 2,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeFalse())
						Expect(createdRegion).To(BeFalse())
						_, authorized := g1.Authorize()
						Expect(authorized).To(BeTrue())
						_, authorized = g2.Authorize()
						Expect(authorized).To(BeFalse())
						t = g2.SetAuthority(control.Absolute)
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From.Subject).To(Equal(control.Subject{Key: "g1"}))
						Expect(t.To.Subject).To(Equal(control.Subject{Key: "g2"}))
						_, authorized = g2.Authorize()
						Expect(authorized).To(BeTrue())
					})
				})
				Context("To the same authority as highest gate", func() {
					Context("Where the next highest gate has a less precedent position", func() {
						It("Should not transfer authority", func() {
							c := controller.New[testEntity](control.Exclusive)
							Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
							g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g1"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeTrue())
							Expect(createdRegion).To(BeFalse())
							g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g2"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeFalse())
							Expect(createdRegion).To(BeFalse())
							_, authorized := g1.Authorize()
							Expect(authorized).To(BeTrue())
							_, authorized = g2.Authorize()
							Expect(authorized).To(BeFalse())
							t = g1.SetAuthority(control.Absolute - 1)
							Expect(t.Occurred()).To(BeTrue())
							_, authorized = g2.Authorize()
							Expect(authorized).To(BeFalse())
							_, authorized = g1.Authorize()
							Expect(authorized).To(BeTrue())
						})
					})
					Context("Where the next highest gate has a more precedent position", func() {
						It("Should transfer authority to the next highest gate", func() {
							c := controller.New[testEntity](control.Exclusive)
							Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
							g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g1"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeTrue())
							Expect(createdRegion).To(BeFalse())
							g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g2"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeTrue())
							Expect(createdRegion).To(BeFalse())
							_, authorized := g1.Authorize()
							Expect(authorized).To(BeFalse())
							_, authorized = g2.Authorize()
							Expect(authorized).To(BeTrue())
							t = g2.SetAuthority(control.Absolute - 1)
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From.Subject).To(Equal(control.Subject{Key: "g2"}))
							Expect(t.To.Subject).To(Equal(control.Subject{Key: "g1"}))
							_, authorized = g2.Authorize()
							Expect(authorized).To(BeFalse())
							_, authorized = g1.Authorize()
							Expect(authorized).To(BeTrue())
						})
					})
				})
				Context("To a lower authority than the next highest gate", func() {
					It("Should transfer authority to the next highest gate", func() {
						c := controller.New[testEntity](control.Exclusive)
						Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
						g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g1"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeTrue())
						Expect(createdRegion).To(BeFalse())
						g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g2"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeFalse())
						Expect(createdRegion).To(BeFalse())
						_, authorized := g1.Authorize()
						Expect(authorized).To(BeTrue())
						_, authorized = g2.Authorize()
						Expect(authorized).To(BeFalse())
						t = g1.SetAuthority(control.Absolute - 2)
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From.Subject).To(Equal(control.Subject{Key: "g1"}))
						Expect(t.To.Subject).To(Equal(control.Subject{Key: "g2"}))
						_, authorized = g2.Authorize()
						Expect(authorized).To(BeTrue())
						_, authorized = g1.Authorize()
						Expect(authorized).To(BeFalse())
					})
				})
			})
		})
		Context("Shared Control", func() {
			It("Should authorize gate with the highest authority", func() {
				c := controller.New[testEntity](control.Shared)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				Expect(createdRegion).To(BeFalse())
				g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				Expect(createdRegion).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeFalse())
			})
			It("Should authorize gates with equal authority", func() {
				c := controller.New[testEntity](control.Shared)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				Expect(createdRegion).To(BeFalse())
				g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				Expect(createdRegion).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeTrue())
			})
		})
	})

	Context("Stealth", func() {
		It("Should let stealth gate take control when it is the only one", func() {
			c := controller.New[testEntity](control.Exclusive)
			stealthGate, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: 0,
				Subject:   control.Subject{Key: "stealthGate"},
				Stealth:   true,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeTrue())
			_, authorized := stealthGate.Authorize()
			Expect(authorized).To(BeTrue())
		})
		It("Should be in stealth until all other writers are released", func() {
			c := controller.New[testEntity](control.Exclusive)
			By("Making a basic gate")
			g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: 0,
				Subject:   control.Subject{Key: "g1"},
				Stealth:   false,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeTrue())
			_, authorized := g1.Authorize()
			Expect(authorized).To(BeTrue())

			By("Making a stealth gate")
			stealthGate, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: 0,
				Subject:   control.Subject{Key: "stealthGate"},
				Stealth:   true,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeFalse())
			Expect(createdRegion).To(BeFalse())
			_, authorized = stealthGate.Authorize()
			Expect(authorized).To(BeFalse())
			_, authorized = g1.Authorize()
			Expect(authorized).To(BeTrue())

			By("Releasing the basic gate")
			_, t = g1.Release()
			Expect(t.Occurred()).To(BeTrue())

			By("Now, stealth gate should come out of stealth")
			_, authorized = stealthGate.Authorize()
			Expect(authorized).To(BeTrue())

			By("This stealth gate should have absolute authority")
			g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
				Subject:   control.Subject{Key: "g2"},
				Stealth:   false,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeFalse())
			Expect(createdRegion).To(BeFalse())
			_, authorized = g2.Authorize()
			Expect(authorized).To(BeFalse())
			_, authorized = stealthGate.Authorize()
			Expect(authorized).To(BeTrue())

			By("Releasing the stealth gate")
			_, t = stealthGate.Release()
			Expect(t.Occurred()).To(BeTrue())
			_, authorized = g2.Authorize()
			Expect(authorized).To(BeTrue())
		})
		It("Should not take control when there is another gate of authority 0", func() {
			c := controller.New[testEntity](control.Exclusive)
			By("Making a basic gate")
			g1, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: 0,
				Subject:   control.Subject{Key: "g1"},
				Stealth:   false,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(createdRegion).To(BeTrue())
			_, authorized := g1.Authorize()
			Expect(authorized).To(BeTrue())

			By("Making a stealth gate")
			stealthGate, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: 0,
				Subject:   control.Subject{Key: "stealthGate"},
				Stealth:   true,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeFalse())
			Expect(createdRegion).To(BeFalse())
			_, authorized = stealthGate.Authorize()
			Expect(authorized).To(BeFalse())
			_, authorized = g1.Authorize()
			Expect(authorized).To(BeTrue())

			g2, t, createdRegion := MustSucceed3(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: 0,
				Subject:   control.Subject{Key: "g2"},
				Stealth:   false,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeFalse())
			Expect(createdRegion).To(BeFalse())

			By("This stealth gate should have lower authority than the gate with 0 priority")
			_, t = g1.Release()
			Expect(t.Occurred()).To(BeTrue())

			_, authorized = stealthGate.Authorize()
			Expect(authorized).To(BeFalse())
			_, authorized = g2.Authorize()
			Expect(authorized).To(BeTrue())

			By("Releasing the other gate")
			_, t = g2.Release()
			Expect(t.Occurred()).To(BeTrue())
			_, authorized = stealthGate.Authorize()
			Expect(authorized).To(BeTrue())
		})
	})
})
