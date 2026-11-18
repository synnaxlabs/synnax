// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Built-in enumerations", func() {
	var db *gorp.DB
	BeforeEach(func() {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
	})
	open := func(ctx SpecContext) *enumeration.Service {
		GinkgoHelper()
		return MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
	}
	stored := func(ctx SpecContext, svc *enumeration.Service) []enumeration.Enumeration {
		GinkgoHelper()
		var res []enumeration.Enumeration
		Expect(svc.NewRetrieve().Entries(&res).Exec(ctx, db)).To(Succeed())
		return res
	}
	byName := func(
		enumerations []enumeration.Enumeration,
		name string,
	) enumeration.Enumeration {
		GinkgoHelper()
		e, ok := lo.Find(enumerations, func(e enumeration.Enumeration) bool {
			return e.Name == name
		})
		Expect(ok).To(BeTrue())
		return e
	}
	type entry struct {
		value float64
		name  string
	}
	entries := func(e enumeration.Enumeration) []entry {
		return lo.Map(e.Entries, func(en enumeration.Entry, _ int) entry {
			return entry{en.Value, en.Name}
		})
	}

	It("Should list the values that a new Core stores", func(ctx SpecContext) {
		Expect(enumeration.Builtins()).To(ConsistOf(stored(ctx, open(ctx))))
	})

	It("Should not let a change to one list reach the next", func() {
		first := enumeration.Builtins()
		first[0].Name = "Changed"
		first[0].Entries[0].Name = "Changed"
		next := enumeration.Builtins()
		Expect(next[0].Name).To(Equal("Valve"))
		Expect(next[0].Entries[0].Name).To(Equal("Closed"))
	})

	DescribeTable(
		"Should ship each built-in with its two states",
		func(ctx SpecContext, name, zero, one string) {
			e := byName(stored(ctx, open(ctx)), name)
			Expect(entries(e)).To(Equal([]entry{{0, zero}, {1, one}}))
			Expect(e.Entries).To(HaveEach(SatisfyAll(
				HaveField("Abnormal", BeFalse()),
				HaveField("Color", BeNil()),
			)))
		},
		Entry("Valve", "Valve", "Closed", "Open"),
		Entry("Normally open valve", "Normally open valve", "Open", "Closed"),
		Entry("Pump", "Pump", "Stopped", "Running"),
		Entry("Switch", "Switch", "Off", "On"),
		Entry("Light", "Light", "Off", "On"),
	)

	It("Should keep an edited built-in when the Core opens again", func(
		ctx SpecContext,
	) {
		svc := open(ctx)
		valve := byName(stored(ctx, svc), "Valve")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Apply(ctx, valve.Key, enumeration.NewSetAction(
				enumeration.SetPayload{
					Path:  []string{"entries", "open", "name"},
					Value: "Opened",
				},
			))
		})).To(Succeed())
		Expect(svc.Close()).To(Succeed())
		all := stored(ctx, open(ctx))
		Expect(all).To(HaveLen(5))
		Expect(entries(byName(all, "Valve"))).
			To(Equal([]entry{{0, "Closed"}, {1, "Opened"}}))
	})

	It("Should create only the missing built-ins", func(ctx SpecContext) {
		svc := open(ctx)
		all := stored(ctx, svc)
		pump := byName(all, "Pump")
		valve := byName(all, "Valve")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := svc.NewWriter(tx).Apply(
				ctx,
				valve.Key,
				enumeration.NewSetAction(
					enumeration.SetPayload{Path: []string{"name"}, Value: "Gate"},
				),
			); err != nil {
				return err
			}
			// A release that adds a built-in finds it missing, as here.
			return gorp.NewDelete[enumeration.Key, enumeration.Enumeration]().
				Where(gorp.MatchKeys[enumeration.Key, enumeration.Enumeration](
					pump.Key,
				)).
				Exec(ctx, tx)
		})).To(Succeed())
		Expect(svc.Close()).To(Succeed())
		all = stored(ctx, open(ctx))
		Expect(all).To(HaveLen(5))
		Expect(byName(all, "Pump")).To(Equal(pump))
		Expect(byName(all, "Gate").Key).To(Equal(valve.Key))
	})

	It("Should refuse to delete a built-in", func(ctx SpecContext) {
		svc := open(ctx)
		valve := byName(stored(ctx, svc), "Valve")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, valve.Key)
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot delete built-in enumeration Valve")),
		))
		Expect(stored(ctx, svc)).To(HaveLen(5))
	})

	It("Should delete nothing when one key names a built-in", func(ctx SpecContext) {
		svc := open(ctx)
		valve := byName(stored(ctx, svc), "Valve")
		tx := DeferClose(db.OpenTx())
		w := svc.NewWriter(tx)
		mine := enumeration.Enumeration{Name: "Breaker"}
		Expect(createOne(ctx, w, &mine)).To(Succeed())
		Expect(w.Delete(ctx, mine.Key, valve.Key)).
			To(MatchError(ContainSubstring("cannot delete built-in enumeration Valve")))
		Expect(svc.NewRetrieve().
			Where(enumeration.MatchKeys(mine.Key)).
			Exists(ctx, tx)).To(BeTrue())
	})
})
