// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package claim_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium/internal/claim"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Claim", func() {
	var (
		s       *claim.Set[*int]
		opened  int
		openErr error
	)
	open := func() (*int, error) {
		if openErr != nil {
			return nil, openErr
		}
		opened++
		v := opened
		return &v, nil
	}
	live := func(start telem.TimeStamp) claim.Config[*int] {
		return claim.Config[*int]{
			Open:      open,
			TimeRange: start.Range(telem.TimeStampMax),
		}
	}
	BeforeEach(func() {
		s = &claim.Set[*int]{}
		opened = 0
		openErr = nil
	})

	Describe("Open", func() {
		It("Should open a region with its own resource", func() {
			c := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			Expect(*c.Resource()).To(Equal(1))
		})

		It("Should join the region that the time range reaches", func() {
			a := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			b := MustSucceed(s.Open(live(20 * telem.SecondTS)))
			Expect(b.Resource()).To(BeIdenticalTo(a.Resource()))
			Expect(opened).To(Equal(1))
		})

		It("Should refuse an exclusive claim that reaches a claim", func() {
			MustSucceed(s.Open(live(10 * telem.SecondTS)))
			cfg := live(5 * telem.SecondTS)
			cfg.Exclusive = true
			Expect(s.Open(cfg)).Error().To(MatchError(claim.ErrClaimed))
		})

		It("Should refuse a bounded claim that starts inside a claim", func() {
			MustSucceed(s.Open(live(10 * telem.SecondTS)))
			Expect(s.Open(claim.Config[*int]{
				Open:      open,
				TimeRange: (20 * telem.SecondTS).SpanRange(1),
				Bounded:   true,
			})).Error().To(MatchError(claim.ErrClaimed))
		})

		It("Should let a live claim open over a bounded claim", func() {
			bounded := MustSucceed(s.Open(claim.Config[*int]{
				Open:      open,
				TimeRange: (20 * telem.SecondTS).SpanRange(1),
				Bounded:   true,
			}))
			c := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			Expect(c.Resource()).ToNot(BeIdenticalTo(bounded.Resource()))
		})

		It("Should remove a region whose resource fails to open", func() {
			openErr = errors.New("failed")
			Expect(s.Open(live(10 * telem.SecondTS))).Error().To(MatchError(openErr))
			openErr = nil
			MustSucceed(s.Open(live(10 * telem.SecondTS)))
			Expect(opened).To(Equal(1))
		})
	})

	Describe("ResourceAt", func() {
		It("Should return the resource of a region the range reaches", func() {
			c := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			res, ok := s.ResourceAt((15 * telem.SecondTS).SpanRange(1))
			Expect(ok).To(BeTrue())
			Expect(res).To(BeIdenticalTo(c.Resource()))
		})

		It("Should return false when no region reaches the range", func() {
			MustSucceed(s.Open(live(10 * telem.SecondTS)))
			_, ok := s.ResourceAt((5 * telem.SecondTS).SpanRange(1))
			Expect(ok).To(BeFalse())
		})
	})

	Describe("MoveStart and Reach", func() {
		It("Should free the time before a moved start", func() {
			c := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			Expect(c.MoveStart(20 * telem.SecondTS)).To(Succeed())
			cfg := live(10 * telem.SecondTS)
			cfg.TimeRange = (10 * telem.SecondTS).Range(20 * telem.SecondTS)
			cfg.Exclusive = true
			MustSucceed(s.Open(cfg))
		})

		It("Should refuse a move back that reaches another region", func() {
			MustSucceed(s.Open(claim.Config[*int]{
				Open:      open,
				TimeRange: (5 * telem.SecondTS).Range(8 * telem.SecondTS),
			}))
			c := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			Expect(c.MoveStart(6 * telem.SecondTS)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("because another writer holds it")),
			))
		})

		It("Should grow a bounded claim until it reaches another region", func() {
			c := MustSucceed(s.Open(claim.Config[*int]{
				Open:      open,
				TimeRange: (10 * telem.SecondTS).SpanRange(1),
				Bounded:   true,
			}))
			Expect(c.Reach(15 * telem.SecondTS)).To(Succeed())
			MustSucceed(s.Open(claim.Config[*int]{
				Open:      open,
				TimeRange: (20 * telem.SecondTS).Range(30 * telem.SecondTS),
				Bounded:   true,
			}))
			Expect(c.Reach(25 * telem.SecondTS)).To(MatchError(validate.ErrValidation))
		})
	})

	Describe("Release", func() {
		It("Should report the last claim of a region", func() {
			a := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			b := MustSucceed(s.Open(live(20 * telem.SecondTS)))
			_, last := a.Release()
			Expect(last).To(BeFalse())
			res, last := b.Release()
			Expect(last).To(BeTrue())
			Expect(*res).To(Equal(1))
			_, ok := s.ResourceAt((15 * telem.SecondTS).SpanRange(1))
			Expect(ok).To(BeFalse())
		})

		It("Should return false for a second release", func() {
			a := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			MustSucceed(s.Open(live(20 * telem.SecondTS)))
			a.Release()
			_, last := a.Release()
			Expect(last).To(BeFalse())
		})
	})

	Describe("Use", func() {
		It("Should hold back a holder while another uses the resource", func() {
			a := MustSucceed(s.Open(live(10 * telem.SecondTS)))
			b := MustSucceed(s.Open(live(20 * telem.SecondTS)))
			entered, release, done := make(
				chan struct{},
			), make(
				chan struct{},
			), make(
				chan struct{},
			)
			go func() {
				defer GinkgoRecover()
				Expect(a.Use(func(*int) error {
					close(entered)
					<-release
					return nil
				})).To(Succeed())
			}()
			<-entered
			go func() {
				defer GinkgoRecover()
				Expect(b.Use(func(*int) error {
					close(done)
					return nil
				})).To(Succeed())
			}()
			Consistently(done).ShouldNot(BeClosed())
			close(release)
			Eventually(done).Should(BeClosed())
		})
	})
})
