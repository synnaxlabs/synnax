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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Where", Ordered, func() {
	var (
		svc      *status.Service
		labelSvc *label.Service
		tx       gorp.Tx
	)
	v := MustSucceed(
		testutil.ReadVectors[status.Status[any], status.Where]("statuses"),
	)
	keys := lo.MapEntries(v.Records, func(
		alias string,
		s status.Status[any],
	) (status.Key, string) {
		return s.Key, alias
	})
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(
			ctx,
			group.ServiceConfig{DB: db, Ontology: otg},
		))
		labelSvc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		svc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Label:    labelSvc,
			Group:    g,
		}))
		tx = DeferClose(db.OpenTx())
		labels := make(map[label.Key]label.Label)
		for _, s := range v.Records {
			for _, l := range s.Labels {
				labels[l.Key] = l
			}
			s.Labels = nil
			Expect(svc.NewWriter(tx).Set(ctx, &s)).To(Succeed())
		}
		for _, l := range labels {
			Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		}
		for _, s := range v.Records {
			Expect(labelSvc.NewWriter(tx).Label(
				ctx,
				s.OntologyID(),
				lo.Map(s.Labels, func(l label.Label, _ int) label.Key { return l.Key }),
			)).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, w status.Where, expected []string) {
			f := MustSucceed(status.MatchWhere[any](w))
			var res []status.Status[any]
			Expect(svc.NewRetrieve[any]().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(lo.Map(res, func(s status.Status[any], _ int) string {
				return keys[s.Key]
			})).To(ConsistOf(expected))
		},
		lo.Map(v.Cases, func(c testutil.Case[status.Where], _ int) TableEntry {
			return Entry(c.Name, c.Where, c.Expected)
		}),
	)

	DescribeTable(
		"Should return a path error for an invalid variant",
		func(variant filter.Set[status.Variant], msg string) {
			Expect(status.MatchWhere[any](status.Where{Variant: &variant})).Error().
				To(SatisfyAll(
					BeAssignableToTypeOf(validate.PathError{}),
					MatchError(msg),
				))
		},
		Entry(
			"eq",
			filter.Set[status.Variant]{Eq: new(status.Variant("bogus"))},
			`variant.eq: invalid variant "bogus": validation error`,
		),
		Entry(
			"in",
			filter.Set[status.Variant]{
				In: []status.Variant{status.VariantError, "bogus"},
			},
			`variant.in.1: invalid variant "bogus": validation error`,
		),
	)
})
