// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[user.Key, user.Action]
	var (
		u     user.User
		taken string
	)
	BeforeEach(func(ctx SpecContext) {
		u = createUser(ctx, uuid.New().String(), "p", false)
		taken = uuid.New().String()
		createUser(ctx, taken, "p", false)
	})
	dispatch := func(ctx context.Context, key user.Key, acts ...user.Action) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).WithDispatchKey("dk").Apply(ctx, key, acts...)
		})
	}
	record := func() *Recorder[user.Key, user.Action] {
		rec := &Recorder[user.Key, user.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	set := func(path string, value any) user.Action {
		return user.NewSetAction(user.SetPayload{Path: []string{path}, Value: value})
	}
	setUsername := func(username string) user.Action {
		return user.NewSetUsernameAction(user.SetUsernamePayload{Username: username})
	}
	retrieve := func(ctx context.Context, key user.Key) user.User {
		GinkgoHelper()
		var res user.User
		Expect(svc.NewRetrieve().Where(user.MatchKeys(key)).Entry(&res).Exec(ctx, db)).
			To(Succeed())
		return res
	}

	It("Should store the edited name and role", func(ctx SpecContext) {
		Expect(dispatch(ctx, u.Key, set("name", "b"), set("role", owner))).To(Succeed())
		Expect(retrieve(ctx, u.Key)).
			To(Equal(user.User{Key: u.Key, Name: "b", Role: owner}))
	})

	It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
		rec := record()
		acts := []user.Action{set("name", "b")}
		Expect(dispatch(ctx, u.Key, acts...)).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(Equal([]frame{
			{Key: u.Key, DispatchKey: "dk", Seq: frames[0].Seq, Actions: acts},
		}))
	})

	DescribeTable("Should refuse an action and store nothing", func(
		ctx SpecContext,
		act func() user.Action,
		matcher OmegaMatcher,
	) {
		rec := record()
		Expect(dispatch(ctx, u.Key, set("name", "b"), act())).To(matcher)
		Expect(retrieve(ctx, u.Key)).To(Equal(u))
		Expect(rec.Snapshot()).To(BeEmpty())
	},
		Entry("a path outside the editable fields", func() user.Action {
			return set("root_user", true)
		}, MatchError(validate.ErrValidation)),
		Entry("an empty name", func() user.Action {
			return set("name", "")
		}, MatchError(ContainSubstring("name: required"))),
		Entry("a role that does not exist", func() user.Action {
			return set("role", uuid.New())
		}, MatchError(MatchRegexp(`role: role \S+ does not exist`))),
		Entry("a create", func() user.Action {
			return user.NewCreateAction(user.CreatePayload{User: u})
		}, MatchError(ContainSubstring("only the Core dispatches a user create"))),
		Entry("a username another user holds", func() user.Action {
			return setUsername(taken)
		}, MatchError(auth.ErrRepeatedUsername)),
	)

	It("Should refuse to change the role of a root user", func(ctx SpecContext) {
		r := createUser(ctx, uuid.New().String(), "p", true)
		Expect(dispatch(ctx, r.Key, set("role", member))).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot change the role of root user")),
		))
		Expect(retrieve(ctx, r.Key).Role).To(Equal(owner))
	})

	It("Should let a root user change its name", func(ctx SpecContext) {
		r := createUser(ctx, uuid.New().String(), "p", true)
		Expect(dispatch(ctx, r.Key, set("name", "renamed root"))).To(Succeed())
		Expect(retrieve(ctx, r.Key).Name).To(Equal("renamed root"))
	})

	It("Should rename the password of the user", func(ctx SpecContext) {
		username := uuid.New().String()
		Expect(dispatch(ctx, u.Key, setUsername(username))).To(Succeed())
		var p password.Password
		Expect(passwordSvc.NewRetrieve().
			Where(password.MatchSubjects(u.ResourceID())).
			Entry(&p).
			Exec(ctx, db)).To(Succeed())
		Expect(p.Username).To(Equal(username))
	})

	It("Should return not found for a missing user", func(ctx SpecContext) {
		Expect(dispatch(ctx, uuid.New(), set("name", "b"))).
			To(MatchError(query.ErrNotFound))
	})

	Describe("Writer", func() {
		It("Should broadcast a create", func(ctx SpecContext) {
			rec := record()
			var created user.User
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				created, err = svc.NewWriter(tx).
					Create(ctx, user.User{Name: "c", Role: member})
				return err
			})).To(Succeed())
			Expect(rec.Snapshot()).To(HaveExactElements(HaveField(
				"Actions",
				Equal([]user.Action{
					user.NewCreateAction(user.CreatePayload{User: created}),
				}),
			)))
		})

		It("Should broadcast a set of the role for each user", func(ctx SpecContext) {
			rec := record()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).SetRole(ctx, owner, u.Key)
			})).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(Equal([]frame{
				{
					Key:     u.Key,
					Seq:     frames[0].Seq,
					Actions: []user.Action{set("role", owner)},
				},
			}))
		})
	})
})
