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
)

var _ = Describe("Delete order", func() {
	// holding returns a Deleter whose records each a record of type holder holds.
	holding := func(holder resource.Type) impact.Deleter {
		return impact.Deleter{
			Plan: func(
				_ context.Context,
				_ gorp.Snapshot,
				ids []resource.ID,
			) ([]resource.ID, gorp.DeleteEffects, error) {
				var effects gorp.DeleteEffects
				for _, id := range ids {
					effects.Restricts = append(effects.Restricts, gorp.DeleteEffect{
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

	It("Should fail a delete whose types hold each other", func(ctx SpecContext) {
		a, b := resource.Type("a"), resource.Type("b")
		svc := MustSucceed(impact.OpenService(impact.ServiceConfig{
			Deleters: map[resource.Type]impact.Deleter{a: holding(b), b: holding(a)},
			RecordOf: func(t gorp.Target) resource.ID {
				return resource.ID{Type: resource.Type(t.Table), Key: t.Key}
			},
		}))
		Expect(svc.Plan(
			ctx,
			nil,
			[]resource.ID{{Type: a, Key: "1"}, {Type: b, Key: "1"}},
			impact.PlanOptions{},
		)).Error().To(MatchError("[impact] - the records hold each other in a cycle"))
	})
})
