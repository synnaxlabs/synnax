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
	"sync"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/search"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		tx gorp.Tx
		w  policy.Writer
	)
	BeforeEach(func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx, false)
	})

	Describe("Create", func() {
		It("Should create a policy with auto-generated UUID", func(ctx SpecContext) {
			p := &policy.Policy{
				Role:    owner.Key,
				Name:    "test-policy",
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil()))
		})

		It(
			"Should return a validation error when an object has an invalid type",
			func(ctx SpecContext) {
				p := &policy.Policy{
					Role:    owner.Key,
					Name:    "test-policy",
					Objects: []resource.ID{{Type: "not-a-real-type", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(w.Create(ctx, p)).
					To(MatchError(ContainSubstring("objects.0.type: invalid type")))
			},
		)

		It("Should create a policy with provided UUID", func(ctx SpecContext) {
			key := uuid.New()
			p := &policy.Policy{
				Role:    owner.Key,
				Key:     key,
				Name:    "test-policy",
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).To(Equal(key))
		})

		It("Should refuse a policy whose key exists", func(ctx SpecContext) {
			p := &policy.Policy{
				Role:    owner.Key,
				Name:    "test-policy",
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			again := *p
			again.Name = "renamed"
			Expect(w.Create(ctx, &again)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("policy %s already exists", p.Key)),
			))
		})

		It("Should create a policy with multiple objects", func(ctx SpecContext) {
			p := &policy.Policy{
				Role: owner.Key,
				Name: "multi-object-policy",
				Objects: []resource.ID{
					{Type: "channel", Key: "ch1"},
					{Type: "channel", Key: "ch2"},
					{Type: "project", Key: "p1"},
				},
				Actions: []access.Action{access.ActionRetrieve, access.ActionUpdate},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
			Expect(p.Key).ToNot(Equal(uuid.Nil()))
		})

		It("Should create a policy with ActionAll wildcard", func(ctx SpecContext) {
			p := &policy.Policy{
				Role:    owner.Key,
				Name:    "wildcard-policy",
				Objects: []resource.ID{{Type: "channel"}},
				Actions: access.AllActions,
			}
			Expect(w.Create(ctx, p)).To(Succeed())
		})

		It(
			"Should create an internal policy when allowInternal is true",
			func(ctx SpecContext) {
				internalWriter := svc.NewWriter(tx, true)
				p := &policy.Policy{
					Role:     owner.Key,
					Name:     "internal-policy",
					Objects:  []resource.ID{{Type: "channel", Key: "ch1"}},
					Actions:  []access.Action{access.ActionRetrieve},
					Internal: true,
				}
				Expect(internalWriter.Create(ctx, p)).To(Succeed())
				Expect(p.Key).ToNot(Equal(uuid.Nil()))
			},
		)

		It(
			"Should fail to create an internal policy when allowInternal is false",
			func(ctx SpecContext) {
				p := &policy.Policy{
					Role:     owner.Key,
					Name:     "internal-policy",
					Objects:  []resource.ID{{Type: "channel", Key: "ch1"}},
					Actions:  []access.Action{access.ActionRetrieve},
					Internal: true,
				}
				Expect(w.Create(ctx, p)).
					Error().
					To(MatchError(ContainSubstring("cannot create internal policy")))
			},
		)
	})

	Describe("CreateMany", func() {
		It("Should create multiple policies", func(ctx SpecContext) {
			policies := []policy.Policy{
				{
					Role:    owner.Key,
					Name:    "policy-1",
					Key:     uuid.New(),
					Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
				{
					Role:    owner.Key,
					Name:    "policy-2",
					Key:     uuid.New(),
					Objects: []resource.ID{{Type: "project", Key: "proj1"}},
					Actions: []access.Action{access.ActionUpdate},
				},
			}
			Expect(w.CreateMany(ctx, &policies)).To(Succeed())

			var retrieved []policy.Policy
			Expect(svc.NewRetrieve().Where(policy.MatchKeys(
				policies[0].Key,
				policies[1].Key,
			)).Entries(&retrieved).Exec(ctx, tx)).To(Succeed())
			Expect(retrieved).To(HaveLen(2))
		})
	})

	Describe("Delete", func() {
		var policies []policy.Policy
		BeforeEach(func(ctx SpecContext) {
			policies = []policy.Policy{
				{
					Role:    owner.Key,
					Name:    "policy-1",
					Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				},
				{
					Role:    owner.Key,
					Name:    "policy-2",
					Objects: []resource.ID{{Type: "project", Key: "p1"}},
					Actions: []access.Action{access.ActionUpdate},
				},
			}
			for i := range policies {
				Expect(w.Create(ctx, &policies[i])).To(Succeed())
			}
		})

		It("Should delete a single policy", func(ctx SpecContext) {
			Expect(w.Delete(ctx, policies[0].Key)).To(Succeed())

			var p policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchKeys(policies[0].Key)).
				Entry(&p).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})

		It("Should delete multiple policies", func(ctx SpecContext) {
			Expect(w.Delete(ctx, policies[0].Key, policies[1].Key)).To(Succeed())

			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchKeys(policies[0].Key, policies[1].Key)).
				Entries(&ps).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
			Expect(ps).To(BeEmpty())
		})
	})

	Describe("Role", func() {
		It(
			"Should return a validation error when the role does not exist",
			func(ctx SpecContext) {
				p := &policy.Policy{
					Role:    uuid.New(),
					Name:    "orphan",
					Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(w.Create(ctx, p)).
					To(MatchError(ContainSubstring("role " + p.Role.String() +
						" does not exist")))
			},
		)

		It("Should delete the policies of a deleted role", func(ctx SpecContext) {
			r := &role.Role{Name: "doomed"}
			Expect(roleSvc.NewWriter(tx, true).Create(ctx, r)).To(Succeed())
			doomed := &policy.Policy{
				Role:    r.Key,
				Name:    "doomed-policy",
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			kept := &policy.Policy{
				Role:    owner.Key,
				Name:    "kept-policy",
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, doomed)).To(Succeed())
			Expect(w.Create(ctx, kept)).To(Succeed())
			Expect(roleSvc.NewWriter(tx, true).Delete(ctx, r.Key)).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(policy.MatchKeys(doomed.Key)).
				Exists(ctx, tx)).To(BeFalse())
			Expect(svc.NewRetrieve().
				Where(policy.MatchKeys(kept.Key)).
				Exists(ctx, tx)).To(BeTrue())
		})
	})
})

