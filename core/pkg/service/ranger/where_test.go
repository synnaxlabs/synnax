// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger_test

import (
	"encoding/json/v2"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Where", func() {
	v := MustSucceed(testutil.ReadVectors[ranger.Range, ranger.Where]("ranges"))
	keys := lo.MapEntries(v.Records, func(
		alias string,
		r ranger.Range,
	) (ranger.Key, string) {
		return r.Key, alias
	})

	BeforeEach(func(ctx SpecContext) {
		openTx()
		for _, k := range lo.Uniq(lo.FlatMap(
			lo.Values(v.Records),
			func(r ranger.Range, _ int) []label.Key { return r.Labels },
		)) {
			l := label.Label{Key: k, Name: k.String()}
			Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		}
		for _, r := range v.Records {
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, w ranger.Where, expected []string) {
			f := MustSucceed(ranger.MatchWhere(w))
			var res []ranger.Range
			Expect(svc.NewRetrieve().
				SkipMissingKeys().
				Where(f).
				Entries(&res).
				Exec(ctx, tx)).To(Succeed())
			Expect(lo.Map(res, func(r ranger.Range, _ int) string {
				return keys[r.Key]
			})).To(ConsistOf(expected))
		},
		lo.Map(v.Cases, func(c testutil.Case[ranger.Where], _ int) TableEntry {
			return Entry(c.Name, c.Where, c.Expected)
		}),
	)

	It("Should return a path error for an invalid regex", func() {
		var w ranger.Where
		Expect(json.Unmarshal(
			[]byte(`{"or":[{"key":{"in":[]}},{"name":{"regex":"(a"}}]}`),
			&w,
		)).To(Succeed())
		Expect(ranger.MatchWhere(w)).Error().To(SatisfyAll(
			BeAssignableToTypeOf(validate.PathError{}),
			MatchError("or.1.name.regex: error parsing regexp: missing closing ): "+
				"`(a`: validation error"),
		))
	})
})
