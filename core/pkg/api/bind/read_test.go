// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bind_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
)

// labelQuery selects labels by key without naming them.
type labelQuery struct {
	keys  []label.Key
	limit int
}

// namedQuery selects labels by key and names them.
type namedQuery struct{ labelQuery }

var _ bind.Named = namedQuery{}

func (q namedQuery) Named() []resource.ID {
	return lo.Map(q.keys, func(k label.Key, _ int) resource.ID {
		return label.ResourceID(k)
	})
}

func labelKeys(labels []label.Label) []label.Key {
	return lo.Map(labels, func(l label.Label, _ int) label.Key { return l.Key })
}

// retrieveLabels is a retrieve route that keeps the labels the subject can retrieve.
func retrieveLabels(
	ctx context.Context,
	snap bind.Snapshot,
	q labelQuery,
) ([]label.Label, error) {
	var res []label.Label
	return res, svc.Label.NewRetrieve().
		Where(label.MatchKeys(q.keys...)).
		Where(bind.Filter(snap, label.Match)).
		Limit(q.limit).
		Entries(&res).
		Exec(ctx, snap)
}

// createLabels creates n labels whose keys a table scan visits in the order created.
func createLabels(ctx context.Context, n int) []label.Label {
	GinkgoHelper()
	base := uuid.New()
	labels := make([]label.Label, n)
	for i := range labels {
		labels[i] = label.Label{Key: base, Name: "label"}
		labels[i].Key[15] = byte(i)
	}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return svc.Label.NewWriter(tx).CreateMany(ctx, &labels)
	})).To(Succeed())
	return labels
}

var _ = Describe("Read", func() {
	var (
		read  func(context.Context, labelQuery) ([]label.Label, error)
		named func(context.Context, namedQuery) ([]label.Label, error)
	)
	BeforeEach(func() {
		read = bind.Read(binder, retrieveLabels)
		named = bind.Read(
			binder,
			func(
				ctx context.Context,
				snap bind.Snapshot,
				q namedQuery,
			) ([]label.Label, error) {
				return retrieveLabels(ctx, snap, q.labelQuery)
			},
		)
	})

	It("Should return every match to a subject that can retrieve the type", func(
		ctx SpecContext,
	) {
		labels := createLabels(ctx, 3)
		u := newUser(ctx)
		grant(ctx, u, access.ActionRetrieve, resource.ID{Type: resource.TypeLabel})
		res := MustSucceed(read(AuthedCtx(ctx, u), labelQuery{keys: labelKeys(labels)}))
		Expect(labelKeys(res)).To(Equal(labelKeys(labels)))
	})

	It("Should leave out the matches that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		labels := createLabels(ctx, 3)
		u := newUser(ctx)
		grant(ctx, u, access.ActionRetrieve, labels[1].ResourceID())
		res := MustSucceed(read(AuthedCtx(ctx, u), labelQuery{keys: labelKeys(labels)}))
		Expect(labelKeys(res)).To(Equal([]label.Key{labels[1].Key}))
	})

	It("Should fill the limit with the matches that the subject can retrieve", func(
		ctx SpecContext,
	) {
		labels := createLabels(ctx, 4)
		u := newUser(ctx)
		grant(
			ctx,
			u,
			access.ActionRetrieve,
			labels[2].ResourceID(),
			labels[3].ResourceID(),
		)
		res := MustSucceed(read(
			AuthedCtx(ctx, u),
			labelQuery{keys: labelKeys(labels), limit: 2},
		))
		Expect(labelKeys(res)).To(Equal(labelKeys(labels[2:])))
	})

	It("Should return the named records that the subject can retrieve", func(
		ctx SpecContext,
	) {
		labels := createLabels(ctx, 2)
		u := newUser(ctx)
		grant(
			ctx,
			u,
			access.ActionRetrieve,
			labels[0].ResourceID(),
			labels[1].ResourceID(),
		)
		res := MustSucceed(named(
			AuthedCtx(ctx, u),
			namedQuery{labelQuery{keys: labelKeys(labels)}},
		))
		Expect(labelKeys(res)).To(Equal(labelKeys(labels)))
	})

	It("Should deny a named record that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		labels := createLabels(ctx, 2)
		u := newUser(ctx)
		grant(ctx, u, access.ActionRetrieve, labels[0].ResourceID())
		Expect(named(
			AuthedCtx(ctx, u),
			namedQuery{labelQuery{keys: labelKeys(labels)}},
		)).Error().To(denied(u, access.ActionRetrieve, labels[1].ResourceID()))
	})

	It("Should deny a named key with no record that the subject cannot retrieve", func(
		ctx SpecContext,
	) {
		u := newUser(ctx)
		missing := uuid.New()
		Expect(named(
			AuthedCtx(ctx, u),
			namedQuery{labelQuery{keys: []label.Key{missing}}},
		)).Error().To(denied(u, access.ActionRetrieve, label.ResourceID(missing)))
	})

	It("Should return a zero response when the retrieve fails", func(ctx SpecContext) {
		fail := errors.New("retrieve failed")
		u := newUser(ctx)
		Expect(bind.Read(
			binder,
			func(context.Context, bind.Snapshot, labelQuery) ([]label.Label, error) {
				return []label.Label{{Name: "partial"}}, fail
			},
		)(AuthedCtx(ctx, u), labelQuery{})).Error().To(MatchError(fail))
	})
})

var _ = Describe("Snapshot", func() {
	It("Should report whether the subject can retrieve a record", func(
		ctx SpecContext,
	) {
		labels := createLabels(ctx, 2)
		u := newUser(ctx)
		grant(ctx, u, access.ActionRetrieve, labels[0].ResourceID())
		var allowed []bool
		MustSucceed(bind.Read(
			binder,
			func(
				_ context.Context,
				snap bind.Snapshot,
				_ labelQuery,
			) (struct{}, error) {
				allowed = []bool{
					snap.Allowed(labels[0].ResourceID()),
					snap.Allowed(labels[1].ResourceID()),
				}
				return struct{}{}, nil
			},
		)(
			AuthedCtx(ctx, u),
			labelQuery{},
		))
		Expect(allowed).To(Equal([]bool{true, false}))
	})
})

var _ = Describe("Keys", func() {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	DescribeTable("Should return the resource ID of each key the set selects", func(
		set *filter.Set[label.Key],
		expected []label.Key,
	) {
		Expect(bind.Keys(set, label.ResourceID)).To(Equal(
			lo.Map(expected, func(k label.Key, _ int) resource.ID {
				return label.ResourceID(k)
			}),
		))
	},
		Entry("an empty set", &filter.Set[label.Key]{}, []label.Key{}),
		Entry("eq", &filter.Set[label.Key]{Eq: &a}, []label.Key{a}),
		Entry("in", &filter.Set[label.Key]{In: []label.Key{b, c}}, []label.Key{b, c}),
		Entry(
			"eq and in",
			&filter.Set[label.Key]{Eq: &a, In: []label.Key{b, c}},
			[]label.Key{a, b, c},
		),
	)

	It("Should return no IDs for a nil set", func() {
		Expect(bind.Keys(nil, label.ResourceID)).To(BeNil())
	})
})
