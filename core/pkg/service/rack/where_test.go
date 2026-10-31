// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Where", Ordered, func() {
	var (
		v   testutil.Vectors[rack.Rack, rack.Where]
		svc *rack.Service
	)
	v = MustSucceed(testutil.ReadVectors[rack.Rack, rack.Where]("racks"))
	aliases := lo.MapEntries(v.Records, func(
		alias string,
		r rack.Rack,
	) (rack.Key, string) {
		return r.Key, alias
	})

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db := DeferClose(gorp.Wrap(memkv.New()))
		// The stack's host is node 1, so racks a and b are on the host.
		svc = DeferClose(openStack(ctx, db, rack.ServiceConfig{})).rack
		for _, r := range v.Records {
			Expect(svc.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, c testutil.Case[rack.Where]) {
			f := MustSucceed(rack.MatchWhere(c.Where))
			var res []rack.Rack
			Expect(svc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, nil)).To(Succeed())
			// The service creates its own embedded rack, which the table does not hold.
			matched := lo.FilterMap(res, func(r rack.Rack, _ int) (string, bool) {
				alias, ok := aliases[r.Key]
				return alias, ok
			})
			Expect(matched).To(ConsistOf(c.Expected))
		},
		lo.Map(v.Cases, func(c testutil.Case[rack.Where], _ int) TableEntry {
			return Entry(c.Name, c)
		}),
	)

	It("Should return a path error for an invalid regex", func() {
		var w rack.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(rack.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError(
				"or.1.name.regex: error parsing regexp: missing closing ): `(a`: validation error",
			),
		))
	})
})
