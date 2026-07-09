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

	"github.com/synnaxlabs/synnax/pkg/service/arc/control"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	arcSubject = xcontrol.Subject{Key: "arc-1", Name: "arc"}
	opSubject  = xcontrol.Subject{Key: "op-1", Name: "operator"}
)

func acquireUpdate(
	subject xcontrol.Subject,
	key channel.Key,
	authority xcontrol.Authority,
) xcontrol.Update[channel.Key] {
	return xcontrol.Update[channel.Key]{Transfers: []xcontrol.Transfer[channel.Key]{{
		To: &xcontrol.State[channel.Key]{Subject: subject, Resource: key, Authority: authority},
	}}}
}

func releaseUpdate(
	subject xcontrol.Subject,
	key channel.Key,
	authority xcontrol.Authority,
) xcontrol.Update[channel.Key] {
	return xcontrol.Update[channel.Key]{Transfers: []xcontrol.Transfer[channel.Key]{{
		From: &xcontrol.State[channel.Key]{Subject: subject, Resource: key, Authority: authority},
	}}}
}

func handoffUpdate(
	from, to xcontrol.Subject,
	key channel.Key,
	fromAuthority, toAuthority xcontrol.Authority,
) xcontrol.Update[channel.Key] {
	return xcontrol.Update[channel.Key]{Transfers: []xcontrol.Transfer[channel.Key]{{
		From: &xcontrol.State[channel.Key]{Subject: from, Resource: key, Authority: fromAuthority},
		To:   &xcontrol.State[channel.Key]{Subject: to, Resource: key, Authority: toAuthority},
	}}}
}

var _ = Describe("States", func() {
	Describe("Apply", func() {
		It("Should treat an uncontrolled channel as authorized", func() {
			s := control.New()
			Expect(s.IsAuthorized(1, arcSubject)).To(BeTrue())
		})
		It("Should record the holder on acquire", func() {
			s := control.New()
			s.Apply(acquireUpdate(arcSubject, 1, 200))
			Expect(s.IsAuthorized(1, arcSubject)).To(BeTrue())
			Expect(s.IsAuthorized(1, opSubject)).To(BeFalse())
		})
		It("Should free the channel on release", func() {
			s := control.New()
			s.Apply(acquireUpdate(arcSubject, 1, 200))
			s.Apply(releaseUpdate(arcSubject, 1, 200))
			Expect(s.IsAuthorized(1, opSubject)).To(BeTrue())
		})
		It("Should move control on handoff", func() {
			s := control.New()
			s.Apply(acquireUpdate(arcSubject, 1, 200))
			s.Apply(handoffUpdate(arcSubject, opSubject, 1, 200, 250))
			Expect(s.IsAuthorized(1, arcSubject)).To(BeFalse())
			Expect(s.IsAuthorized(1, opSubject)).To(BeTrue())
		})
	})

	DescribeTable("ApplyIncrease",
		func(arcAuthority, opAuthority xcontrol.Authority, holder xcontrol.Subject) {
			s := control.New()
			s.ApplyIncrease(arcSubject, 1, arcAuthority)
			s.ApplyIncrease(opSubject, 1, opAuthority)
			Expect(s.IsAuthorized(1, holder)).To(BeTrue())
		},
		Entry("higher authority takes control", xcontrol.Authority(100), xcontrol.Authority(200), opSubject),
		Entry("equal authority is ignored", xcontrol.Authority(200), xcontrol.Authority(200), arcSubject),
		Entry("lower authority is ignored", xcontrol.Authority(200), xcontrol.Authority(100), arcSubject),
	)

	Describe("ApplySeries", func() {
		It("Should apply an acquire encoded as JSON", func() {
			s := control.New()
			s.ApplySeries(telem.NewSeriesV(
				`{"transfers":[{"to":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200}}]}`,
			))
			Expect(s.IsAuthorized(1, arcSubject)).To(BeTrue())
			Expect(s.IsAuthorized(1, opSubject)).To(BeFalse())
		})
		It("Should apply a release encoded as JSON", func() {
			s := control.New()
			s.ApplyIncrease(arcSubject, 1, 200)
			s.ApplySeries(telem.NewSeriesV(
				`{"transfers":[{"from":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200},"to":null}]}`,
			))
			Expect(s.IsAuthorized(1, opSubject)).To(BeTrue())
		})
		It("Should ignore invalid JSON", func() {
			s := control.New()
			s.ApplySeries(telem.NewSeriesV("not valid json"))
			Expect(s.IsAuthorized(1, arcSubject)).To(BeTrue())
		})
		It("Should ignore a non-string series", func() {
			s := control.New()
			s.ApplyIncrease(arcSubject, 1, 200)
			s.ApplySeries(telem.NewSeriesV[float32](1.0))
			Expect(s.IsAuthorized(1, arcSubject)).To(BeTrue())
		})
		It("Should apply multiple updates in one series", func() {
			s := control.New()
			s.ApplySeries(telem.NewSeriesV(
				`{"transfers":[{"to":{"resource":1,"subject":{"key":"arc-1","name":"arc"},"authority":200}}]}`,
				`{"transfers":[{"to":{"resource":2,"subject":{"key":"op-1","name":"operator"},"authority":250}}]}`,
			))
			Expect(s.IsAuthorized(1, arcSubject)).To(BeTrue())
			Expect(s.IsAuthorized(2, opSubject)).To(BeTrue())
			Expect(s.IsAuthorized(1, opSubject)).To(BeFalse())
			Expect(s.IsAuthorized(2, arcSubject)).To(BeFalse())
		})
	})

	Describe("Holder", func() {
		It("Should return the holder for a controlled channel", func() {
			s := control.New()
			s.Apply(acquireUpdate(arcSubject, 1, 200))
			holder := MustBeOk(s.Holder(1))
			Expect(holder.Subject).To(Equal(arcSubject))
			Expect(holder.Authority).To(Equal(xcontrol.Authority(200)))
		})
		It("Should report absence for an uncontrolled channel", func() {
			s := control.New()
			_, ok := s.Holder(1)
			Expect(ok).To(BeFalse())
		})
	})
})