var _ = Describe("Retriever", func() {
	var (
		tx       gorp.Tx
		w        policy.Writer
		policies []policy.Policy
	)
	BeforeEach(func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		w = svc.NewWriter(tx, false)
		policies = []policy.Policy{
			{
				Role:    owner.Key,
				Name:    "alpha-policy",
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			},
			{
				Role:    owner.Key,
				Name:    "beta-policy",
				Objects: []resource.ID{{Type: "project", Key: "p1"}},
				Actions: []access.Action{access.ActionDelete},
			},
			{
				Role:    owner.Key,
				Name:    "gamma-policy",
				Objects: []resource.ID{{Type: "user", Key: "u1"}},
				Actions: []access.Action{access.ActionUpdate},
			},
		}
		for i := range policies {
			Expect(w.Create(ctx, &policies[i])).To(Succeed())
		}
	})

	Describe("WhereKeys", func() {
		It("Should retrieve a single policy by key", func(ctx SpecContext) {
			var p policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchKeys(policies[0].Key)).
				Entry(&p).
				Exec(ctx, tx)).To(Succeed())
			Expect(p.Key).To(Equal(policies[0].Key))
			Expect(p.Name).To(Equal(policies[0].Name))
		})

		It("Should retrieve multiple policies by keys", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchKeys(policies[0].Key, policies[1].Key)).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
		})

		It("Should return empty when key not found", func(ctx SpecContext) {
			var p policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchKeys(uuid.New())).
				Entry(&p).
				Exec(ctx, tx)).To(MatchError(query.ErrNotFound))
		})
	})

	Describe("MatchNames", func() {
		It("Should retrieve a policy by name", func(ctx SpecContext) {
			var p policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchNames("alpha-policy")).
				Entry(&p).
				Exec(ctx, tx)).To(Succeed())
			Expect(p.Name).To(Equal("alpha-policy"))
		})

		It("Should retrieve multiple policies by names", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchNames("alpha-policy", "beta-policy")).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
		})
	})

	Describe("MatchRoles", func() {
		var other *role.Role
		BeforeEach(func(ctx SpecContext) {
			other = &role.Role{Name: "other"}
			Expect(roleSvc.NewWriter(tx, true).Create(ctx, other)).To(Succeed())
			p := &policy.Policy{
				Role:    other.Key,
				Name:    "other-policy",
				Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(w.Create(ctx, p)).To(Succeed())
		})

		It("Should retrieve the policies of a role", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchRoles(other.Key)).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveExactElements(HaveField("Name", "other-policy")))
		})

		It("Should retrieve the policies of several roles", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchRoles(owner.Key, other.Key)).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(len(policies) + 1))
		})
	})

	Describe("Limit", func() {
		It("Should limit results", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Limit(2).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(2))
		})
	})

	Describe("MatchInternal", func() {
		var internalPolicy, regularPolicy policy.Policy
		BeforeEach(func(ctx SpecContext) {
			internalWriter := svc.NewWriter(tx, true)
			internalPolicy = policy.Policy{
				Role:     owner.Key,
				Name:     "internal-policy",
				Objects:  []resource.ID{{Type: "channel", Key: "ch1"}},
				Actions:  []access.Action{access.ActionRetrieve},
				Internal: true,
			}
			regularPolicy = policy.Policy{
				Role:     owner.Key,
				Name:     "regular-policy",
				Objects:  []resource.ID{{Type: "channel", Key: "ch2"}},
				Actions:  []access.Action{access.ActionRetrieve},
				Internal: false,
			}
			Expect(internalWriter.Create(ctx, &internalPolicy)).To(Succeed())
			Expect(internalWriter.Create(ctx, &regularPolicy)).To(Succeed())
		})

		It("Should retrieve only internal policies", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchInternal(true)).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			for _, p := range ps {
				Expect(p.Internal).To(BeTrue())
			}
			Expect(ps).To(ContainElement(HaveField("Key", internalPolicy.Key)))
		})

		It("Should retrieve only non-internal policies", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().
				Where(policy.MatchInternal(false)).
				Entries(&ps).
				Exec(ctx, tx)).To(Succeed())
			for _, p := range ps {
				Expect(p.Internal).To(BeFalse())
			}
			Expect(ps).To(ContainElement(HaveField("Key", regularPolicy.Key)))
		})
	})

	Describe("Combinators", func() {
		var internalPolicy, regularPolicy policy.Policy
		BeforeEach(func(ctx SpecContext) {
			internalWriter := svc.NewWriter(tx, true)
			internalPolicy = policy.Policy{
				Role:     owner.Key,
				Name:     "combo-internal",
				Objects:  []resource.ID{{Type: "channel", Key: "cix"}},
				Actions:  []access.Action{access.ActionRetrieve},
				Internal: true,
			}
			regularPolicy = policy.Policy{
				Role:    owner.Key,
				Name:    "combo-regular",
				Objects: []resource.ID{{Type: "channel", Key: "crg"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(internalWriter.Create(ctx, &internalPolicy)).To(Succeed())
			Expect(internalWriter.Create(ctx, &regularPolicy)).To(Succeed())
		})
		It("Should compose filters with And", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().Where(policy.And(
				policy.MatchNames("combo-internal", "combo-regular"),
				policy.MatchInternal(true),
			)).Entries(&ps).Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(1))
			Expect(ps[0].Key).To(Equal(internalPolicy.Key))
		})
		It("Should compose filters with Or", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().Where(policy.Or(
				policy.MatchNames("combo-internal"),
				policy.MatchNames("combo-regular"),
			)).Entries(&ps).Exec(ctx, tx)).To(Succeed())
			names := make([]string, len(ps))
			for i, p := range ps {
				names[i] = p.Name
			}
			Expect(names).To(ContainElements("combo-internal", "combo-regular"))
		})
		It("Should invert a filter with Not", func(ctx SpecContext) {
			var ps []policy.Policy
			Expect(svc.NewRetrieve().Where(policy.And(
				policy.MatchNames("combo-internal", "combo-regular"),
				policy.Not(policy.MatchInternal(true)),
			)).Entries(&ps).Exec(ctx, tx)).To(Succeed())
			Expect(ps).To(HaveLen(1))
			Expect(ps[0].Key).To(Equal(regularPolicy.Key))
		})
	})

	Describe("Count and Exists", func() {
		It("Should count policies matching a filter", func(ctx SpecContext) {
			p1 := policy.Policy{
				Role:    owner.Key,
				Name:    "count-a",
				Objects: []resource.ID{{Type: "channel", Key: "a"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			p2 := policy.Policy{
				Role:    owner.Key,
				Name:    "count-b",
				Objects: []resource.ID{{Type: "channel", Key: "b"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(svc.NewWriter(tx, false).Create(ctx, &p1)).To(Succeed())
			Expect(svc.NewWriter(tx, false).Create(ctx, &p2)).To(Succeed())
			n := MustSucceed(svc.NewRetrieve().
				Where(policy.MatchNames("count-a", "count-b")).
				Count(ctx, tx))
			Expect(n).To(Equal(2))
		})
		It("Should report existence with Exists", func(ctx SpecContext) {
			p := policy.Policy{
				Role:    owner.Key,
				Name:    "exists-policy",
				Objects: []resource.ID{{Type: "channel", Key: "x"}},
				Actions: []access.Action{access.ActionRetrieve},
			}
			Expect(svc.NewWriter(tx, false).Create(ctx, &p)).To(Succeed())
			Expect(MustSucceed(svc.NewRetrieve().
				Where(policy.MatchNames("exists-policy")).
				Exists(ctx, tx))).To(BeTrue())
			Expect(MustSucceed(svc.NewRetrieve().
				Where(policy.MatchNames("nope-policy")).
				Exists(ctx, tx))).To(BeFalse())
		})
	})
})

var _ = Describe("Resources", func() {
	var tx gorp.Tx
	BeforeEach(func(ctx SpecContext) { tx = DeferClose(db.OpenTx()) })

	Describe("Search", func() {
		It(
			"Should return the policies whose name matches the term",
			func(ctx SpecContext) {
				p := &policy.Policy{
					Role:    owner.Key,
					Name:    "turbopump readers",
					Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(svc.NewWriter(tx, false).Create(ctx, p)).To(Succeed())
				q := search.Query[resource.ID]{Term: "turbopump", Limit: 10}
				Expect(MustSucceed(svc.Resources().Search(ctx, tx, q)).Keys).
					To(Equal([]resource.ID{p.ResourceID()}))
			},
		)
	})
})

var _ = Describe("Writer in concurrent transactions", func() {
	It(
		"Should not let a user edit undo a concurrent move to a built-in role",
		func(ctx SpecContext) {
			custom := role.Role{Name: "custom"}
			builtin := role.Role{Name: "built-in", Internal: true}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return roleSvc.NewWriter(tx, true).Create(ctx, &custom)
			})).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return roleSvc.NewWriter(tx, true).Create(ctx, &builtin)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return roleSvc.NewWriter(tx, true).Delete(ctx, custom.Key)
				})).To(Succeed())
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return roleSvc.NewWriter(tx, true).Delete(ctx, builtin.Key)
				})).To(Succeed())
			})
			setRole := func(r role.Key) policy.Action {
				return policy.NewSetAction(
					policy.SetPayload{Path: []string{"role"}, Value: r},
				)
			}
			for range 30 {
				p := policy.Policy{
					Role:    custom.Key,
					Name:    "moving",
					Objects: []resource.ID{{Type: "channel", Key: "ch1"}},
					Actions: []access.Action{access.ActionRetrieve},
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx, false).Create(ctx, &p)
				})).To(Succeed())
				var (
					wg      sync.WaitGroup
					userErr error
				)
				wg.Go(func() {
					defer GinkgoRecover()
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx, true).
							Apply(ctx, p.Key, setRole(builtin.Key))
					})).To(Succeed())
				})
				wg.Go(func() {
					userErr = db.WithTx(ctx, func(tx gorp.Tx) error {
						return svc.NewWriter(tx, false).
							Apply(ctx, p.Key, setRole(custom.Key))
					})
				})
				wg.Wait()
				Expect(userErr).To(Or(
					Succeed(),
					MatchError(
						ContainSubstring("cannot change the policies of built-in"),
					),
				))
				var stored policy.Policy
				Expect(svc.NewRetrieve().
					Where(policy.MatchKeys(p.Key)).
					Entry(&stored).
					Exec(ctx, db)).To(Succeed())
				Expect(stored.Role).To(Equal(builtin.Key))
			}
		},
	)
})

