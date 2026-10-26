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
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// The case table is shared with the TypeScript matches, so that the Core and the
// client cache agree on every filter.
const vectorsPath = "../../../../x/testdata/filter/vectors.json"

type whereCase struct {
	Name     string       `json:"name"`
	Where    ranger.Where `json:"where"`
	Expected []string     `json:"expected"`
}

type vectors struct {
	Ranges struct {
		Records map[string]ranger.Range `json:"records"`
		Cases   []whereCase             `json:"cases"`
	} `json:"ranges"`
}

var _ = Describe("Where", func() {
	var v vectors
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &v)).To(Succeed())
	keys := lo.MapEntries(v.Ranges.Records, func(
		alias string,
		r ranger.Range,
	) (ranger.Key, string) {
		return r.Key, alias
	})

	BeforeEach(func(ctx SpecContext) {
		labels := make(map[label.Key]label.Label)
		for _, r := range v.Ranges.Records {
			r.Labels = nil
			Expect(svc.NewWriter(tx).Create(ctx, &r)).To(Succeed())
		}
		for _, r := range v.Ranges.Records {
			for _, l := range r.Labels {
				labels[l.Key] = l
			}
		}
		for _, l := range labels {
			Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		}
		for _, r := range v.Ranges.Records {
			Expect(labelSvc.NewWriter(tx).Label(
				ctx,
				r.OntologyID(),
				lo.Map(r.Labels, func(l label.Label, _ int) label.Key { return l.Key }),
			)).To(Succeed())
		}
	})

	DescribeTable(
		"Should match the case",
		func(ctx SpecContext, w ranger.Where, expected []string) {
			f := MustSucceed(ranger.MatchWhere(w))
			var res []ranger.Range
			// The API skips not-found, since a missing key is a non-match.
			Expect(errors.Skip(
				svc.NewRetrieve().Where(f).Entries(&res).Exec(ctx, tx),
				query.ErrNotFound,
			)).To(Succeed())
			Expect(lo.Map(res, func(r ranger.Range, _ int) string {
				return keys[r.Key]
			})).To(ConsistOf(expected))
		},
		lo.Map(v.Ranges.Cases, func(c whereCase, _ int) TableEntry {
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
