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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apilabel "github.com/synnaxlabs/synnax/pkg/api/label"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	var (
		lbl     label.Label
		rangeID resource.ID
		labels  resource.ID
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
		rangeID = rng.ResourceID()
		Expect(svc.Label.NewWriter(nil).Label(
			ctx, rangeID, []label.Key{lbl.Key},
		)).To(Succeed())
		labels = resource.ID{Type: resource.TypeLabel}
	})

	DescribeTable(
		"Should deny the retrieve when a for line names a resource without access",
		func(ctx SpecContext, where func(id resource.ID) label.Where) {
			u := createUser(ctx)
			grant(ctx, access.ActionRetrieve, u.ResourceID(), labels)
			Expect(apiSvc.Retrieve(AuthedCtx(ctx, u), apilabel.RetrieveRequest{
				Where: new(where(rangeID)),
			})).Error().To(MatchError(access.ErrDenied))
		},
		Entry("alone", func(id resource.ID) label.Where {
			return label.Where{For: &id}
		}),
		Entry("under or", func(id resource.ID) label.Where {
			return label.Where{Or: []label.Where{
				{For: &id},
				{Name: &filter.String{Eq: new("Depressurized")}},
			}}
		}),
		Entry("under not", func(id resource.ID) label.Where {
			return label.Where{Not: &label.Where{For: &id}}
		}),
	)

	It("Should return the labels of a resource the subject can retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), labels, rangeID)
		res := MustSucceed(apiSvc.Retrieve(AuthedCtx(ctx, u), apilabel.RetrieveRequest{
			Where: &label.Where{For: &rangeID},
		}))
		Expect(res.Labels).To(ConsistOf(HaveField("Key", lbl.Key)))
	})
})

func createLabel(ctx SpecContext) label.Label {
	GinkgoHelper()
	l := label.Label{Name: "Nominal"}
	Expect(svc.Label.NewWriter(nil).Create(ctx, &l)).To(Succeed())
	return l
}

func retrieveLabel(ctx SpecContext, key label.Key) label.Label {
	GinkgoHelper()
	var l label.Label
	Expect(svc.Label.NewRetrieve().
		Where(label.MatchKeys(key)).
		Entry(&l).
		Exec(ctx, nil)).To(Succeed())
	return l
}

var _ = Describe("Dispatch", func() {
	rename := []label.Action{label.NewSetAction(label.SetPayload{
		Path:  []string{"name"},
		Value: "Anomaly",
	})}
	It("Should apply the actions when update is granted", func(ctx SpecContext) {
		u := createUser(ctx)
		l := createLabel(ctx)
		grant(ctx, access.ActionUpdate, u.ResourceID(), l.ResourceID())
		MustSucceed(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apilabel.DispatchRequest{Key: l.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveLabel(ctx, l.Key).Name).To(Equal("Anomaly"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		l := createLabel(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apilabel.DispatchRequest{Key: l.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveLabel(ctx, l.Key).Name).To(Equal("Nominal"))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), label.ResourceID(key))
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
			nil,
			apilabel.DispatchRequest{Key: key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(query.ErrNotFound))
	})
})

var _ = Describe("Create", func() {
	It(
		"Should stamp the dispatch key onto the create broadcast",
		func(ctx SpecContext) {
			u := createUser(ctx)
			grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
				Type: resource.TypeLabel,
			})
			rec := &Recorder[label.Key, label.Action]{}
			DeferCleanup(svc.Label.ObserveActions().OnChange(rec.Record))
			res := MustSucceed(
				apiSvc.Create(AuthedCtx(ctx, u), nil, apilabel.CreateRequest{
					Labels:      []label.Label{{Name: "Created"}},
					DispatchKey: "dk-create",
				}),
			)
			Expect(res.Labels).To(HaveLen(1))
			frames := rec.Snapshot()
			Expect(frames).To(HaveLen(1))
			Expect(frames[0].DispatchKey).To(Equal("dk-create"))
		},
	)
})
