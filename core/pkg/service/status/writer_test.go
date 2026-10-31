// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Problem times", func() {
	type step struct {
		variant status.Variant
		at      telem.TimeStamp
	}
	var (
		db  *gorp.DB
		svc *status.Service
		now telem.TimeStamp
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		lbl := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    lbl,
			Now:      func() telem.TimeStamp { return now },
		}))
	})
	retrieve := func(ctx context.Context, key status.Key) status.Status[any] {
		GinkgoHelper()
		var s status.Status[any]
		Expect(svc.NewRetrieve[any]().
			Where(status.MatchKeys[any](key)).
			Entry(&s).
			Exec(ctx, nil)).To(Succeed())
		return s
	}
	set := func(ctx context.Context, key status.Key, st step) status.Status[any] {
		GinkgoHelper()
		now = st.at
		Expect(svc.NewWriter(nil).Set(ctx, &status.Status[any]{
			Key:     key,
			Variant: st.variant,
			Message: "Message",
			Time:    st.at,
		})).To(Succeed())
		return retrieve(ctx, key)
	}
	at := func(seconds int) telem.TimeStamp {
		return telem.TimeStamp(seconds) * telem.SecondTS
	}

	DescribeTable(
		"Should set since and cleared from the previous status",
		func(ctx SpecContext, steps []step, since, cleared *telem.TimeStamp) {
			key := uuid.New().String()
			var got status.Status[any]
			for _, st := range steps {
				got = set(ctx, key, st)
			}
			Expect(got.Since).To(Equal(since))
			Expect(got.Cleared).To(Equal(cleared))
		},
		Entry(
			"good, cleared less than 30 s ago, then bad: since kept",
			[]step{
				{status.VariantError, at(10)},
				{status.VariantSuccess, at(20)},
				{status.VariantWarning, at(49)},
			},
			new(at(10)),
			nil,
		),
		Entry(
			"good, cleared 30 s ago, then bad: since now",
			[]step{
				{status.VariantError, at(10)},
				{status.VariantSuccess, at(20)},
				{status.VariantError, at(50)},
			},
			new(at(50)),
			nil,
		),
		Entry(
			"good, never cleared, then bad: since now",
			[]step{{status.VariantSuccess, at(10)}, {status.VariantError, at(15)}},
			new(at(15)),
			nil,
		),
		Entry(
			"no status, then bad: since now",
			[]step{{status.VariantWarning, at(10)}},
			new(at(10)),
			nil,
		),
		Entry(
			"bad, then bad: since kept",
			[]step{{status.VariantWarning, at(10)}, {status.VariantError, at(90)}},
			new(at(10)),
			nil,
		),
		Entry(
			"bad, then good: since kept, cleared now",
			[]step{{status.VariantError, at(10)}, {status.VariantSuccess, at(25)}},
			new(at(10)),
			new(at(25)),
		),
		Entry(
			"good, then good: since and cleared kept",
			[]step{
				{status.VariantError, at(10)},
				{status.VariantSuccess, at(25)},
				{status.VariantInfo, at(90)},
			},
			new(at(10)),
			new(at(25)),
		),
		Entry(
			"no status, then good: since and cleared empty",
			[]step{{status.VariantSuccess, at(10)}},
			nil,
			nil,
		),
	)

	DescribeTable(
		"Should treat only warnings and errors as problems",
		func(ctx SpecContext, variant status.Variant, since *telem.TimeStamp) {
			got := set(ctx, uuid.New().String(), step{variant, at(10)})
			Expect(got.Since).To(Equal(since))
		},
		Entry("success", status.VariantSuccess, nil),
		Entry("info", status.VariantInfo, nil),
		Entry("warning", status.VariantWarning, new(at(10))),
		Entry("error", status.VariantError, new(at(10))),
		Entry("loading", status.VariantLoading, nil),
		Entry("disabled", status.VariantDisabled, nil),
	)

	It(
		"Should set since on a bad status stored without one",
		func(ctx SpecContext) {
			stored := status.Status[any]{
				Key:     uuid.New().String(),
				Variant: status.VariantError,
				Message: "Message",
				Time:    at(10),
			}
			Expect(gorp.NewCreate[status.Key, status.Status[any]]().
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			got := set(ctx, stored.Key, step{status.VariantError, at(20)})
			Expect(got.Since).To(Equal(new(at(20))))
		},
	)

	It("Should ignore the times a client sends", func(ctx SpecContext) {
		now = at(30)
		s := status.Status[any]{
			Key:     uuid.New().String(),
			Variant: status.VariantSuccess,
			Message: "Message",
			Time:    at(30),
			Since:   new(at(1)),
			Cleared: new(at(2)),
		}
		Expect(svc.NewWriter(nil).Set(ctx, &s)).To(Succeed())
		Expect(s.Since).To(BeNil())
		Expect(s.Cleared).To(BeNil())
		s.Variant = status.VariantError
		s.Since, s.Cleared = new(at(3)), new(at(4))
		Expect(svc.NewWriter(nil).Set(ctx, &s)).To(Succeed())
		got := retrieve(ctx, s.Key)
		Expect(got.Since).To(Equal(new(at(30))))
		Expect(got.Cleared).To(BeNil())
	})

	It("Should set since and cleared on each status of SetMany", func(ctx SpecContext) {
		cleared := uuid.New().String()
		set(ctx, cleared, step{status.VariantError, at(10)})
		now = at(20)
		statuses := []status.Status[any]{
			{
				Key:     cleared,
				Variant: status.VariantSuccess,
				Message: "Message",
				Time:    at(20),
			},
			{
				Key:     uuid.New().String(),
				Variant: status.VariantWarning,
				Message: "Message",
				Time:    at(20),
			},
		}
		Expect(svc.NewWriter(nil).SetMany(ctx, &statuses)).To(Succeed())
		Expect(statuses[0].Since).To(Equal(new(at(10))))
		Expect(statuses[0].Cleared).To(Equal(new(at(20))))
		Expect(statuses[1].Since).To(Equal(new(at(20))))
		Expect(statuses[1].Cleared).To(BeNil())
		Expect(retrieve(ctx, cleared).Cleared).To(Equal(new(at(20))))
		Expect(retrieve(ctx, statuses[1].Key).Since).To(Equal(new(at(20))))
	})

	It("Should keep since and cleared on a rename", func(ctx SpecContext) {
		key := uuid.New().String()
		set(ctx, key, step{status.VariantError, at(10)})
		set(ctx, key, step{status.VariantSuccess, at(20)})
		now = at(30)
		Expect(svc.NewWriter(nil).Rename(ctx, key, "Renamed")).To(Succeed())
		got := retrieve(ctx, key)
		Expect(got.Name).To(Equal("Renamed"))
		Expect(got.Since).To(Equal(new(at(10))))
		Expect(got.Cleared).To(Equal(new(at(20))))
	})
})
