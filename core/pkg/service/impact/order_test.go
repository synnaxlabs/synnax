// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Delete order", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	recordOf := func(t gorp.Target) resource.ID {
		return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
	}
	// held returns a Deleter whose each record a record of type holder holds, with the
	// effects that add returns for the link.
	held := func(
		holder resource.Type,
		add func(*gorp.DeleteEffects, gorp.DeleteEffect),
	) impact.Deleter {
		return impact.Deleter{
			Plan: func(
				_ context.Context,
				_ gorp.Snapshot,
				ids []resource.ID,
			) ([]resource.ID, gorp.DeleteEffects, error) {
				var effects gorp.DeleteEffects
				for _, id := range ids {
					add(&effects, gorp.DeleteEffect{
						Holder: gorp.Target{Table: string(holder), Key: id.Key},
						Target: gorp.Target{Table: string(id.Type), Key: id.Key},
					})
				}
				return ids, effects, nil
			},
			Delete: func(
				context.Context,
				gorp.Tx,
				[]resource.ID,
				gorp.DeleteEffects,
			) error {
				return nil
			},
		}
	}

	restrict := func(e *gorp.DeleteEffects, d gorp.DeleteEffect) {
		e.Restricts = append(e.Restricts, d)
	}
	weak := func(e *gorp.DeleteEffects, d gorp.DeleteEffect) {
		e.Weak = append(e.Weak, d)
	}
	nothing := func(*gorp.DeleteEffects, gorp.DeleteEffect) {}
	a, b := resource.Type("a"), resource.Type("b")

	It("Should fail a delete whose types hold each other", func(ctx SpecContext) {
		svc := MustSucceed(impact.OpenService(impact.ServiceConfig{
			Deleters: map[resource.Type]impact.Deleter{
				a: held(b, restrict),
				b: held(a, restrict),
			},
			RecordOf: recordOf,
		}))
		Expect(svc.Plan(
			ctx,
			db,
			[]resource.ID{{Type: a, Key: "1"}, {Type: b, Key: "1"}},
			impact.PlanOptions{},
		)).Error().To(MatchError("[impact] - the records hold each other in a cycle"))
	})

	It("Should not count a weak holder that the delete removes as kept", func(
		ctx SpecContext,
	) {
		svc := MustSucceed(impact.OpenService(impact.ServiceConfig{
			Deleters: map[resource.Type]impact.Deleter{
				a: held(b, weak),
				b: held(a, nothing),
			},
			RecordOf: recordOf,
		}))
		Expect(svc.Plan(
			ctx,
			db,
			[]resource.ID{{Type: a, Key: "1"}, {Type: b, Key: "1"}},
			impact.PlanOptions{},
		)).To(HaveField("Kept", BeZero()))
		Expect(svc.Plan(
			ctx,
			db,
			[]resource.ID{{Type: a, Key: "1"}},
			impact.PlanOptions{},
		)).To(HaveField("Kept", Equal(uint32(1))))
	})
})