var _ = Describe("ResolveRolesWith", func() {
	var (
		tx                 gorp.Tx
		held, other        role.Role
		heldPolicy, otherP policy.Policy
		subject            resource.ID
		rolesOf            policy.RolesOf
	)
	BeforeEach(func(ctx SpecContext) {
		tx = DeferClose(db.OpenTx())
		held, other = role.Role{Name: "held"}, role.Role{Name: "other"}
		rw := roleSvc.NewWriter(tx, true)
		Expect(rw.Create(ctx, &held)).To(Succeed())
		Expect(rw.Create(ctx, &other)).To(Succeed())
		heldPolicy = policy.Policy{
			Role:    held.Key,
			Name:    "held-policy",
			Objects: []resource.ID{{Type: resource.TypeChannel}},
			Actions: []access.Action{access.ActionRetrieve},
		}
		otherP = policy.Policy{
			Role:    other.Key,
			Name:    "other-policy",
			Objects: []resource.ID{{Type: resource.TypeChannel}},
			Actions: []access.Action{access.ActionRetrieve},
		}
		w := svc.NewWriter(tx, true)
		Expect(w.Create(ctx, &heldPolicy)).To(Succeed())
		Expect(w.Create(ctx, &otherP)).To(Succeed())
		subject = resource.ID{Type: resource.TypeUser, Key: uuid.New().String()}
		rolesOf = func(
			_ context.Context,
			_ gorp.Snapshot,
			subjects ...resource.ID,
		) ([]role.Key, error) {
			if subjects[0] != subject {
				return nil, query.ErrNotFound
			}
			return []role.Key{held.Key}, nil
		}
	})

	retrieve := func(ctx SpecContext, f policy.Filter) ([]policy.Policy, error) {
		var ps []policy.Policy
		err := svc.NewRetrieve().ResolveRolesWith(rolesOf).Where(f).Entries(&ps).
			Exec(ctx, tx)
		return ps, err
	}

	It("Should match the policies of the roles a subject holds", func(ctx SpecContext) {
		Expect(retrieve(ctx, policy.MatchSubjects(subject))).To(ConsistOf(heldPolicy))
	})

	It("Should match no policies for a subject that does not exist", func(
		ctx SpecContext,
	) {
		missing := resource.ID{Type: resource.TypeUser, Key: uuid.New().String()}
		Expect(retrieve(ctx, policy.Or(
			policy.MatchSubjects(missing),
			policy.MatchKeys(otherP.Key),
		))).To(ConsistOf(otherP))
	})

	It("Should return any other error of the role lookup", func(ctx SpecContext) {
		rolesOf = func(
			context.Context,
			gorp.Snapshot,
			...resource.ID,
		) ([]role.Key, error) {
			return nil, errors.New("lookup failed")
		}
		Expect(retrieve(ctx, policy.MatchSubjects(subject))).Error().
			To(MatchError(ContainSubstring("lookup failed")))
	})

	It("Should look up roles through the snapshot of the query", func(
		ctx SpecContext,
	) {
		Expect(tx.Commit(ctx)).To(Succeed())
		var (
			looked gorp.Snapshot
			lookup = rolesOf
			ps     []policy.Policy
		)
		Expect(db.WithSnapshot(ctx, func(snap gorp.Snapshot) error {
			return svc.NewRetrieve().
				ResolveRolesWith(func(
					ctx context.Context,
					snap gorp.Snapshot,
					subjects ...resource.ID,
				) ([]role.Key, error) {
					looked = snap
					return lookup(ctx, snap, subjects...)
				}).
				Where(policy.MatchSubjects(subject)).
				Entries(&ps).
				Exec(ctx, snap)
		})).To(Succeed())
		Expect(looked).ToNot(BeNil())
		_, isTx := looked.(gorp.Tx)
		Expect(isTx).To(BeFalse())
		Expect(ps).To(ConsistOf(heldPolicy))
	})

	It("Should fail to filter by subject without a role lookup", func(ctx SpecContext) {
		var ps []policy.Policy
		Expect(svc.NewRetrieve().
			Where(policy.MatchSubjects(subject)).
			Entries(&ps).
			Exec(ctx, tx)).
			To(MatchError(ContainSubstring("cannot filter by subject")))
	})
})
