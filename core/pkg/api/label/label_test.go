// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilabel "github.com/synnaxlabs/synnax/pkg/api/label"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	var (
		lbl     label.Label
		rangeID ontology.ID
		labels  ontology.ID
	)
	BeforeEach(func(ctx SpecContext) {
		lbl = label.Label{Name: "Pressurized"}
		Expect(svc.Label.NewWriter(nil).Create(ctx, &lbl)).To(Succeed())
		rng := ranger.Range{
			Name: "Hotfire",
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(10 * telem.Second),
			},
		}
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &rng)).To(Succeed())
		rangeID = rng.OntologyID()
		Expect(svc.Label.NewWriter(nil).Label(
			ctx, rangeID, []label.Key{lbl.Key},
		)).To(Succeed())
		labels = ontology.ID{Type: ontology.ResourceTypeLabel}
	})

	DescribeTable(
		"Should deny the retrieve when a for line names a resource without access",
		func(ctx SpecContext, where func(id ontology.ID) label.Where) {
			u := createUser(ctx)
			grantRetrieveOn(ctx, u.OntologyID(), labels)
			Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apilabel.RetrieveRequest{
				Where: new(where(rangeID)),
			})).Error().To(MatchError(access.ErrDenied))
		},
		Entry("alone", func(id ontology.ID) label.Where {
			return label.Where{For: &id}
		}),
		Entry("under or", func(id ontology.ID) label.Where {
			return label.Where{Or: []label.Where{
				{For: &id},
				{Name: &filter.String{Eq: new("Depressurized")}},
			}}
		}),
		Entry("under not", func(id ontology.ID) label.Where {
			return label.Where{Not: &label.Where{For: &id}}
		}),
	)

	It("Should return the labels of a resource the subject can retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantRetrieveOn(ctx, u.OntologyID(), labels, rangeID)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apilabel.RetrieveRequest{
			Where: &label.Where{For: &rangeID},
		}))
		Expect(res.Labels).To(ConsistOf(HaveField("Key", lbl.Key)))
	})
})
