// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

type testResource struct {
	value int
}

var _ control.Resource = (*testResource)(nil)

func (t testResource) ChannelKey() channel.Key { return channel.Key(0) }

func createResourceNoErr(value int) (func() (t testResource, err error), func() int) {
	var count int
	return func() (t testResource, err error) {
			count++
			return testResource{value: value}, nil
		}, func() int {
			return count
		}
}

func baseConfig(value int) (control.GateConfig[testResource], func() int) {
	openResource, count := createResourceNoErr(value)
	return control.GateConfig[testResource]{
		Subject:      xcontrol.Subject{Key: "test", Name: "test"},
		TimeRange:    telem.TimeRangeMax,
		Authority:    xcontrol.AuthorityAbsolute,
		OpenResource: openResource,
	}, count
}

var _ = Describe("Control", func() {
	var (
		c   *control.Controller[testResource]
		cfg = control.Config{Concurrency: xcontrol.ConcurrencyExclusive}
	)
	JustBeforeEach(func() {
		c = MustSucceed(control.New[testResource](cfg))
	})

	Context("Exclusive", func() {
		BeforeEach(func() {
			cfg.Concurrency = xcontrol.ConcurrencyExclusive
		})

		Describe("OpenGate", func() {
			It("Should create a new region if the time range is not controller", func() {
				cfg, createCount := baseConfig(1)
				g, t := MustSucceed2(c.OpenGate(cfg))
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.IsAcquire()).To(BeTrue())
				Expect(t.IsTransfer()).To(BeFalse())
				Expect(createCount()).To(Equal(1))
				Expect(g).ToNot(BeNil())
				Expect(g.Authority()).To(Equal(xcontrol.AuthorityAbsolute))
				Expect(g.Subject().Key).To(Equal("test"))
				Expect(g.Subject().Name).To(Equal("test"))
			})

			It("Should not create a new region if the time range is already in the controller", func() {
				cfg1, createCount1 := baseConfig(1)
				cfg1.TimeRange = telem.NewRangeSeconds(1, 5)
				cfg1.Subject.Key = "test1"
				g1, t1 := MustSucceed2(c.OpenGate(cfg1))
				Expect(g1).ToNot(BeNil())
				Expect(t1.IsAcquire()).To(BeTrue())
				Expect(createCount1()).To(Equal(1))

				By("Not creating a new region when the time range is identical")
				cfg2, createCount2 := baseConfig(1)
				cfg2.TimeRange = telem.NewRangeSeconds(1, 5)
				cfg2.Subject.Key = "test2"
				g2, t2 := MustSucceed2(c.OpenGate(cfg2))
				Expect(g2).ToNot(BeNil())
				Expect(t2.Occurred()).To(BeFalse())
				Expect(createCount2()).To(Equal(0))

				By("Not creating a new region when the time ranges overlap")
				cfg3, createCount3 := baseConfig(1)
				cfg3.TimeRange = telem.NewRangeSeconds(1, 10)
				cfg3.Subject.Key = "test3"
				g3, t3 := MustSucceed2(c.OpenGate(cfg3))
				Expect(g3).ToNot(BeNil())
				Expect(t3.Occurred()).To(BeFalse())
				Expect(createCount3()).To(Equal(0))
			})

			It("Should return an error if the gate overlaps with multiple regions", func() {
				cfg1, count1 := baseConfig(1)
				cfg1.TimeRange = telem.NewRangeSeconds(1, 5)
				cfg1.Subject.Key = "test1"
				g1, t1 := MustSucceed2(c.OpenGate(cfg1))
				Expect(g1).ToNot(BeNil())
				Expect(t1.IsAcquire()).To(BeTrue())
				Expect(count1()).To(Equal(1))

				cfg2, count2 := baseConfig(1)
				cfg2.TimeRange = telem.NewRangeSeconds(5, 10)
				cfg2.Subject.Key = "test2"
				g2, t2 := MustSucceed2(c.OpenGate(cfg2))
				Expect(g2).ToNot(BeNil())
				Expect(t2.IsAcquire()).To(BeTrue())
				Expect(count2()).To(Equal(1))

				cfg3, count3 := baseConfig(1)
				cfg3.TimeRange = telem.NewRangeSeconds(0, 20)
				cfg3.Subject.Key = "test3"
				g3, t3, err := c.OpenGate(cfg3)
				Expect(g3).To(BeNil())
				Expect(t3.Occurred()).To(BeFalse())
				Expect(count3()).To(Equal(0))
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("encountered multiple control regions")))
			})

			It("Should return an error if the controlled resource cannot be created", func() {
				cfg, _ := baseConfig(1)
				count := 0
				cfg.OpenResource = func() (testResource, error) {
					count++
					return testResource{value: 11}, errors.Wrapf(validate.Error, "could not great gate")
				}
				g, t, err := c.OpenGate(cfg)
				Expect(err).To(HaveOccurredAs(validate.Error))
				Expect(t.Occurred()).To(BeFalse())
				Expect(g).To(BeNil())
			})

			It("Should return an error if the caller attempts to register a duplicate control subject", func() {
				cfg, _ := baseConfig(1)
				g, t := MustSucceed2(c.OpenGate(cfg))
				Expect(t.Occurred()).To(BeTrue())
				Expect(g).ToNot(BeNil())
				g, t, err := c.OpenGate(cfg)
				Expect(err).To(HaveOccurredAs(validate.Error))
				Expect(err).To(MatchError(ContainSubstring("control subject [test]<test> is already registered in the region")))
				Expect(t.Occurred()).To(BeFalse())
				Expect(g).To(BeNil())
			})

			It("Should return an error if the user tries to create a gate with a zero time range", func() {
				cfg, _ := baseConfig(1)
				cfg.TimeRange = telem.TimeRange{}
				g, t, err := c.OpenGate(cfg)
				Expect(err).To(MatchError(ContainSubstring("time_range: must be non-zero")))
				Expect(t.Occurred()).To(BeFalse())
				Expect(g).To(BeNil())
			})

			It("Should return an error if the resource subject key is an empty string", func() {
				cfg, _ := baseConfig(1)
				cfg.Subject.Key = ""
				g, t, err := c.OpenGate(cfg)
				Expect(err).To(MatchError(ContainSubstring("subject.key: required")))
				Expect(t.Occurred()).To(BeFalse())
				Expect(g).To(BeNil())
			})

			Describe("ErrOnControlled", func() {
				It("Should take control when there are no other gates in the region", func() {
					By("Getting an absolute gate on an uncontrolled region")
					cfg1, _ := baseConfig(1)
					cfg1.Subject.Key = "g1"
					cfg1.ErrIfControlled = config.True()
					g1, t := MustSucceed2(c.OpenGate(cfg1))
					Expect(t.Occurred()).To(BeTrue())
					_, err := g1.Authorize()
					Expect(err).ToNot(HaveOccurred())

					By("Creating another gate on that region")
					cfg2, _ := baseConfig(1)
					cfg2.Subject.Key = "g2"
					cfg2.Authority = xcontrol.AuthorityAbsolute
					g2, t := MustSucceed2(c.OpenGate(cfg2))
					Expect(t.Occurred()).To(BeFalse())
					_, err = g1.Authorize()
					Expect(err).ToNot(HaveOccurred())
					_, err = g2.Authorize()
					Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
				})

				It("Should fail when there is another gate in the region", func() {
					cfg1, createCount := baseConfig(1)
					cfg1.Subject.Key = "g1"
					cfg1.Authority = 0
					cfg1.ErrIfControlled = config.True()
					g, t := MustSucceed2(c.OpenGate(cfg1))
					Expect(t.Occurred()).To(BeTrue())
					Expect(createCount()).To(Equal(1))
					_, err := g.Authorize()
					Expect(err).ToNot(HaveOccurred())
					cfg2, _ := baseConfig(1)
					cfg2.Subject.Key = "g2"
					cfg2.Authority = 0
					cfg2.ErrIfControlled = config.True()
					g1, t, err := c.OpenGate(cfg2)
					Expect(t.Occurred()).To(BeFalse())
					Expect(g1).To(BeNil())
					Expect(t.Occurred()).To(BeFalse())
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("overlaps with a controlled region")))
				})
			})

			Describe("ErrOnUnauthorizedOpen", func() {
				It("Should return nil if gate is err on open", func() {
					cfg, _ := baseConfig(1)
					cfg.Authority = 0
					cfg.ErrOnUnauthorizedOpen = config.True()
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(t.Occurred()).To(BeTrue())
					Expect(t.IsAcquire()).To(BeTrue())
					Expect(g).ToNot(BeNil())
				})

				It("Should return an error if a higher priority gate is already open", func() {
					cfg1, _ := baseConfig(1)
					cfg1.Subject.Key = "g1"
					cfg1.Authority = xcontrol.AuthorityAbsolute
					g1, t := MustSucceed2(c.OpenGate(cfg1))
					Expect(t.Occurred()).To(BeTrue())
					Expect(g1).ToNot(BeNil())

					cfg2, _ := baseConfig(1)
					cfg2.Subject.Key = "g2"
					cfg2.Authority = xcontrol.AuthorityAbsolute - 1
					cfg2.ErrOnUnauthorizedOpen = config.True()
					g2, t, err := c.OpenGate(cfg2)
					Expect(t.Occurred()).To(BeFalse())
					Expect(g2).To(BeNil())
					Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
				})
			})
		})

		Describe("Authorization, Transfer, and Release", func() {

			Context("One Gate", func() {
				It("Should authorize the gate with absolute control", func() {
					cfg, createCount := baseConfig(10)
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(createCount()).To(Equal(1))
					Expect(t.Occurred()).To(BeTrue())
					Expect(t.IsAcquire()).To(BeTrue())
					Expect(t.IsTransfer()).To(BeFalse())
					v, err := g.Authorize()
					Expect(err).ToNot(HaveOccurred())
					Expect(v.value).To(Equal(10))
				})

				It("Should authorize the gate with 0 control", func() {
					cfg, createCount := baseConfig(10)
					cfg.Authority = 0
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(createCount()).To(Equal(1))
					Expect(t.Occurred()).To(BeTrue())
					Expect(t.IsAcquire()).To(BeTrue())
					Expect(t.IsTransfer()).To(BeFalse())
					v, err := g.Authorize()
					Expect(err).ToNot(HaveOccurred())
					Expect(v.value).To(Equal(10))
				})

				It("Should return false for authorize after the gate has been released", func() {
					cfg, _ := baseConfig(1)
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(t.IsAcquire()).To(BeTrue())
					e, t := g.Release()
					Expect(e.value).To(Equal(1))
					Expect(t.Occurred()).To(BeTrue())
					v, err := g.Authorize()
					Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
					Expect(v.value).To(Equal(0))
				})

				It("Should delete a region when all gates from that region are removed", func() {
					cfg1, createCount1 := baseConfig(11)
					g, t := MustSucceed2(c.OpenGate(cfg1))
					Expect(t.Occurred()).To(BeTrue())
					Expect(createCount1()).To(Equal(1))
					v, t := g.Release()
					Expect(t.Occurred()).To(BeTrue())
					Expect(t.IsRelease()).To(BeTrue())
					Expect(v.value).To(Equal(11))

					cfg2, createCount2 := baseConfig(42)
					g2, t2 := MustSucceed2(c.OpenGate(cfg2))
					Expect(t2.Occurred()).To(BeTrue())
					Expect(t2.IsAcquire()).To(BeTrue())
					Expect(createCount2()).To(Equal(1))
					v = MustSucceed(g2.Authorize())
					Expect(v.value).To(Equal(42))
				})
			})

			Context("Two Gates", func() {
				Describe("Open", func() {
					Context("Open gate 2 with lower authority", func() {
						Specify("The first gate should maintain control", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From).To(BeNil())
							Expect(t.To).ToNot(BeNil())
							Expect(t.To.Subject.Key).To(Equal("g1"))

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 1
							g2, t2 := MustSucceed2(c.OpenGate(cfg2))
							Expect(t2.Occurred()).To(BeFalse())
							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						})
					})

					Context("Open gate 2 with equal authority", func() {
						Specify("The first gate should maintain control", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						})
					})

					Context("Open gate 2 with higher authority", func() {
						Specify("The second gate should take control", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute - 1
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeTrue())
							Expect(t.From.Subject.Key).To(Equal("g1"))
							Expect(t.To.Subject.Key).To(Equal("g2"))

							_, err := g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})
					})
				})

				Describe("SetAuthority", func() {
					Context("Open gate 2 with lower authority, raise authority to higher than gate 1", func() {
						It("Should transfer authority to gate 2", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute - 1
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 2
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							t = g2.SetAuthority(xcontrol.AuthorityAbsolute)
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From.Subject.Key).To(Equal("g1"))
							Expect(t.To.Subject.Key).To(Equal("g2"))

							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						})
					})

					Context("Open gate 2 with lower authority, raise authority to equal to gate 1", func() {
						It("Should not transfer authority to gate 2", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())
							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 1
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())
							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							t = g1.SetAuthority(xcontrol.AuthorityAbsolute - 1)
							Expect(t.Occurred()).To(BeTrue())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})
					})

					Context("Open gate 2 with higher authority, lower authority to equal than gate 1", func() {
						It("Should transfer authority back to gate 1", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute - 1
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeTrue())

							_, err := g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
							t = g2.SetAuthority(xcontrol.AuthorityAbsolute - 1)
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From.Subject.Key).To(Equal("g2"))
							Expect(t.To.Subject.Key).To(Equal("g1"))
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})
					})

					Context("Open gate 2 with higher authority, lower authority to lower than gate 1", func() {
						It("Should transfer control to gate 1", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 1
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							t = g1.SetAuthority(xcontrol.AuthorityAbsolute - 2)
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From.Subject.Key).To(Equal("g1"))
							Expect(t.To.Subject.Key).To(Equal("g2"))

							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})
					})
				})

				Describe("Release", func() {
					Context("Open gate 2 with lower authority, release gate 1", func() {
						It("Should transfer control to gate 2", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 1
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeTrue())
							Expect(v.value).To(Equal(1))
							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						})

					})

					Context("Open gate 2 with equal authority, release gate 1", func() {
						It("Should transfer control to gate 2", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeTrue())
							Expect(v.value).To(Equal(1))
						})

					})

					Context("Open gate 2 with higher authority, release gate 1", func() {
						Specify("Gate 2 should remain in control", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute - 1
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeTrue())

							_, err := g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeFalse())
							Expect(v.value).To(Equal(0))

							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})
					})

					Context("Open gate 1 with higher authority, transfer control to gate 2, release gate 1", func() {
						Specify("Gate 2 should remain in control", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute - 1
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 2
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							t = g2.SetAuthority(xcontrol.AuthorityAbsolute)
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeTrue())

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeFalse())
							Expect(v.value).To(Equal(0))

							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})

					})

					Context("Release both gates", func() {
						It("Should transfer control to the first gate", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute - 1
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(1)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeTrue())

							_, err := g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeFalse())
							Expect(v.value).To(Equal(0))

							v, t = g2.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeFalse())
							Expect(t.IsRelease()).To(BeTrue())
							Expect(v.value).To(Equal(1))
						})
					})
				})
			})

			Context("Three Gates", func() {
				Context("Open gate 1 lowest, gate 2 medium, gate 3 highest", func() {
					It("Should transfer control to each subsequent gate", func() {
						cfg1, _ := baseConfig(1)
						cfg1.Subject.Key = "g1"
						cfg1.Authority = xcontrol.AuthorityAbsolute - 2
						g1, t := MustSucceed2(c.OpenGate(cfg1))
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.IsAcquire()).To(BeTrue())
						Expect(t.IsTransfer()).To(BeFalse())
						Expect(t.From).To(BeNil())
						Expect(t.To).ToNot(BeNil())
						Expect(t.To.Subject.Key).To(Equal("g1"))

						cfg2, _ := baseConfig(1)
						cfg2.Subject.Key = "g2"
						cfg2.Authority = xcontrol.AuthorityAbsolute - 1
						g2, t := MustSucceed2(c.OpenGate(cfg2))
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.IsTransfer()).To(BeTrue())
						Expect(t.From.Subject.Key).To(Equal("g1"))
						Expect(t.To.Subject.Key).To(Equal("g2"))

						_, err := g1.Authorize()
						Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						_, err = g2.Authorize()
						Expect(err).ToNot(HaveOccurred())

						cfg3, _ := baseConfig(1)
						cfg3.Subject.Key = "g3"
						cfg3.Authority = xcontrol.AuthorityAbsolute
						g3, t := MustSucceed2(c.OpenGate(cfg3))
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.IsTransfer()).To(BeTrue())
						Expect(t.From.Subject.Key).To(Equal("g2"))
						Expect(t.To.Subject.Key).To(Equal("g3"))

						_, err = g1.Authorize()
						Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						_, err = g2.Authorize()
						Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						_, err = g3.Authorize()
						Expect(err).ToNot(HaveOccurred())
					})
				})

				Context("Open gate 1 highest, gate 2 medium, gate 3 lowest", func() {
					Specify("Gate 1 should remain in control", func() {
						cfg1, _ := baseConfig(1)
						cfg1.Subject.Key = "g1"
						cfg1.Authority = xcontrol.AuthorityAbsolute
						g1, t := MustSucceed2(c.OpenGate(cfg1))
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.IsAcquire()).To(BeTrue())
						Expect(t.IsTransfer()).To(BeFalse())
						Expect(t.From).To(BeNil())
						Expect(t.To).ToNot(BeNil())
						Expect(t.To.Subject.Key).To(Equal("g1"))

						cfg2, _ := baseConfig(1)
						cfg2.Subject.Key = "g2"
						cfg2.Authority = xcontrol.AuthorityAbsolute - 1
						g2, t := MustSucceed2(c.OpenGate(cfg2))
						Expect(t.Occurred()).To(BeFalse())

						_, err := g1.Authorize()
						Expect(err).ToNot(HaveOccurred())
						_, err = g2.Authorize()
						Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

						cfg3, _ := baseConfig(1)
						cfg3.Subject.Key = "g3"
						cfg3.Authority = xcontrol.AuthorityAbsolute - 2
						g3, t := MustSucceed2(c.OpenGate(cfg3))
						Expect(t.Occurred()).To(BeFalse())

						_, err = g1.Authorize()
						Expect(err).ToNot(HaveOccurred())
						_, err = g2.Authorize()
						Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						_, err = g3.Authorize()
						Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
					})
				})

				Describe("Release", func() {
					Context("Open gate 1 highest, gate 2 then gate 3 equal, release gate 1", func() {
						It("Should transfer control to gate 2", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(2)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 1
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							cfg3, _ := baseConfig(3)
							cfg3.Subject.Key = "g3"
							cfg3.Authority = xcontrol.AuthorityAbsolute - 1
							g3, t := MustSucceed2(c.OpenGate(cfg3))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g3.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeTrue())
							Expect(t.From.Subject.Key).To(Equal("g1"))
							Expect(t.To.Subject.Key).To(Equal("g2"))
							Expect(v.value).To(Equal(1))

							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g3.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
						})
					})

					Context("Open gate 1 highest, gate 2 lowest, gate 3 medium, release g1", func() {
						It("Should transfer control to gate 3", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(2)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 2
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							cfg3, _ := baseConfig(3)
							cfg3.Subject.Key = "g3"
							cfg3.Authority = xcontrol.AuthorityAbsolute - 1
							g3, t := MustSucceed2(c.OpenGate(cfg3))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g3.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeTrue())
							Expect(t.From.Subject.Key).To(Equal("g1"))
							Expect(t.To.Subject.Key).To(Equal("g3"))
							Expect(v.value).To(Equal(1))

							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g3.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})

					})

					Context("Open gate 1 highest, gate 2 medium, gate 3 lowest, release g1", func() {
						It("Should transfer control to gate 2", func() {
							cfg1, _ := baseConfig(1)
							cfg1.Subject.Key = "g1"
							cfg1.Authority = xcontrol.AuthorityAbsolute
							g1, t := MustSucceed2(c.OpenGate(cfg1))
							Expect(t.Occurred()).To(BeTrue())

							cfg2, _ := baseConfig(2)
							cfg2.Subject.Key = "g2"
							cfg2.Authority = xcontrol.AuthorityAbsolute - 1
							g2, t := MustSucceed2(c.OpenGate(cfg2))
							Expect(t.Occurred()).To(BeFalse())

							cfg3, _ := baseConfig(3)
							cfg3.Subject.Key = "g3"
							cfg3.Authority = xcontrol.AuthorityAbsolute - 2
							g3, t := MustSucceed2(c.OpenGate(cfg3))
							Expect(t.Occurred()).To(BeFalse())

							_, err := g1.Authorize()
							Expect(err).ToNot(HaveOccurred())
							_, err = g2.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g3.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.IsTransfer()).To(BeTrue())
							Expect(t.From.Subject.Key).To(Equal("g1"))
							Expect(t.To.Subject.Key).To(Equal("g2"))
							Expect(v.value).To(Equal(1))

							_, err = g1.Authorize()
							Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
							_, err = g2.Authorize()
							Expect(err).ToNot(HaveOccurred())
						})
					})
				})
			})

		})

		Describe("PeekResource", func() {
			It("Should allow the caller to peek at a gate's resource without being controlled", func() {
				cfg1, _ := baseConfig(12)
				cfg1.Subject.Key = "test"
				cfg1.Subject.Name = "test"
				cfg1.Authority = xcontrol.AuthorityAbsolute
				g, t := MustSucceed2(c.OpenGate(cfg1))
				Expect(t.Occurred()).To(BeTrue())
				v := g.PeekResource()
				Expect(v.value).To(Equal(12))
			})
		})

		Describe("LeadingState", func() {
			It("Should return the leading state of the controller", func() {
				cfg1, _ := baseConfig(1)
				cfg1.Subject.Key = "test"
				cfg1.Subject.Name = "test"
				cfg1.Authority = xcontrol.AuthorityAbsolute
				g, t := MustSucceed2(c.OpenGate(cfg1))
				Expect(t.Occurred()).To(BeTrue())
				Expect(t.IsAcquire()).To(BeTrue())
				Expect(g).ToNot(BeNil())
				lead := c.LeadingState()
				Expect(lead).ToNot(BeNil())
				Expect(lead.Subject).To(Equal(xcontrol.Subject{Key: "test", Name: "test"}))
			})
		})
	})

	Context("Shared Control", func() {
		BeforeEach(func() {
			cfg.Concurrency = xcontrol.ConcurrencyShared
		})
		It("Should authorize gate with the highest authority", func() {
			cfg1, _ := baseConfig(1)
			cfg1.Subject.Key = "g1"
			cfg1.Authority = xcontrol.AuthorityAbsolute
			g1, t := MustSucceed2(c.OpenGate(cfg1))
			Expect(t.Occurred()).To(BeTrue())

			cfg2, _ := baseConfig(1)
			cfg2.Subject.Key = "g2"
			cfg2.Authority = xcontrol.AuthorityAbsolute - 1
			g2, t := MustSucceed2(c.OpenGate(cfg2))
			Expect(t.Occurred()).To(BeFalse())

			_, err := g1.Authorize()
			Expect(err).ToNot(HaveOccurred())
			_, err = g2.Authorize()
			Expect(err).To(HaveOccurredAs(xcontrol.ErrUnauthorized))
		})

		It("Should authorize gates with equal authority", func() {
			cfg1, _ := baseConfig(1)
			cfg1.Subject.Key = "g1"
			cfg1.Authority = xcontrol.AuthorityAbsolute
			g1, t := MustSucceed2(c.OpenGate(cfg1))

			Expect(t.Occurred()).To(BeTrue())
			cfg2, _ := baseConfig(1)
			cfg2.Subject.Key = "g2"
			cfg2.Authority = xcontrol.AuthorityAbsolute
			g2, t := MustSucceed2(c.OpenGate(cfg2))
			Expect(t.Occurred()).To(BeFalse())
			_, err := g1.Authorize()
			Expect(err).ToNot(HaveOccurred())

			_, err = g2.Authorize()
			Expect(err).ToNot(HaveOccurred())
		})

	})
})
