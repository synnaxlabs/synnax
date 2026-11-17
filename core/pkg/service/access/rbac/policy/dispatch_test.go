// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

func notWhole(path string) OmegaMatcher {
	return SatisfyAll(
		MatchError(validate.ErrValidation),
		MatchError(ContainSubstring(
			path+": a policy takes only a set of its whole name, objects, actions, "+
				"or role",
		)),
	)
}

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[policy.Key, policy.Action]
	var (
		editable role.Role
		p        policy.Policy
	)
	createRole := func(ctx context.Context, r *role.Role) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return roleSvc.NewWriter(tx, true).Create(ctx, r)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return roleSvc.NewWriter(tx, true).Delete(ctx, r.Key)
			})).To(Succeed())
		})
	}
	create := func(ctx context.Context, p *policy.Policy) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx, true).Create(ctx, p)
		})).To(Succeed())
	}
	BeforeEach(func(ctx SpecContext) {
		editable = role.Role{Name: "editable"}
		createRole(ctx, &editable)
		p = policy.Policy{
			Name:    "read channels",
			Objects: []resource.ID{{Type: resource.TypeChannel}},
			Actions: []access.Action{access.ActionRetrieve},
			Role:    editable.Key,
		}
		create(ctx, &p)
	})
	dispatch := func(ctx context.Context, key policy.Key, acts ...policy.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx, false).
				WithDispatchKey("dk").
				Apply(ctx, key, acts...)
		})
	}
	record := func() *Recorder[policy.Key, policy.Action] {
		rec := &Recorder[policy.Key, policy.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	set := func(path string, value any) policy.Action {
		return policy.NewSetAction(
			policy.SetPayload{Path: []string{path}, Value: value},
		)
	}
	retrieve := func(ctx context.Context, key policy.Key) policy.Policy {
		GinkgoHelper()
		var res policy.Policy
		Expect(
			svc.NewRetrieve().Where(policy.MatchKeys(key)).Entry(&res).Exec(ctx, db),
		).
			To(Succeed())
		return res
	}

	It("Should store the edited name, objects, and actions", func(ctx SpecContext) {
		objects := []resource.ID{{Type: resource.TypeRange}, {Type: resource.TypeTask}}
		acts := []policy.Action{
			set("name", "edit ranges"),
			set("objects", objects),
			set("actions", []access.Action{access.ActionRetrieve, access.ActionUpdate}),
		}
		rec := record()
		Expect(dispatch(ctx, p.Key, acts...)).To(Succeed())
		Expect(retrieve(ctx, p.Key)).To(Equal(policy.Policy{
			Key:     p.Key,
			Name:    "edit ranges",
			Objects: objects,
			Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate},
			Role:    editable.Key,
		}))
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{Key: p.Key, DispatchKey: "dk", Seq: frames[0].Seq, Actions: acts},
		}))
	})

	DescribeTable(
		"Should refuse an action and store nothing",
		func(
			ctx SpecContext,
			act policy.Action,
			matcher OmegaMatcher,
		) {
			rec := record()
			Expect(dispatch(ctx, p.Key, set("name", "edit ranges"), act)).To(matcher)
			Expect(retrieve(ctx, p.Key)).To(Equal(p))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry(
			"a path outside the editable fields",
			set("internal", true),
			MatchError(validate.ErrValidation),
		),
		Entry(
			"a role that does not exist",
			set("role", uuid.New()),
			MatchError(ContainSubstring("does not exist")),
		),
		Entry(
			"an empty name",
			set("name", ""),
			MatchError(ContainSubstring("required")),
		),
		Entry(
			"a create",
			policy.NewCreateAction(policy.CreatePayload{}),
			MatchError(ContainSubstring("only the Core dispatches a policy create")),
		),
		Entry(
			"a set inside the objects",
			policy.NewSetAction(policy.SetPayload{
				Path:  []string{"objects", "", "type"},
				Value: resource.TypeRange,
			}),
			notWhole("[objects, , type]"),
		),
		Entry(
			"a remove from the objects",
			policy.NewRemoveAction(policy.RemovePayload{Path: []string{"objects"}}),
			notWhole("[objects, ]"),
		),
		Entry(
			"a remove from the actions",
			policy.NewRemoveAction(policy.RemovePayload{
				Path: []string{"actions"},
				Key:  string(access.ActionRetrieve),
			}),
			notWhole("[actions, retrieve]"),
		),
		Entry(
			"an insert into the objects",
			policy.NewInsertAction(policy.InsertPayload{
				Path:  []string{"objects"},
				Value: resource.ID{Type: resource.TypeTask, Key: "5"},
			}),
			notWhole("[objects]"),
		),
		Entry(
			"a move in the actions",
			policy.NewMoveAction(policy.MovePayload{
				Path: []string{"actions"},
				Key:  string(access.ActionRetrieve),
			}),
			notWhole("[actions]"),
		),
	)

	It("Should move the policy to another role", func(ctx SpecContext) {
		other := role.Role{Name: "other"}
		createRole(ctx, &other)
		rec := record()
		act := set("role", other.Key)
		Expect(dispatch(ctx, p.Key, act)).To(Succeed())
		Expect(retrieve(ctx, p.Key).Role).To(Equal(other.Key))
		Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
			"Actions",
			Equal([]policy.Action{act}),
		)))
	})

	It("Should refuse to move a policy to a built-in role", func(ctx SpecContext) {
		internal := role.Role{Name: "internal", Internal: true}
		createRole(ctx, &internal)
		Expect(dispatch(ctx, p.Key, set("role", internal.Key))).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot change the policies of built-in role")),
		))
		Expect(retrieve(ctx, p.Key)).To(Equal(p))
	})

	It("Should refuse to change a policy of a built-in role", func(ctx SpecContext) {
		internal := role.Role{Name: "internal", Internal: true}
		createRole(ctx, &internal)
		builtin := policy.Policy{Name: "builtin", Role: internal.Key}
		create(ctx, &builtin)
		Expect(dispatch(ctx, builtin.Key, set("name", "renamed"))).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot change the policies of built-in role")),
		))
	})

	It("Should return not found for a missing policy", func(ctx SpecContext) {
		Expect(dispatch(ctx, uuid.New(), set("name", "edit ranges"))).
			To(MatchError(query.ErrNotFound))
	})

	It("Should broadcast a create", func(ctx SpecContext) {
		rec := record()
		created := policy.Policy{Name: "created", Role: editable.Key}
		create(ctx, &created)
		Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
			"Actions",
			Equal([]policy.Action{
				policy.NewCreateAction(policy.CreatePayload{Policy: created}),
			}),
		)))
	})
})
