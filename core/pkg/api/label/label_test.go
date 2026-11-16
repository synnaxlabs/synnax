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
	"github.com/synnaxlabs/gorp"
	apilabel "github.com/synnaxlabs/synnax/pkg/api/label"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func createLabel(ctx SpecContext) label.Label {
	GinkgoHelper()
	l := label.Label{Name: "Nominal"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Label.NewWriter(tx).Create(ctx, &l)
	})).To(Succeed())
	return l
}

func retrieveLabel(ctx SpecContext, key label.Key) label.Label {
	GinkgoHelper()
	var l label.Label
	Expect(svc.Label.NewRetrieve().
		Where(label.MatchKeys(key)).
		Entry(&l).
		Exec(ctx, db)).To(Succeed())
	return l
}

func labelExists(ctx SpecContext, key label.Key) bool {
	GinkgoHelper()
	return MustSucceed(
		svc.Label.NewRetrieve().Where(label.MatchKeys(key)).Exists(ctx, db),
	)
}

func createRange(ctx SpecContext, labels ...label.Key) ranger.Range {
	GinkgoHelper()
	r := ranger.Range{
		Name:      "Hot fire",
		TimeRange: telem.NewRangeSeconds(1, 2),
		Labels:    labels,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Ranger.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Ranger.NewWriter(tx).Delete(ctx, r.Key)
		})).To(Succeed())
	})
	return r
}

func rangeLabels(ctx SpecContext, key ranger.Key) []label.Key {
	GinkgoHelper()
	var r ranger.Range
	Expect(svc.Ranger.NewRetrieve().
		Where(ranger.MatchKeys(key)).
		Entry(&r).
		Exec(ctx, db)).To(Succeed())
	return r.Labels
}

func createStatus(ctx SpecContext, labels ...label.Key) status.Status[any] {
	GinkgoHelper()
	s := status.Status[any]{
		Name:    "Alarm",
		Variant: status.VariantInfo,
		Time:    telem.Now(),
		Labels:  labels,
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Status.NewWriter(tx).Set(ctx, &s)
	})).To(Succeed())
	DeferCleanup(func(ctx SpecContext) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.Status.NewWriter(tx).Delete(ctx, s.Key)
		})).To(Succeed())
	})
	return s
}

