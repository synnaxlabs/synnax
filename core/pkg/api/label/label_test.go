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
	"context"
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
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

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
			apilabel.DispatchRequest{Key: l.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveLabel(ctx, l.Key).Name).To(Equal("Anomaly"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		l := createLabel(ctx)
		Expect(apiSvc.Dispatch(
			AuthedCtx(ctx, u),
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

var _ = Describe("Add and Remove", func() {
	authed := func(ctx SpecContext, types ...resource.Type) context.Context {
		GinkgoHelper()
		u := createUser(ctx)
		ids := []resource.ID{{Type: resource.TypeLabel}}
		for _, t := range types {
			ids = append(ids, resource.ID{Type: t})
		}
		grant(ctx, access.ActionUpdate, u.ResourceID(), ids...)
		return AuthedCtx(ctx, u)
	}
	createRange := func(ctx SpecContext) ranger.Range {
		GinkgoHelper()
		r := ranger.Range{Name: "Hot fire", TimeRange: telem.NewRangeSeconds(1, 2)}
		Expect(svc.Ranger.NewWriter(nil).Create(ctx, &r)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.Ranger.NewWriter(nil).Delete(ctx, r.Key)).To(Succeed())
		})
		return r
	}
	rangeLabels := func(ctx SpecContext, key ranger.Key) []label.Key {
		GinkgoHelper()
		var r ranger.Range
		Expect(svc.Ranger.NewRetrieve().
			Where(ranger.MatchKeys(key)).
			Entry(&r).
			Exec(ctx, nil)).To(Succeed())
		return r.Labels
	}

	It("Should add labels to a range", func(ctx SpecContext) {
		r, a, b := createRange(ctx), createLabel(ctx), createLabel(ctx)
		actx := authed(ctx, resource.TypeRange)
		MustSucceed(apiSvc.Add(actx, nil, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		MustSucceed(apiSvc.Add(actx, nil, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{b.Key},
		}))
		Expect(rangeLabels(ctx, r.Key)).To(ConsistOf(a.Key, b.Key))
	})

	It("Should replace the labels of a range", func(ctx SpecContext) {
		r, a, b := createRange(ctx), createLabel(ctx), createLabel(ctx)
		actx := authed(ctx, resource.TypeRange)
		MustSucceed(apiSvc.Add(actx, nil, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		MustSucceed(apiSvc.Add(actx, nil, apilabel.AddRequest{
			ID:      r.ResourceID(),
			Labels:  []label.Key{b.Key},
			Replace: true,
		}))
		Expect(rangeLabels(ctx, r.Key)).To(Equal([]label.Key{b.Key}))
	})

	It("Should add and remove the labels of a status", func(ctx SpecContext) {
		a, b := createLabel(ctx), createLabel(ctx)
		s := status.Status[any]{
			Name:    "Alarm",
			Variant: status.VariantInfo,
			Time:    telem.Now(),
		}
		Expect(svc.Status.NewWriter(nil).Set(ctx, &s)).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(svc.Status.NewWriter(nil).Delete(ctx, s.Key)).To(Succeed())
		})
		actx := authed(ctx, resource.TypeStatus)
		MustSucceed(apiSvc.Add(actx, nil, apilabel.AddRequest{
			ID:     s.ResourceID(),
			Labels: []label.Key{a.Key, b.Key},
		}))
		MustSucceed(apiSvc.Remove(actx, nil, apilabel.RemoveRequest{
			ID:     s.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		var stored status.Status[any]
		Expect(svc.Status.NewRetrieve[any]().
			Where(status.MatchKeys[any](s.Key)).
			Entry(&stored).
			Exec(ctx, nil)).To(Succeed())
		Expect(stored.Labels).To(Equal([]label.Key{b.Key}))
	})

	It("Should remove labels from a range", func(ctx SpecContext) {
		r, a, b := createRange(ctx), createLabel(ctx), createLabel(ctx)
		actx := authed(ctx, resource.TypeRange)
		MustSucceed(apiSvc.Add(actx, nil, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key, b.Key},
		}))
		MustSucceed(apiSvc.Remove(actx, nil, apilabel.RemoveRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		Expect(rangeLabels(ctx, r.Key)).To(Equal([]label.Key{b.Key}))
	})

	It("Should reject a resource that cannot have labels", func(ctx SpecContext) {
		a := createLabel(ctx)
		id := resource.ID{Type: resource.TypeChannel, Key: "1"}
		actx := authed(ctx, resource.TypeChannel)
		Expect(apiSvc.Add(actx, nil, apilabel.AddRequest{
			ID:     id,
			Labels: []label.Key{a.Key},
		})).Error().To(MatchError(ContainSubstring("id: a channel cannot have labels")))
	})
})
