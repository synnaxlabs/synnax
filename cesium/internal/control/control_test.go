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
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
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

// hookedResource calls onChannelKey from ChannelKey, which a region calls while it
// holds its lock.
type hookedResource struct {
	onChannelKey func()
}

var _ control.Resource = (*hookedResource)(nil)

func (h *hookedResource) ChannelKey() channel.Key {
	if h.onChannelKey != nil {
		h.onChannelKey()
	}
	return channel.Key(0)
}

// parkedInOpenGate returns true if a goroutine in Controller.OpenGate waits on a lock.
func parkedInOpenGate() bool {
	buf := make([]byte, 1<<20)
	buf = buf[:runtime.Stack(buf, true)]
	for g := range strings.SplitSeq(string(buf), "\n\n") {
		state, _, _ := strings.Cut(g, "\n")
		if strings.Contains(state, "[sync.") && strings.Contains(g, ").OpenGate") {
			return true
		}
	}
	return false
}

func createResourceNoErr(value int) (func() (t testResource, err error), func() int) {
	var count int
	return func() (t testResource, err error) {
			count++
			return testResource{value: value}, nil
		}, func() int {
			return count
		}
}

func authorize(g *control.Gate[testResource]) (r testResource, err error) {
	err = g.Authorize(func(v testResource) error {
		r = v
		return nil
	})
	return r, err
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
			It(
				"Should create a new region if the time range is not controller",
				func() {
					cfg, createCount := baseConfig(1)
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(t.Occurred()).To(BeTrue())
					Expect(t.IsAcquire()).To(BeTrue())
					Expect(createCount()).To(Equal(1))
					Expect(g).ToNot(BeNil())
					Expect(g.Authority()).To(Equal(xcontrol.AuthorityAbsolute))
					Expect(g.Subject().Key).To(Equal("test"))
					Expect(g.Subject().Name).To(Equal("test"))
				},
			)

			It(
				"Should not create a new region if the time range is already in the controller",
				func() {
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
				},
			)

			It(
				"Should return an error if the gate overlaps with multiple regions",
				func() {
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
					Expect(
						err,
					).To(MatchError(ContainSubstring("encountered multiple control regions")))
				},
			)

			It(
				"Should return an error if the controlled resource cannot be created",
				func() {
					cfg, _ := baseConfig(1)
					count := 0
					cfg.OpenResource = func() (testResource, error) {
						count++
						return testResource{value: 11}, errors.Wrapf(
							validate.ErrValidation,
							"could not great gate",
						)
					}
					g, t, err := c.OpenGate(cfg)
					Expect(err).To(MatchError(validate.ErrValidation))
					Expect(t.Occurred()).To(BeFalse())
					Expect(g).To(BeNil())
				},
			)

			It(
				"Should return an error if the caller attempts to register a duplicate control subject",
				func() {
					cfg, _ := baseConfig(1)
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(t.Occurred()).To(BeTrue())
					Expect(g).ToNot(BeNil())
					g, t, err := c.OpenGate(cfg)
					Expect(err).To(MatchError(validate.ErrValidation))
					Expect(
						err,
					).To(MatchError(ContainSubstring("control subject [test]<test> is already registered in the region")))
					Expect(t.Occurred()).To(BeFalse())
					Expect(g).To(BeNil())
				},
			)

			It(
				"Should return an error if the user tries to create a gate with a zero time range",
				func() {
					cfg, _ := baseConfig(1)
					cfg.TimeRange = telem.TimeRange{}
					g, t, err := c.OpenGate(cfg)
					Expect(
						err,
					).To(MatchError(ContainSubstring("time_range: must be non-zero")))
					Expect(t.Occurred()).To(BeFalse())
					Expect(g).To(BeNil())
				},
			)

			It(
				"Should return an error if the resource subject key is an empty string",
				func() {
					cfg, _ := baseConfig(1)
					cfg.Subject.Key = ""
					g, t, err := c.OpenGate(cfg)
					Expect(
						err,
					).To(MatchError(ContainSubstring("subject.key: required")))
					Expect(t.Occurred()).To(BeFalse())
					Expect(g).To(BeNil())
				},
			)

			Describe("ErrOnControlled", func() {
				It(
					"Should take control when there are no other gates in the region",
					func() {
						By("Getting an absolute gate on an uncontrolled region")
						cfg1, _ := baseConfig(1)
						cfg1.Subject.Key = "g1"
						cfg1.ErrIfControlled = new(true)
						g1, t := MustSucceed2(c.OpenGate(cfg1))
						Expect(t.Occurred()).To(BeTrue())
						MustSucceed(authorize(g1))

						By("Creating another gate on that region")
						cfg2, _ := baseConfig(1)
						cfg2.Subject.Key = "g2"
						cfg2.Authority = xcontrol.AuthorityAbsolute
						g2, t := MustSucceed2(c.OpenGate(cfg2))
						Expect(t.Occurred()).To(BeFalse())
						MustSucceed(authorize(g1))
						Expect(authorize(g2)).Error().
							To(MatchError(xcontrol.ErrUnauthorized))
					},
				)

				It("Should fail when there is another gate in the region", func() {
					cfg1, createCount := baseConfig(1)
					cfg1.Subject.Key = "g1"
					cfg1.Authority = 0
					cfg1.ErrIfControlled = new(true)
					g, t := MustSucceed2(c.OpenGate(cfg1))
					Expect(t.Occurred()).To(BeTrue())
					Expect(createCount()).To(Equal(1))
					MustSucceed(authorize(g))
					cfg2, _ := baseConfig(1)
					cfg2.Subject.Key = "g2"
					cfg2.Authority = 0
					cfg2.ErrIfControlled = new(true)
					g1, t, err := c.OpenGate(cfg2)
					Expect(t.Occurred()).To(BeFalse())
					Expect(g1).To(BeNil())
					Expect(t.Occurred()).To(BeFalse())
					Expect(
						err,
					).To(MatchError(ContainSubstring("cannot access time range")))
				})
			})

			Describe("ErrOnUnauthorizedOpen", func() {
				It("Should return nil if gate is err on open", func() {
					cfg, _ := baseConfig(1)
					cfg.Authority = 0
					cfg.ErrOnUnauthorizedOpen = new(true)
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(t.Occurred()).To(BeTrue())
					Expect(t.IsAcquire()).To(BeTrue())
					Expect(g).ToNot(BeNil())
				})

				It(
					"Should return an error if a higher priority gate is already open",
					func() {
						cfg1, _ := baseConfig(1)
						cfg1.Subject.Key = "g1"
						cfg1.Authority = xcontrol.AuthorityAbsolute
						g1, t := MustSucceed2(c.OpenGate(cfg1))
						Expect(t.Occurred()).To(BeTrue())
						Expect(g1).ToNot(BeNil())

						cfg2, _ := baseConfig(1)
						cfg2.Subject.Key = "g2"
						cfg2.Authority = xcontrol.AuthorityAbsolute - 1
						cfg2.ErrOnUnauthorizedOpen = new(true)
						g2, t, err := c.OpenGate(cfg2)
						Expect(t.Occurred()).To(BeFalse())
						Expect(g2).To(BeNil())
						Expect(err).To(MatchError(xcontrol.ErrUnauthorized))
					},
				)
			})
		})

		openAt := func(key string, tr telem.TimeRange) *control.Gate[testResource] {
			GinkgoHelper()
			cfg, _ := baseConfig(1)
			cfg.Subject.Key = key
			cfg.TimeRange = tr
			g, _ := MustSucceed2(c.OpenGate(cfg))
			return g
		}
		expectControlled := func(tr telem.TimeRange, controlled bool) {
			GinkgoHelper()
			cfg, _ := baseConfig(2)
			cfg.Subject.Key = "probe"
			cfg.TimeRange = tr
			cfg.ErrIfControlled = new(true)
			if controlled {
				Expect(c.OpenGate(cfg)).Error().
					To(MatchError(xcontrol.ErrUnauthorized))
				return
			}
			g, _ := MustSucceed2(c.OpenGate(cfg))
			g.Release()
		}

		Describe("MoveStart", func() {
			It("Should move the start of the region back", func() {
				g := openAt("test", telem.NewRangeSeconds(10, 20))
				expectControlled(telem.NewRangeSeconds(5, 6), false)
				Expect(g.MoveStart(5 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(5, 6), true)
				expectControlled(telem.NewRangeSeconds(1, 5), false)
			})
			It("Should not move the region into another region", func() {
				other := openAt("other", telem.NewRangeSeconds(1, 3))
				g := openAt("test", telem.NewRangeSeconds(10, 20))
				Expect(g.MoveStart(2 * telem.SecondTS)).To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(fmt.Sprintf(
						"cannot write to time range %v because writer %v holds it",
						telem.NewRangeSeconds(2, 3),
						other.Subject(),
					))),
				))
				expectControlled(telem.NewRangeSeconds(5, 6), false)
			})
			It("Should move the region up to the end of another region", func() {
				openAt("other", telem.NewRangeSeconds(1, 3))
				g := openAt("test", telem.NewRangeSeconds(10, 20))
				Expect(g.MoveStart(3 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(3, 4), true)
			})
			It("Should free the time before the new start", func() {
				g := openAt("live", telem.NewRangeSeconds(1, 20))
				Expect(g.MoveStart(10 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(1, 10), false)
				expectControlled(telem.NewRangeSeconds(10, 11), true)
			})
			It("Should do nothing for a start at the end of the region", func() {
				g := openAt("live", telem.NewRangeSeconds(1, 20))
				Expect(g.MoveStart(20 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(1, 2), true)
			})
			It("Should not move the start of a bounded region forward", func() {
				cfg, _ := baseConfig(1)
				cfg.Subject.Key = "backfill"
				cfg.TimeRange = telem.NewRangeSeconds(1, 20)
				cfg.Bounded = new(true)
				g, _ := MustSucceed2(c.OpenGate(cfg))
				Expect(g.MoveStart(10 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(1, 2), true)
			})
			It("Should let a bounded gate open in the time it frees", func() {
				g := openAt("live", telem.NewRangeSeconds(1, 20))
				Expect(g.MoveStart(10 * telem.SecondTS)).To(Succeed())
				cfg, _ := baseConfig(2)
				cfg.Subject.Key = "backfill"
				cfg.TimeRange = telem.NewRangeSeconds(5, 6)
				cfg.Bounded = new(true)
				backfill, _ := MustSucceed2(c.OpenGate(cfg))
				MustSucceed(authorize(backfill))
				MustSucceed(authorize(g))
			})
			It("Should not race with a concurrent lookup", func() {
				g := openAt("live", telem.NewRangeSeconds(1, 2000))
				var wg sync.WaitGroup
				wg.Go(func() {
					defer GinkgoRecover()
					for i := range telem.TimeStamp(500) {
						Expect(g.MoveStart(telem.SecondTS * (2 + i))).To(Succeed())
					}
				})
				wg.Go(func() {
					for range 500 {
						c.ResourceAt(telem.NewRangeSeconds(1, 2))
					}
				})
				wg.Wait()
			})
		})

		Describe("Reach", func() {
			It("Should move the end of the region forward", func() {
				g := openAt("test", telem.NewRangeSeconds(10, 20))
				expectControlled(telem.NewRangeSeconds(25, 26), false)
				Expect(g.Reach(30 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(25, 26), true)
				expectControlled(telem.NewRangeSeconds(30, 40), false)
			})
			It("Should do nothing when the region already holds the end", func() {
				g := openAt("test", telem.NewRangeSeconds(10, 20))
				Expect(g.Reach(15 * telem.SecondTS)).To(Succeed())
				Expect(g.Reach(20 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(15, 16), true)
				expectControlled(telem.NewRangeSeconds(20, 30), false)
			})
			It("Should not move the region into another region", func() {
				openAt("other", telem.NewRangeSeconds(30, 40))
				g := openAt("test", telem.NewRangeSeconds(10, 20))
				Expect(g.Reach(35 * telem.SecondTS)).To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("cannot write to time range")),
				))
				expectControlled(telem.NewRangeSeconds(25, 26), false)
			})
			It("Should move the region up to the start of another region", func() {
				openAt("other", telem.NewRangeSeconds(30, 40))
				g := openAt("test", telem.NewRangeSeconds(10, 20))
				Expect(g.Reach(30 * telem.SecondTS)).To(Succeed())
				expectControlled(telem.NewRangeSeconds(29, 30), true)
			})
		})

		Describe("Bounded", func() {
			openBounded := func(key string, tr telem.TimeRange) *control.Gate[testResource] {
				GinkgoHelper()
				cfg, _ := baseConfig(1)
				cfg.Subject.Key = key
				cfg.TimeRange = tr
				cfg.Bounded = new(true)
				g, _ := MustSucceed2(c.OpenGate(cfg))
				return g
			}
			It("Should skip a bounded region in the leading state", func() {
				openBounded("backfill", telem.NewRangeSeconds(1, 2))
				openAt("live", telem.NewRangeSeconds(10, 20))
				Expect(c.LeadingState().Subject.Key).To(Equal("live"))
			})
			It("Should have no leading state when every region is bounded", func() {
				openBounded("backfill", telem.NewRangeSeconds(1, 2))
				Expect(c.LeadingState()).To(BeNil())
			})
			It("Should not let an unbounded gate join a bounded region", func() {
				g := openBounded("backfill", telem.NewRangeSeconds(1, 2))
				live := openAt("live", telem.NewRangeSeconds(1, 20))
				Expect(c.LeadingState().Subject.Key).To(Equal("live"))
				MustSucceed(authorize(g))
				MustSucceed(authorize(live))
			})
			It(
				"Should open an unbounded gate over a bounded region and another region",
				func() {
					g := openBounded("backfill", telem.NewRangeSeconds(1, 2))
					live := openAt("live", telem.NewRangeSeconds(10, 20))
					cfg, _ := baseConfig(1)
					cfg.Subject.Key = "operator"
					cfg.TimeRange = telem.NewRangeSeconds(1, 20)
					cfg.Authority = 1
					operator, _ := MustSucceed2(c.OpenGate(cfg))
					MustSucceed(authorize(g))
					MustSucceed(authorize(live))
					Expect(authorize(operator)).Error().
						To(MatchError(xcontrol.ErrUnauthorized))
					live.Release()
					MustSucceed(authorize(operator))
				},
			)
			It("Should fail to open a bounded gate over a bounded region", func() {
				other := openBounded("backfill", telem.NewRangeSeconds(1, 5))
				cfg, _ := baseConfig(1)
				cfg.Subject.Key = "second"
				cfg.TimeRange = telem.NewRangeSeconds(3, 8)
				cfg.Bounded = new(true)
				Expect(c.OpenGate(cfg)).Error().To(And(
					MatchError(xcontrol.ErrUnauthorized),
					MatchError(ContainSubstring(fmt.Sprintf(
						"cannot access time range %v because writer %v holds it",
						telem.NewRangeSeconds(3, 5),
						other.Subject(),
					))),
				))
			})
			It(
				"Should fail to open an exclusive gate where two regions overlap",
				func() {
					openBounded("backfill", telem.NewRangeSeconds(1, 5))
					openAt("live", telem.NewRangeSeconds(1, 20))
					expectControlled(telem.NewRangeSeconds(3, 4), true)
				},
			)
			It("Should fail a reach into an unbounded region", func() {
				g := openBounded("backfill", telem.NewRangeSeconds(1, 2))
				openAt("live", telem.NewRangeSeconds(10, 20))
				Expect(g.Reach(11 * telem.SecondTS)).To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("cannot write to time range")),
				))
			})
			It("Should extend an unbounded region over a bounded region", func() {
				g := openBounded("backfill", telem.NewRangeSeconds(5, 6))
				live := openAt("live", telem.NewRangeSeconds(10, 20))
				Expect(live.MoveStart(1 * telem.SecondTS)).To(Succeed())
				Expect(g.Reach(7 * telem.SecondTS)).To(And(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("cannot write to time range")),
				))
			})
			It("Should prefer a bounded region in ResourceAt", func() {
				cfg, _ := baseConfig(2)
				cfg.Subject.Key = "backfill"
				cfg.TimeRange = telem.NewRangeSeconds(5, 6)
				cfg.Bounded = new(true)
				MustSucceed2(c.OpenGate(cfg))
				cfg, _ = baseConfig(1)
				cfg.Subject.Key = "live"
				cfg.TimeRange = telem.NewRangeSeconds(10, 20)
				live, _ := MustSucceed2(c.OpenGate(cfg))
				Expect(live.MoveStart(1 * telem.SecondTS)).To(Succeed())
				res, ok := c.ResourceAt(telem.NewRangeSeconds(5, 6))
				Expect(ok).To(BeTrue())
				Expect(res.value).To(Equal(2))
				res, ok = c.ResourceAt(telem.NewRangeSeconds(7, 8))
				Expect(ok).To(BeTrue())
				Expect(res.value).To(Equal(1))
			})
			It("Should not race with a reach and a concurrent leading state", func() {
				g := openBounded("backfill", telem.NewRangeSeconds(1, 2))
				var wg sync.WaitGroup
				wg.Go(func() {
					defer GinkgoRecover()
					for i := range telem.TimeStamp(500) {
						Expect(g.Reach(2*telem.SecondTS + i)).To(Succeed())
					}
				})
				wg.Go(func() {
					for range 500 {
						c.LeadingState()
					}
				})
				wg.Wait()
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
					v := MustSucceed(authorize(g))
					Expect(v.value).To(Equal(10))
				})

				It("Should authorize the gate with 0 control", func() {
					cfg, createCount := baseConfig(10)
					cfg.Authority = 0
					g, t := MustSucceed2(c.OpenGate(cfg))
					Expect(createCount()).To(Equal(1))
					Expect(t.Occurred()).To(BeTrue())
					Expect(t.IsAcquire()).To(BeTrue())
					v := MustSucceed(authorize(g))
					Expect(v.value).To(Equal(10))
				})

				It(
					"Should return false for authorize after the gate has been released",
					func() {
						cfg, _ := baseConfig(1)
						g, t := MustSucceed2(c.OpenGate(cfg))
						Expect(t.IsAcquire()).To(BeTrue())
						e, t := g.Release()
						Expect(e.value).To(Equal(1))
						Expect(t.Occurred()).To(BeTrue())
						v, err := authorize(g)
						Expect(err).To(MatchError(xcontrol.ErrUnauthorized))
						Expect(v.value).To(Equal(0))
					},
				)

				It(
					"Should delete a region when all gates from that region are removed",
					func() {
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
						v = MustSucceed(authorize(g2))
						Expect(v.value).To(Equal(42))
					},
				)
			})

			Context("Two Gates", func() {
				It(
					"Should wait for the previous controller to finish using the resource",
					func() {
						cfg1, _ := baseConfig(1)
						cfg1.Subject.Key = "g1"
						cfg1.Authority = xcontrol.AuthorityAbsolute - 1
						g1, _ := MustSucceed2(c.OpenGate(cfg1))
						inG1, releaseG1 := make(chan struct{}), make(chan struct{})
						release := sync.OnceFunc(func() { close(releaseG1) })
						DeferCleanup(release)
						g1Done := make(chan error, 1)
						go func() {
							defer GinkgoRecover()
							g1Done <- g1.Authorize(func(testResource) error {
								close(inG1)
								Eventually(releaseG1).Should(BeClosed())
								return nil
							})
						}()
						Eventually(inG1).Should(BeClosed())

						cfg2, _ := baseConfig(1)
						cfg2.Subject.Key = "g2"
						g2, t := MustSucceed2(c.OpenGate(cfg2))
						Expect(t.From).ToNot(BeNil())
						Expect(t.To).ToNot(BeNil())
						g2Done := make(chan error, 1)
						go func() {
							g2Done <- g2.Authorize(func(testResource) error {
								return nil
							})
						}()
						Consistently(g2Done, 50*time.Millisecond).ShouldNot(Receive())

						release()
						Eventually(g1Done).Should(Receive(BeNil()))
						Eventually(g2Done).Should(Receive(BeNil()))
					},
				)

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
							MustSucceed(authorize(g1))
							Expect(
								authorize(g2),
							).Error().
								To(MatchError(xcontrol.ErrUnauthorized))
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

							MustSucceed(authorize(g1))
							Expect(
								authorize(g2),
							).Error().
								To(MatchError(xcontrol.ErrUnauthorized))
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
							Expect(t.From).ToNot(BeNil())
							Expect(t.To).ToNot(BeNil())
							Expect(t.From.Subject.Key).To(Equal("g1"))
							Expect(t.To.Subject.Key).To(Equal("g2"))

							Expect(authorize(g1)).Error().
								To(MatchError(xcontrol.ErrUnauthorized))
							MustSucceed(authorize(g2))
						})
					})
				})

				Describe("SetAuthority", func() {
					Context(
						"Open gate 2 with lower authority, raise authority to higher than gate 1",
						func() {
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

								MustSucceed(authorize(g1))
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))

								t = g2.SetAuthority(xcontrol.AuthorityAbsolute)
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.From.Subject.Key).To(Equal("g1"))
								Expect(t.To.Subject.Key).To(Equal("g2"))

								MustSucceed(authorize(g2))
								Expect(
									authorize(g1),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
							})
						},
					)

					Context(
						"Open gate 2 with lower authority, raise authority to equal to gate 1",
						func() {
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
								MustSucceed(authorize(g1))
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								t = g1.SetAuthority(xcontrol.AuthorityAbsolute - 1)
								Expect(t.Occurred()).To(BeTrue())
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g1))
							})
						},
					)

					Context(
						"Open gate 2 with higher authority, lower authority to equal than gate 1",
						func() {
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

								Expect(authorize(g1)).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g2))
								t = g2.SetAuthority(xcontrol.AuthorityAbsolute - 1)
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.From.Subject.Key).To(Equal("g2"))
								Expect(t.To.Subject.Key).To(Equal("g1"))
								Expect(authorize(g2)).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g1))
							})
						},
					)

					Context(
						"Open gate 2 with higher authority, lower authority to lower than gate 1",
						func() {
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

								MustSucceed(authorize(g1))
								Expect(authorize(g2)).Error().
									To(MatchError(xcontrol.ErrUnauthorized))

								t = g1.SetAuthority(xcontrol.AuthorityAbsolute - 2)
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.From.Subject.Key).To(Equal("g1"))
								Expect(t.To.Subject.Key).To(Equal("g2"))

								Expect(authorize(g1)).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g2))
							})
						},
					)
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

							MustSucceed(authorize(g1))
							Expect(
								authorize(g2),
							).Error().
								To(MatchError(xcontrol.ErrUnauthorized))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From).ToNot(BeNil())
							Expect(t.To).ToNot(BeNil())
							Expect(v.value).To(Equal(1))
							Expect(authorize(g1)).Error().
								To(MatchError(xcontrol.ErrUnauthorized))
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

							MustSucceed(authorize(g1))
							Expect(
								authorize(g2),
							).Error().
								To(MatchError(xcontrol.ErrUnauthorized))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeTrue())
							Expect(t.From).ToNot(BeNil())
							Expect(t.To).ToNot(BeNil())
							Expect(v.value).To(Equal(1))
						})
					})

					Context(
						"Open gate 2 with higher authority, release gate 1",
						func() {
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

								Expect(
									authorize(g1),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g2))

								v, t := g1.Release()
								Expect(t.Occurred()).To(BeFalse())
								Expect(v.value).To(Equal(0))

								Expect(
									authorize(g1),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g2))
							})
						},
					)

					Context(
						"Open gate 1 with higher authority, transfer control to gate 2, release gate 1",
						func() {
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

								MustSucceed(authorize(g1))
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))

								t = g2.SetAuthority(xcontrol.AuthorityAbsolute)
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.From).ToNot(BeNil())
								Expect(t.To).ToNot(BeNil())

								v, t := g1.Release()
								Expect(t.Occurred()).To(BeFalse())
								Expect(v.value).To(Equal(0))

								Expect(
									authorize(g1),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g2))
							})
						},
					)

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

							Expect(
								authorize(g1),
							).Error().
								To(MatchError(xcontrol.ErrUnauthorized))
							MustSucceed(authorize(g2))

							v, t := g1.Release()
							Expect(t.Occurred()).To(BeFalse())
							Expect(v.value).To(Equal(0))

							v, t = g2.Release()
							Expect(t.Occurred()).To(BeTrue())
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
						Expect(t.From).To(BeNil())
						Expect(t.To).ToNot(BeNil())
						Expect(t.To.Subject.Key).To(Equal("g1"))

						cfg2, _ := baseConfig(1)
						cfg2.Subject.Key = "g2"
						cfg2.Authority = xcontrol.AuthorityAbsolute - 1
						g2, t := MustSucceed2(c.OpenGate(cfg2))
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From).ToNot(BeNil())
						Expect(t.To).ToNot(BeNil())
						Expect(t.From.Subject.Key).To(Equal("g1"))
						Expect(t.To.Subject.Key).To(Equal("g2"))

						Expect(
							authorize(g1),
						).Error().
							To(MatchError(xcontrol.ErrUnauthorized))
						MustSucceed(authorize(g2))

						cfg3, _ := baseConfig(1)
						cfg3.Subject.Key = "g3"
						cfg3.Authority = xcontrol.AuthorityAbsolute
						g3, t := MustSucceed2(c.OpenGate(cfg3))
						Expect(t.Occurred()).To(BeTrue())
						Expect(t.From).ToNot(BeNil())
						Expect(t.To).ToNot(BeNil())
						Expect(t.From.Subject.Key).To(Equal("g2"))
						Expect(t.To.Subject.Key).To(Equal("g3"))

						Expect(
							authorize(g1),
						).Error().
							To(MatchError(xcontrol.ErrUnauthorized))
						Expect(
							authorize(g2),
						).Error().
							To(MatchError(xcontrol.ErrUnauthorized))
						MustSucceed(authorize(g3))
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
						Expect(t.From).To(BeNil())
						Expect(t.To).ToNot(BeNil())
						Expect(t.To.Subject.Key).To(Equal("g1"))

						cfg2, _ := baseConfig(1)
						cfg2.Subject.Key = "g2"
						cfg2.Authority = xcontrol.AuthorityAbsolute - 1
						g2, t := MustSucceed2(c.OpenGate(cfg2))
						Expect(t.Occurred()).To(BeFalse())

						MustSucceed(authorize(g1))
						Expect(
							authorize(g2),
						).Error().
							To(MatchError(xcontrol.ErrUnauthorized))

						cfg3, _ := baseConfig(1)
						cfg3.Subject.Key = "g3"
						cfg3.Authority = xcontrol.AuthorityAbsolute - 2
						g3, t := MustSucceed2(c.OpenGate(cfg3))
						Expect(t.Occurred()).To(BeFalse())

						MustSucceed(authorize(g1))
						Expect(
							authorize(g2),
						).Error().
							To(MatchError(xcontrol.ErrUnauthorized))
						Expect(
							authorize(g3),
						).Error().
							To(MatchError(xcontrol.ErrUnauthorized))
					})
				})

				Describe("Release", func() {
					Context(
						"Open gate 1 highest, gate 2 then gate 3 equal, release gate 1",
						func() {
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

								MustSucceed(authorize(g1))
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								Expect(
									authorize(g3),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))

								v, t := g1.Release()
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.From).ToNot(BeNil())
								Expect(t.To).ToNot(BeNil())
								Expect(t.From.Subject.Key).To(Equal("g1"))
								Expect(t.To.Subject.Key).To(Equal("g2"))
								Expect(v.value).To(Equal(1))

								Expect(
									authorize(g1),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g2))
								Expect(
									authorize(g3),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
							})
						},
					)

					Context(
						"Open gate 1 highest, gate 2 lowest, gate 3 medium, release g1",
						func() {
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

								MustSucceed(authorize(g1))
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								Expect(
									authorize(g3),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))

								v, t := g1.Release()
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.From).ToNot(BeNil())
								Expect(t.To).ToNot(BeNil())
								Expect(t.From.Subject.Key).To(Equal("g1"))
								Expect(t.To.Subject.Key).To(Equal("g3"))
								Expect(v.value).To(Equal(1))

								Expect(
									authorize(g1),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g3))
							})
						},
					)

					Context(
						"Open gate 1 highest, gate 2 medium, gate 3 lowest, release g1",
						func() {
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

								MustSucceed(authorize(g1))
								Expect(
									authorize(g2),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								Expect(
									authorize(g3),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))

								v, t := g1.Release()
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.From).ToNot(BeNil())
								Expect(t.To).ToNot(BeNil())
								Expect(t.From.Subject.Key).To(Equal("g1"))
								Expect(t.To.Subject.Key).To(Equal("g2"))
								Expect(v.value).To(Equal(1))

								Expect(
									authorize(g1),
								).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								MustSucceed(authorize(g2))
							})
						},
					)
				})
			})
		})

		Describe("Authority Transfer With SetAuthority", func() {
			Context(
				"Arc at authority 200, two schematics at authority 1, schematics take/release absolute control sequentially",
				func() {
					It(
						"Should always transfer control back to the highest-authority gate (Arc) when releasing absolute",
						func() {
							for i := range 50 {
								By(fmt.Sprintf("Iteration %d", i))
								innerC := MustSucceed(
									control.New[testResource](control.Config{
										Concurrency: xcontrol.ConcurrencyExclusive,
									}),
								)

								arcCfg, _ := baseConfig(1)
								arcCfg.Subject.Key = "arc"
								arcCfg.Subject.Name = "arc"
								arcCfg.Authority = 200
								arc, t := MustSucceed2(innerC.OpenGate(arcCfg))
								Expect(t.IsAcquire()).To(BeTrue())

								sch1Cfg, _ := baseConfig(1)
								sch1Cfg.Subject.Key = "schematic-1"
								sch1Cfg.Subject.Name = "schematic-1"
								sch1Cfg.Authority = 1
								sch1, _ := MustSucceed2(innerC.OpenGate(sch1Cfg))

								sch2Cfg, _ := baseConfig(1)
								sch2Cfg.Subject.Key = "schematic-2"
								sch2Cfg.Subject.Name = "schematic-2"
								sch2Cfg.Authority = 1
								sch2, _ := MustSucceed2(innerC.OpenGate(sch2Cfg))

								// Arc should be in control (highest authority)
								MustSucceed(authorize(arc))

								// Schematic 1 takes then releases absolute
								sch1.SetAuthority(xcontrol.AuthorityAbsolute)
								t = sch1.SetAuthority(1)
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.To.Subject.Key).To(Equal("arc"))

								// Schematic 2 takes then releases absolute -
								// this is the critical step where the bug manifests
								sch2.SetAuthority(xcontrol.AuthorityAbsolute)
								t = sch2.SetAuthority(1)
								Expect(t.Occurred()).To(BeTrue())
								Expect(t.To.Subject.Key).To(Equal("arc"),
									fmt.Sprintf("iteration %d: control went to %s instead of arc", i, t.To.Subject.Key),
								)

								MustSucceed(authorize(arc))
								Expect(authorize(sch1)).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
								Expect(authorize(sch2)).Error().
									To(MatchError(xcontrol.ErrUnauthorized))
							}
						},
					)
				},
			)
		})

		Describe("PeekResource", func() {
			It(
				"Should allow the caller to peek at a gate's resource without being controlled",
				func() {
					cfg1, _ := baseConfig(12)
					cfg1.Subject.Key = "test"
					cfg1.Subject.Name = "test"
					cfg1.Authority = xcontrol.AuthorityAbsolute
					g, t := MustSucceed2(c.OpenGate(cfg1))
					Expect(t.Occurred()).To(BeTrue())
					v := g.PeekResource()
					Expect(v.value).To(Equal(12))
				},
			)
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
				Expect(
					lead.Subject,
				).To(Equal(xcontrol.Subject{Key: "test", Name: "test"}))
			})

			It("Should not race with concurrent gate open and release", func() {
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					defer GinkgoRecover()
					for i := range 50 {
						cfg, _ := baseConfig(i)
						cfg.Subject.Key = fmt.Sprintf("subject-%d", i)
						g, _ := MustSucceed2(c.OpenGate(cfg))
						g.Release()
					}
				}()
				go func() {
					defer wg.Done()
					for range 500 {
						c.LeadingState()
					}
				}()
				wg.Wait()
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

			MustSucceed(authorize(g1))
			Expect(authorize(g2)).Error().To(MatchError(xcontrol.ErrUnauthorized))
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
			MustSucceed(authorize(g1))

			MustSucceed(authorize(g2))
		})
	})

	Context("Concurrent Open and Close", func() {
		BeforeEach(func() {
			cfg.Concurrency = xcontrol.ConcurrencyExclusive
		})

		It("Should not deadlock when opening and closing gates concurrently", func() {
			const (
				numGoroutines   = 10
				opsPerGoroutine = 200
			)
			var wg sync.WaitGroup
			wg.Add(numGoroutines)
			for i := range numGoroutines {
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					for j := range opsPerGoroutine {
						gateCfg := control.GateConfig[testResource]{
							Subject: xcontrol.Subject{
								Key:  fmt.Sprintf("g-%d-%d", i, j),
								Name: fmt.Sprintf("g-%d-%d", i, j),
							},
							TimeRange:    telem.TimeRangeMax,
							Authority:    xcontrol.AuthorityAbsolute,
							OpenResource: func() (testResource, error) { return testResource{value: i*1000 + j}, nil },
						}
						g, _, err := c.OpenGate(gateCfg)
						if err != nil {
							continue
						}
						g.Release()
					}
				}()
			}
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				Fail(
					"Deadlock detected: concurrent open/close did not complete within 10 seconds",
				)
			}
		})
	})

	Describe("Releasing the last gate", func() {
		It("Should not let a new gate join the region", func() {
			var (
				c      = MustSucceed(control.New[*hookedResource](cfg))
				first  = &hookedResource{}
				second = &hookedResource{}
				joined = make(chan *control.Gate[*hookedResource], 1)
				once   sync.Once
			)
			gateCfg := func(
				key string,
				r *hookedResource,
			) control.GateConfig[*hookedResource] {
				return control.GateConfig[*hookedResource]{
					Subject:      xcontrol.Subject{Key: key},
					TimeRange:    telem.TimeRangeMax,
					Authority:    xcontrol.AuthorityAbsolute,
					OpenResource: func() (*hookedResource, error) { return r, nil },
				}
			}
			g, _ := MustSucceed2(c.OpenGate(gateCfg("first", first)))
			// Runs under the region lock, so the new gate is already waiting when the
			// release lets go of that lock.
			first.onChannelKey = func() {
				once.Do(func() {
					go func() {
						defer GinkgoRecover()
						j, _ := MustSucceed2(c.OpenGate(gateCfg("second", second)))
						joined <- j
					}()
					Eventually(parkedInOpenGate).Should(BeTrue())
				})
			}
			res, t := g.Release()
			Expect(t.IsRelease()).To(BeTrue())
			Expect(res).To(BeIdenticalTo(first))
			var j *control.Gate[*hookedResource]
			Eventually(joined).Should(Receive(&j))
			Expect(j.PeekResource()).To(BeIdenticalTo(second))
			res, t = j.Release()
			Expect(t.IsRelease()).To(BeTrue())
			Expect(res).To(BeIdenticalTo(second))
		})
	})

	Describe("Opening a resource", func() {
		type opened struct {
			err error
			g   *control.Gate[testResource]
			t   control.Transfer
		}
		var (
			regionA = (0 * telem.SecondTS).Range(10 * telem.SecondTS)
			regionB = (20 * telem.SecondTS).Range(30 * telem.SecondTS)
		)
		gateCfg := func(
			key string,
			tr telem.TimeRange,
			value int,
		) control.GateConfig[testResource] {
			cfg, _ := baseConfig(value)
			cfg.Subject.Key = key
			cfg.TimeRange = tr
			return cfg
		}
		openAsync := func(cfg control.GateConfig[testResource]) <-chan opened {
			result := make(chan opened, 1)
			go func() {
				defer GinkgoRecover()
				g, t, err := c.OpenGate(cfg)
				result <- opened{g: g, t: t, err: err}
			}()
			return result
		}
		receive := func(result <-chan opened) (
			*control.Gate[testResource],
			control.Transfer,
			error,
		) {
			GinkgoHelper()
			var o opened
			Eventually(result).Should(Receive(&o))
			return o.g, o.t, o.err
		}
		// openBlocked opens a gate on tr and returns once its OpenResource is running.
		// OpenResource returns the error sent on finish.
		openBlocked := func(
			tr telem.TimeRange,
			bounded bool,
		) (result <-chan opened, finish chan<- error) {
			GinkgoHelper()
			var (
				cfg     = gateCfg("blocked", tr, 1)
				entered = make(chan struct{})
				errs    = make(chan error, 1)
			)
			cfg.Bounded = new(bounded)
			cfg.OpenResource = func() (testResource, error) {
				close(entered)
				var err error
				Eventually(errs, 10*time.Second).Should(Receive(&err))
				if err != nil {
					return testResource{}, err
				}
				return testResource{value: 1}, nil
			}
			result = openAsync(cfg)
			Eventually(entered).Should(BeClosed())
			DeferCleanup(func() {
				select {
				case errs <- nil:
				default:
				}
			})
			return result, errs
		}
		expectReturns := func(f func()) {
			GinkgoHelper()
			done := make(chan struct{})
			go func() {
				defer GinkgoRecover()
				defer close(done)
				f()
			}()
			Eventually(done).Should(BeClosed())
		}

		It("Should not block the release of a gate on another region", func() {
			held, _ := MustSucceed2(c.OpenGate(gateCfg("held", regionA, 2)))
			openBlocked(regionB, false)
			expectReturns(func() { held.Release() })
		})

		It("Should hide the region from LeadingState until it opens", func() {
			opening, finish := openBlocked(regionB, false)
			expectReturns(func() { Expect(c.LeadingState()).To(BeNil()) })
			finish <- nil
			_, t := MustSucceed2(receive(opening))
			Expect(t.IsAcquire()).To(BeTrue())
			Expect(c.LeadingState()).ToNot(BeNil())
		})

		It("Should make ResourceAt wait for the region to open", func() {
			opening, finish := openBlocked(regionB, false)
			found := make(chan testResource, 1)
			go func() {
				defer GinkgoRecover()
				found <- MustBeOk(c.ResourceAt(regionB))
			}()
			Consistently(found, 50*time.Millisecond).ShouldNot(Receive())
			finish <- nil
			Eventually(found).Should(Receive(Equal(testResource{value: 1})))
			MustSucceed2(receive(opening))
		})

		It("Should not make ResourceAt wait with a ready bounded region", func() {
			cfg := gateCfg("bounded", regionB, 2)
			cfg.Bounded = new(true)
			MustSucceed2(c.OpenGate(cfg))
			openBlocked((10 * telem.SecondTS).Range(40*telem.SecondTS), false)
			expectReturns(func() {
				Expect(
					MustBeOk(c.ResourceAt(regionB)),
				).To(Equal(testResource{value: 2}))
			})
		})

		It("Should make ResourceAt wait for an opening bounded region", func() {
			opening, finish := openBlocked(regionB, true)
			unbounded := (10 * telem.SecondTS).Range(40 * telem.SecondTS)
			MustSucceed2(c.OpenGate(gateCfg("unbounded", unbounded, 2)))
			found := make(chan testResource, 1)
			go func() {
				defer GinkgoRecover()
				found <- MustBeOk(c.ResourceAt(regionB))
			}()
			Consistently(found, 50*time.Millisecond).ShouldNot(Receive())
			finish <- nil
			Eventually(found).Should(Receive(Equal(testResource{value: 1})))
			MustSucceed2(receive(opening))
		})

		It("Should report no resource at a region that fails to open", func() {
			opening, finish := openBlocked(regionB, false)
			found := make(chan bool, 1)
			go func() {
				defer GinkgoRecover()
				_, ok := c.ResourceAt(regionB)
				found <- ok
			}()
			Consistently(found, 50*time.Millisecond).ShouldNot(Receive())
			openErr := errors.New("open failed")
			finish <- openErr
			Eventually(found).Should(Receive(BeFalse()))
			Expect(receive(opening)).Error().To(MatchError(openErr))
		})

		It("Should make a gate wait for the region opening on its time range", func() {
			opening, finish := openBlocked(regionB, false)
			cfg := gateCfg("waiting", regionB, 2)
			cfg.Authority = xcontrol.AuthorityAbsolute - 1
			waiting := openAsync(cfg)
			Consistently(waiting, 50*time.Millisecond).ShouldNot(Receive())
			finish <- nil
			first, _ := MustSucceed2(receive(opening))
			second, t := MustSucceed2(receive(waiting))
			Expect(t.Occurred()).To(BeFalse())
			Expect(second.PeekResource()).To(Equal(first.PeekResource()))
		})

		It("Should fail a gate that errors if the region is controlled", func() {
			openBlocked(regionB, false)
			cfg := gateCfg("exclusive", regionB, 2)
			cfg.ErrIfControlled = new(true)
			expectReturns(func() {
				Expect(c.OpenGate(cfg)).Error().To(SatisfyAll(
					MatchError(xcontrol.ErrUnauthorized),
					MatchError(ContainSubstring("a writer is opening")),
				))
			})
		})

		It("Should open one resource for gates racing for a time range", func() {
			const gates = 32
			var (
				count atomic.Int32
				wg    sync.WaitGroup
			)
			open := make([]*control.Gate[testResource], gates)
			for i := range gates {
				wg.Go(func() {
					defer GinkgoRecover()
					cfg := gateCfg(fmt.Sprintf("gate-%d", i), regionA, i)
					cfg.OpenResource = func() (testResource, error) {
						return testResource{value: int(count.Add(1))}, nil
					}
					open[i], _ = MustSucceed2(c.OpenGate(cfg))
				})
			}
			wg.Wait()
			Expect(count.Load()).To(Equal(int32(1)))
			for _, g := range open {
				Expect(g.PeekResource()).To(Equal(testResource{value: 1}))
			}
		})

		It("Should let a waiting gate open the resource after a failed open", func() {
			opening, finish := openBlocked(regionB, false)
			waiting := openAsync(gateCfg("waiting", regionB, 2))
			Consistently(waiting, 50*time.Millisecond).ShouldNot(Receive())
			openErr := errors.New("open failed")
			finish <- openErr
			Expect(receive(opening)).Error().To(MatchError(openErr))
			g, t := MustSucceed2(receive(waiting))
			Expect(t.IsAcquire()).To(BeTrue())
			Expect(g.PeekResource()).To(Equal(testResource{value: 2}))
		})

		It("Should let a waiting gate open the resource after a panic", func() {
			var (
				cfg       = gateCfg("panics", regionB, 1)
				entered   = make(chan struct{})
				proceed   = make(chan struct{})
				recovered = make(chan any, 1)
			)
			cfg.OpenResource = func() (testResource, error) {
				close(entered)
				Eventually(proceed).Should(BeClosed())
				panic("open failed")
			}
			go func() {
				defer func() { recovered <- recover() }()
				MustSucceed2(c.OpenGate(cfg))
			}()
			Eventually(entered).Should(BeClosed())
			waiting := openAsync(gateCfg("waiting", regionB, 2))
			Consistently(waiting, 50*time.Millisecond).ShouldNot(Receive())
			close(proceed)
			Eventually(recovered).Should(Receive(Equal("open failed")))
			g, t := MustSucceed2(receive(waiting))
			Expect(t.IsAcquire()).To(BeTrue())
			Expect(g.PeekResource()).To(Equal(testResource{value: 2}))
		})

		It("Should fail a reach into the region", func() {
			held, _ := MustSucceed2(c.OpenGate(gateCfg("held", regionA, 2)))
			openBlocked(regionB, false)
			expectReturns(func() {
				Expect(held.Reach(25 * telem.SecondTS)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("a writer that is opening")),
				))
			})
		})
	})

	Describe("ResourceAt", func() {
		at := func(ts telem.TimeStamp) telem.TimeRange { return ts.Range(ts + 1) }
		openAt := func(value int, tr telem.TimeRange) *control.Gate[testResource] {
			GinkgoHelper()
			cfg, _ := baseConfig(value)
			cfg.TimeRange = tr
			cfg.Subject.Key = fmt.Sprintf("gate-%d", value)
			g, _ := MustSucceed2(c.OpenGate(cfg))
			return g
		}

		It(
			"Should return the resource held by the region covering the time range",
			func() {
				// Opened out of chronological order so that a lookup keyed off
				// insertion order or sort position resolves a different region than the
				// correct one.
				openAt(2, (20 * telem.SecondTS).Range(30*telem.SecondTS))
				openAt(1, (0 * telem.SecondTS).Range(10*telem.SecondTS))
				openAt(3, (40 * telem.SecondTS).Range(50*telem.SecondTS))
				Expect(
					MustBeOk(c.ResourceAt(at(5 * telem.SecondTS))),
				).To(Equal(testResource{value: 1}))
				Expect(
					MustBeOk(c.ResourceAt(at(25 * telem.SecondTS))),
				).To(Equal(testResource{value: 2}))
				Expect(
					MustBeOk(c.ResourceAt(at(45 * telem.SecondTS))),
				).To(Equal(testResource{value: 3}))
			},
		)

		It(
			"Should resolve a region that the time range only partially overlaps",
			func() {
				openAt(1, (10 * telem.SecondTS).Range(20*telem.SecondTS))
				Expect(
					MustBeOk(
						c.ResourceAt((5 * telem.SecondTS).Range(15 * telem.SecondTS)),
					),
				).
					To(Equal(testResource{value: 1}))
				Expect(
					MustBeOk(
						c.ResourceAt((15 * telem.SecondTS).Range(25 * telem.SecondTS)),
					),
				).
					To(Equal(testResource{value: 1}))
			},
		)

		It(
			"Should report no resource for a time range falling between regions",
			func() {
				openAt(1, (0 * telem.SecondTS).Range(10*telem.SecondTS))
				openAt(2, (20 * telem.SecondTS).Range(30*telem.SecondTS))
				Expect(c.ResourceAt(at(15 * telem.SecondTS))).To(Equal(testResource{}))
			},
		)

		It("Should report no resource for a time range past every region", func() {
			openAt(1, (0 * telem.SecondTS).Range(10*telem.SecondTS))
			Expect(c.ResourceAt(at(50 * telem.SecondTS))).To(Equal(testResource{}))
		})

		It("Should report no resource when the controller holds no regions", func() {
			Expect(c.ResourceAt(at(5 * telem.SecondTS))).To(Equal(testResource{}))
		})

		It("Should report no resource once the region's last gate is released", func() {
			g := openAt(1, (0 * telem.SecondTS).Range(10*telem.SecondTS))
			Expect(
				MustBeOk(c.ResourceAt(at(5 * telem.SecondTS))),
			).To(Equal(testResource{value: 1}))
			g.Release()
			Expect(c.ResourceAt(at(5 * telem.SecondTS))).To(Equal(testResource{}))
		})

		It(
			"Should return the shared resource to every gate bound to the same region",
			func() {
				cfg, createCount := baseConfig(1)
				cfg.TimeRange = (0 * telem.SecondTS).Range(20 * telem.SecondTS)
				MustSucceed2(c.OpenGate(cfg))
				second, _ := baseConfig(2)
				second.TimeRange = (10 * telem.SecondTS).Range(30 * telem.SecondTS)
				second.Subject.Key = "second"
				MustSucceed2(c.OpenGate(second))
				// The second gate joined the first gate's region, so its OpenResource
				// is never called and both stamps resolve to the resource opened for
				// the first.
				Expect(createCount()).To(Equal(1))
				Expect(
					MustBeOk(c.ResourceAt(at(5 * telem.SecondTS))),
				).To(Equal(testResource{value: 1}))
				Expect(
					MustBeOk(c.ResourceAt(at(25 * telem.SecondTS))),
				).To(Equal(testResource{value: 1}))
			},
		)
	})
})