func statusLabels(ctx SpecContext, key status.Key) []label.Key {
	GinkgoHelper()
	var s status.Status[any]
	Expect(svc.Status.NewRetrieve[any]().
		Where(status.MatchKeys[any](key)).
		Entry(&s).
		Exec(ctx, db)).To(Succeed())
	return s.Labels
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
		MustSucceed(dispatch(
			AuthedCtx(ctx, u),
			apilabel.DispatchRequest{Key: l.Key, DispatchKey: "dk", Actions: rename},
		))
		Expect(retrieveLabel(ctx, l.Key).Name).To(Equal("Anomaly"))
	})
	It("Should reject the request when update is not granted", func(ctx SpecContext) {
		u := createUser(ctx)
		l := createLabel(ctx)
		Expect(dispatch(
			AuthedCtx(ctx, u),
			apilabel.DispatchRequest{Key: l.Key, DispatchKey: "dk", Actions: rename},
		)).Error().To(MatchError(access.ErrDenied))
		Expect(retrieveLabel(ctx, l.Key).Name).To(Equal("Nominal"))
	})
	It("Should return the dispatch error", func(ctx SpecContext) {
		u := createUser(ctx)
		key := uuid.New()
		grant(ctx, access.ActionUpdate, u.ResourceID(), label.ResourceID(key))
		Expect(dispatch(
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
			DeferCleanup(svc.Label.Actions().OnChange(rec.Record))
			res := MustSucceed(
				create(AuthedCtx(ctx, u), apilabel.CreateRequest{
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

	It("Should refuse a create without create on the label", func(ctx SpecContext) {
		u := createUser(ctx)
		l := label.Label{Key: uuid.New(), Name: "Denied"}
		Expect(create(AuthedCtx(ctx, u), apilabel.CreateRequest{
			Labels: []label.Label{l},
		})).Error().To(denied(u, access.ActionCreate, l.ResourceID()))
		Expect(labelExists(ctx, l.Key)).To(BeFalse())
	})

	It("Should refuse a label whose key exists", func(ctx SpecContext) {
		u := createUser(ctx)
		grant(ctx, access.ActionCreate, u.ResourceID(), resource.ID{
			Type: resource.TypeLabel,
		})
		stored := createLabel(ctx)
		Expect(create(AuthedCtx(ctx, u), apilabel.CreateRequest{
			Labels: []label.Label{{Key: stored.Key, Name: "Overwritten"}},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("label %s already exists", stored.Key)),
		))
		Expect(retrieveLabel(ctx, stored.Key).Name).To(Equal("Nominal"))
	})
})

var _ = Describe("Delete", func() {
	var (
		u user.User
		l label.Label
		r ranger.Range
		s status.Status[any]
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx)
		l = createLabel(ctx)
		r = createRange(ctx, l.Key)
		s = createStatus(ctx, l.Key)
		grant(ctx, access.ActionDelete, u.ResourceID(), l.ResourceID())
	})
	deleteAs := func(ctx SpecContext) (struct{}, error) {
		return del(AuthedCtx(ctx, u), apilabel.DeleteRequest{
			Keys: []label.Key{l.Key},
		})
	}
	rangeID := func() resource.ID { return r.ResourceID() }
	statusID := func() resource.ID { return s.ResourceID() }
	DescribeTable(
		"Should refuse the delete and keep the label",
		func(ctx SpecContext, granted, refused func() resource.ID) {
			grant(ctx, access.ActionUpdate, u.ResourceID(), granted())
			Expect(deleteAs(ctx)).Error().
				To(denied(u, access.ActionUpdate, refused()))
			Expect(labelExists(ctx, l.Key)).To(BeTrue())
			Expect(rangeLabels(ctx, r.Key)).To(Equal([]label.Key{l.Key}))
		},
		Entry("without update on a range that holds it", statusID, rangeID),
		Entry("without update on a status that holds it", rangeID, statusID),
	)
	It("Should refuse the delete without delete on the label", func(ctx SpecContext) {
		v := createUser(ctx)
		grant(ctx, access.ActionUpdate, v.ResourceID(), rangeID(), statusID())
		Expect(del(AuthedCtx(ctx, v), apilabel.DeleteRequest{
			Keys: []label.Key{l.Key},
		})).Error().To(denied(v, access.ActionDelete, l.ResourceID()))
		Expect(labelExists(ctx, l.Key)).To(BeTrue())
	})
	It(
		"Should delete the label with update on each range and status that holds it",
		func(ctx SpecContext) {
			grant(ctx, access.ActionUpdate, u.ResourceID(), rangeID(), statusID())
			MustSucceed(deleteAs(ctx))
			Expect(labelExists(ctx, l.Key)).To(BeFalse())
			Expect(rangeLabels(ctx, r.Key)).To(BeEmpty())
			Expect(statusLabels(ctx, s.Key)).To(BeEmpty())
		},
	)
})

var _ = Describe("Add and Remove", func() {
	authed := func(ctx SpecContext, types ...resource.Type) context.Context {
		GinkgoHelper()
		u := createUser(ctx)
		ids := make([]resource.ID, len(types))
		for i, t := range types {
			ids[i] = resource.ID{Type: t}
		}
		grant(ctx, access.ActionUpdate, u.ResourceID(), ids...)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), resource.ID{
			Type: resource.TypeLabel,
		})
		return AuthedCtx(ctx, u)
	}

	It("Should add labels to a range", func(ctx SpecContext) {
		r, a, b := createRange(ctx), createLabel(ctx), createLabel(ctx)
		actx := authed(ctx, resource.TypeRange)
		MustSucceed(add(actx, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		MustSucceed(add(actx, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{b.Key},
		}))
		Expect(rangeLabels(ctx, r.Key)).To(ConsistOf(a.Key, b.Key))
	})

	It("Should replace the labels of a range", func(ctx SpecContext) {
		r, a, b := createRange(ctx), createLabel(ctx), createLabel(ctx)
		actx := authed(ctx, resource.TypeRange)
		MustSucceed(add(actx, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		MustSucceed(add(actx, apilabel.AddRequest{
			ID:      r.ResourceID(),
			Labels:  []label.Key{b.Key},
			Replace: true,
		}))
		Expect(rangeLabels(ctx, r.Key)).To(Equal([]label.Key{b.Key}))
	})

	It("Should add and remove the labels of a status", func(ctx SpecContext) {
		a, b := createLabel(ctx), createLabel(ctx)
		s := createStatus(ctx)
		actx := authed(ctx, resource.TypeStatus)
		MustSucceed(add(actx, apilabel.AddRequest{
			ID:     s.ResourceID(),
			Labels: []label.Key{a.Key, b.Key},
		}))
		MustSucceed(remove(actx, apilabel.RemoveRequest{
			ID:     s.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		Expect(statusLabels(ctx, s.Key)).To(Equal([]label.Key{b.Key}))
	})

	It("Should remove labels from a range", func(ctx SpecContext) {
		r, a, b := createRange(ctx), createLabel(ctx), createLabel(ctx)
		actx := authed(ctx, resource.TypeRange)
		MustSucceed(add(actx, apilabel.AddRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key, b.Key},
		}))
		MustSucceed(remove(actx, apilabel.RemoveRequest{
			ID:     r.ResourceID(),
			Labels: []label.Key{a.Key},
		}))
		Expect(rangeLabels(ctx, r.Key)).To(Equal([]label.Key{b.Key}))
	})

	It(
		"Should refuse an add without update on the range",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r, l := createRange(ctx), createLabel(ctx)
			grant(ctx, access.ActionRetrieve, u.ResourceID(), l.ResourceID())
			Expect(add(AuthedCtx(ctx, u), apilabel.AddRequest{
				ID:     r.ResourceID(),
				Labels: []label.Key{l.Key},
			})).Error().To(denied(u, access.ActionUpdate, r.ResourceID()))
			Expect(rangeLabels(ctx, r.Key)).To(BeEmpty())
		},
	)

	It(
		"Should refuse an add without retrieve on the label",
		func(ctx SpecContext) {
			u := createUser(ctx)
			r, l := createRange(ctx), createLabel(ctx)
			grant(
				ctx,
				access.ActionUpdate,
				u.ResourceID(),
				r.ResourceID(),
				l.ResourceID(),
			)
			Expect(add(AuthedCtx(ctx, u), apilabel.AddRequest{
				ID:     r.ResourceID(),
				Labels: []label.Key{l.Key},
			})).Error().To(denied(u, access.ActionRetrieve, l.ResourceID()))
			Expect(rangeLabels(ctx, r.Key)).To(BeEmpty())
		},
	)

	It(
		"Should refuse a remove without update on the range",
		func(ctx SpecContext) {
			u := createUser(ctx)
			l := createLabel(ctx)
			r := createRange(ctx, l.Key)
			grant(ctx, access.ActionUpdate, u.ResourceID(), l.ResourceID())
			Expect(remove(AuthedCtx(ctx, u), apilabel.RemoveRequest{
				ID:     r.ResourceID(),
				Labels: []label.Key{l.Key},
			})).Error().To(denied(u, access.ActionUpdate, r.ResourceID()))
			Expect(rangeLabels(ctx, r.Key)).To(Equal([]label.Key{l.Key}))
		},
	)

	It(
		"Should remove a label with update on the range and no access to the label",
		func(ctx SpecContext) {
			u := createUser(ctx)
			l := createLabel(ctx)
			r := createRange(ctx, l.Key)
			grant(ctx, access.ActionUpdate, u.ResourceID(), r.ResourceID())
			MustSucceed(remove(AuthedCtx(ctx, u), apilabel.RemoveRequest{
				ID:     r.ResourceID(),
				Labels: []label.Key{l.Key},
			}))
			Expect(rangeLabels(ctx, r.Key)).To(BeEmpty())
		},
	)

	It("Should reject a resource that cannot have labels", func(ctx SpecContext) {
		a := createLabel(ctx)
		id := resource.ID{Type: resource.TypeChannel, Key: "1"}
		actx := authed(ctx, resource.TypeChannel)
		Expect(add(actx, apilabel.AddRequest{
			ID:     id,
			Labels: []label.Key{a.Key},
		})).Error().To(MatchError(ContainSubstring("id: a channel cannot have labels")))
	})
})

var _ = Describe("Retrieve", func() {
	It("Should leave out the labels that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		granted := createLabel(ctx)
		createLabel(ctx)
		grant(ctx, access.ActionRetrieve, u.ResourceID(), granted.ResourceID())
		res := MustSucceed(retrieve(AuthedCtx(ctx, u), apilabel.RetrieveRequest{}))
		Expect(res.Labels).To(HaveExactElements(HaveField("Key", granted.Key)))
	})
})
