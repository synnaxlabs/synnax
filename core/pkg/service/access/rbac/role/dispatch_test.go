// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[role.Key, role.Action]
	var r role.Role
	create := func(ctx context.Context, r *role.Role) {
		GinkgoHelper()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx, true).Create(ctx, r)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx, true).Delete(ctx, r.Key)
			})).To(Succeed())
		})
	}
	BeforeEach(func(ctx SpecContext) {
		r = role.Role{Name: "engineer"}
		create(ctx, &r)
	})
	dispatch := func(ctx context.Context, key role.Key, acts ...role.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx, false).
				WithDispatchKey("dk").
				Apply(ctx, key, acts...)
		})
	}
	record := func() *Recorder[role.Key, role.Action] {
		rec := &Recorder[role.Key, role.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	set := func(path string, value any) role.Action {
		return role.NewSetAction(role.SetPayload{Path: []string{path}, Value: value})
	}
	retrieve := func(ctx context.Context) role.Role {
		GinkgoHelper()
		var res role.Role
		Expect(svc.NewRetrieve().
			Where(role.MatchKeys(r.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should store the edited name and description", func(ctx SpecContext) {
		acts := []role.Action{set("name", "operator"), set("description", "runs tests")}
		rec := record()
		Expect(dispatch(ctx, r.Key, acts...)).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(role.Role{
			Key:         r.Key,
			Name:        "operator",
			Description: "runs tests",
		}))
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{Key: r.Key, DispatchKey: "dk", Seq: frames[0].Seq, Actions: acts},
		}))
	})

	DescribeTable(
		"Should refuse an action and store nothing",
		func(
			ctx SpecContext,
			act role.Action,
			matcher OmegaMatcher,
		) {
			rec := record()
			Expect(dispatch(ctx, r.Key, set("name", "operator"), act)).To(matcher)
			Expect(retrieve(ctx)).To(Equal(r))
			Expect(rec.Snapshot()).To(BeEmpty())
		},
		Entry(
			"a path outside the editable fields",
			set("internal", true),
			MatchError(validate.ErrValidation),
		),
		Entry(
			"an empty name",
			set("name", ""),
			MatchError(ContainSubstring("required")),
		),
		Entry(
			"a create",
			role.NewCreateAction(role.CreatePayload{}),
			MatchError(actions.ErrCoreOnly),
		),
	)

	It("Should refuse to change a built-in role", func(ctx SpecContext) {
		builtin := role.Role{Name: "builtin", Internal: true}
		create(ctx, &builtin)
		Expect(dispatch(ctx, builtin.Key, set("name", "renamed"))).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot change built-in role builtin")),
		))
	})

	It("Should return not found for a missing role", func(ctx SpecContext) {
		Expect(dispatch(ctx, uuid.New(), set("name", "operator"))).
			To(MatchError(query.ErrNotFound))
	})

	It("Should broadcast a create", func(ctx SpecContext) {
		rec := record()
		created := role.Role{Name: "viewer"}
		create(ctx, &created)
		Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
			"Actions",
			Equal([]role.Action{
				role.NewCreateAction(role.CreatePayload{Role: created}),
			}),
		)))
	})
})
