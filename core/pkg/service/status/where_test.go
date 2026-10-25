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
	"encoding/json/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
)

// The case table is shared with the TypeScript matches, so that the Core and the
// client cache agree on every filter.
const vectorsPath = "../../../../x/testdata/filter/vectors.json"

type whereCase struct {
	Name     string       `json:"name"`
	Where    status.Where `json:"where"`
	Expected []string     `json:"expected"`
}

type vectors struct {
	Statuses struct {
		Records map[string]status.Status[any] `json:"records"`
		Cases   []whereCase                   `json:"cases"`
	} `json:"statuses"`
}

var _ = Describe("Where", Ordered, func() {
	var (
		v        vectors
		svc      *status.Service
		labelSvc *label.Service
		tx       gorp.Tx
	)
	Expect(json.Unmarshal(MustSucceed(os.ReadFile(vectorsPath)), &v)).To(Succeed())
	keys := lo.MapEntries(v.Statuses.Records, func(
		alias string,
		s status.Status[any],
	) (status.Key, string) {
		return s.Key, alias
	})
	BeforeAll(func(ctx SpecContext) {
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
		for _, s := range v.Statuses.Records {
			for _, l := range s.Labels {
				labels[l.Key] = l
			}
			s.Labels = nil
			Expect(svc.NewWriter(tx).Set(ctx, &s)).To(Succeed())
		}
		for _, l := range labels {
			Expect(labelSvc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
		}
		for _, s := range v.Statuses.Records {
			Expect(labelSvc.NewWriter(tx).Label(
				ctx,
				s.OntologyID(),
				lo.Map(s.Labels, func(l label.Label, _ int) label.Key { return l.Key }),
			)).To(Succeed())
		}
	})

	for _, c := range v.Statuses.Cases {
		It("Should match the case: "+c.Name, func(ctx SpecContext) {
			f := MustSucceed(status.MatchWhere[any](c.Where))
			var res []status.Status[any]
			// The API skips not-found, since a missing key is a non-match.
			Expect(errors.Skip(
				svc.NewRetrieve[any]().Where(f).Entries(&res).Exec(ctx, tx),
				query.ErrNotFound,
			)).To(Succeed())
			Expect(lo.Map(res, func(s status.Status[any], _ int) string {
				return keys[s.Key]
			})).To(ConsistOf(c.Expected))
		})
	}
})
