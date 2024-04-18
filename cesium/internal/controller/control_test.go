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
	"github.com/cockroachdb/errors"
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
			g, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(g).ToNot(BeNil())
		})
		It("Should return an error when the configuration is invalid", func() {
			c := controller.New[testEntity](control.Exclusive)
			_, _, err := c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeZero,
				Subject:   control.Subject{Key: "a"},
			}, createEntityAndNoError)
			Expect(err).To(MatchError(ContainSubstring("TimeRange must be non-zero")))
		})
		It("Should return an error when the configuration is invalid", func() {
			c := controller.New[testEntity](control.Exclusive)
			_, _, err := c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Subject:   control.Subject{Key: ""},
			}, createEntityAndNoError)
			Expect(err).To(MatchError(ContainSubstring("subject.key must be set")))
		})
		It("Should return an error when opening a gate of same name", func() {
			c := controller.New[testEntity](control.Exclusive)
			g, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
			Expect(g).ToNot(BeNil())

			_, _, err := c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError)
			Expect(err).To(MatchError(ContainSubstring("[controller] - gate with subject key test already exists")))
		})
	})

	Describe("LeadingState", func() {
		It("Should return the leading State of the controller", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			g, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key:  "test",
					Name: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.IsAcquire()).To(BeTrue())
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
			_, _, err := c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRange{Start: 5, End: 15},
			}, createEntityAndNoError)
			Expect(err).To(HaveOccurred())
		})
		It("Should return an error if callback is invalid", func() {
			c := controller.New[testEntity](control.Exclusive)
			_, _, err := c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRange{Start: 5, End: 15},
			}, func() (testEntity, error) {
				return testEntity{value: 11}, errors.New("haha error")
			})
			Expect(err).To(MatchError(Equal("haha error")))
		})
		It("Should work if a new region was created", func() {
			c := controller.New[testEntity](control.Exclusive)
			g, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(g).ToNot(BeNil())
		})
		It("Should work if a new region was not created", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 12})).To(Succeed())
			g, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(g).ToNot(BeNil())
		})
		It("Should open a control gate for the given time range", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			_, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.From).To(BeNil())
			Expect(t.To).ToNot(BeNil())
		})
	})

	Context("Single Gate", func() {
		It("Already existing region", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 10})).To(Succeed())
			g, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject: control.Subject{
					Key: "test",
				},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			v, authorized := g.Authorize()
			Expect(authorized).To(BeTrue())
			Expect(v.value).To(Equal(10))
		})
		It("Should delete a region when all gates from that region are removed", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{value: 11})).To(Succeed())
			g, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject:   control.Subject{Key: "test"},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			v, t := g.Release()
			Expect(t.Occurred()).To(BeTrue())
			Expect(t.IsRelease()).To(BeTrue())
			Expect(v.value).To(Equal(11))
			By("Returning false when opening a new gate")
			_, t = MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				Subject:   control.Subject{Key: "test2"},
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
		})
	})

	Context("Multiple Gates", func() {
		Context("Exclusive Control", func() {
			It("Should authorize the gate with the highest authority", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.From).To(BeNil())
				Expect(t.To).ToNot(BeNil())
				Expect(t.To.Subject).To(Equal(control.Subject{Key: "g1"}))
				g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeFalse())
			})
			It("Should authorize the most recently opened gate if gates are equal", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeFalse())
			})
			It("Should return control to the next highest authority when the highest authority gate is released", func() {
				c := controller.New[testEntity](control.Exclusive)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
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
						g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g1"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeTrue())
						g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g2"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 2,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeFalse())
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
							g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g1"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeTrue())
							g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g2"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeFalse())
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
							g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g1"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute - 1,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeTrue())
							g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
								Subject:   control.Subject{Key: "g2"},
								TimeRange: telem.TimeRangeMax,
								Authority: control.Absolute,
							}, createEntityAndNoError))
							Expect(t.Occurred()).To(BeTrue())
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
						g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g1"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeTrue())
						g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
							Subject:   control.Subject{Key: "g2"},
							TimeRange: telem.TimeRangeMax,
							Authority: control.Absolute - 1,
						}, createEntityAndNoError))
						Expect(t.Occurred()).To(BeFalse())
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
				g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute - 1,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeFalse())
			})
			It("Should authorize gates with equal authority", func() {
				c := controller.New[testEntity](control.Shared)
				Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
				g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g1"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeTrue())
				g2, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
					Subject:   control.Subject{Key: "g2"},
					TimeRange: telem.TimeRangeMax,
					Authority: control.Absolute,
				}, createEntityAndNoError))
				Expect(t.Occurred()).To(BeFalse())
				_, authorized := g1.Authorize()
				Expect(authorized).To(BeTrue())
				_, authorized = g2.Authorize()
				Expect(authorized).To(BeTrue())
			})
		})
	})
	Context("OpenAbsoluteGateIfUncontrolled", func() {
		It("Should take control when there are no other gates in the region", func() {
			c := controller.New[testEntity](control.Exclusive)
			Expect(c.Register(telem.TimeRangeMax, testEntity{})).To(Succeed())
			By("Getting an absolute gate on an uncontrolled region")
			g, t := MustSucceed2(c.OpenAbsoluteGateIfUncontrolled(telem.TimeRangeMax, control.Subject{Key: "g"}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			_, authorized := g.Authorize()
			Expect(authorized).To(BeTrue())

			By("Creating another gate on that region")
			g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRangeMax,
				Authority: control.Absolute,
				Subject:   control.Subject{Key: "g1"},
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeFalse())
			_, authorized = g1.Authorize()
			Expect(authorized).To(BeFalse())
			_, authorized = g.Authorize()
			Expect(authorized).To(BeTrue())
		})

		It("Should fail when there is another gate in the region", func() {
			c := controller.New[testEntity](control.Exclusive)
			g1, t := MustSucceed2(c.OpenGateAndMaybeRegister(controller.GateConfig{
				TimeRange: telem.TimeRange{
					Start: 10 * telem.SecondTS,
					End:   100 * telem.SecondTS,
				},
				Authority: 0,
				Subject:   control.Subject{Key: "g1"},
			}, createEntityAndNoError))
			Expect(t.Occurred()).To(BeTrue())
			_, authorized := g1.Authorize()
			Expect(authorized).To(BeTrue())

			_, t, err := c.OpenAbsoluteGateIfUncontrolled(telem.TimeRange{
				Start: 99 * telem.SecondTS,
				End:   110 * telem.SecondTS,
			}, control.Subject{Key: "g2"}, createEntityAndNoError)
			Expect(t.Occurred()).To(BeFalse())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already being controlled"))
		})
	})
})
