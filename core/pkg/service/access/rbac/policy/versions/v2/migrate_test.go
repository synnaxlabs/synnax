// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"slices"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	v2 "github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy/versions/v2"
	role "github.com/synnaxlabs/synnax/pkg/service/access/rbac/role/versions/v0"
	ontology "github.com/synnaxlabs/synnax/pkg/service/ontology/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("RoleMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	createRole := func(ctx SpecContext) role.Key {
		GinkgoHelper()
		r := role.Role{Key: uuid.New(), Name: "role"}
		Expect(gorp.NewCreate[role.Key, role.Role]().Entry(&r).Exec(ctx, db)).
			To(Succeed())
		return r.Key
	}
	createPolicy := func(ctx SpecContext, parents ...role.Key) v2.Policy {
		GinkgoHelper()
		p := v2.Policy{
			Key:     uuid.New(),
			Name:    "policy",
			Objects: []resource.ID{{Type: resource.TypeChannel}},
			Actions: []access.Action{access.ActionRetrieve},
		}
		Expect(gorp.NewCreate[v2.Key, v2.Policy]().Entry(&p).Exec(ctx, db)).
			To(Succeed())
		edges := make([]ontology.Relationship, len(parents))
		for i, r := range parents {
			edges[i] = ontology.Relationship{
				From: resource.ID{Type: resource.TypeRole, Key: r.String()},
				To:   resource.ID{Type: resource.TypePolicy, Key: p.Key.String()},
				Type: ontology.RelationshipTypeParentOf,
			}
		}
		Expect(gorp.WrapWriter[string, ontology.Relationship](db).
			Set(ctx, edges...)).To(Succeed())
		return p
	}
	migrate := func(ctx SpecContext) []v2.Policy {
		GinkgoHelper()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Policy",
			Migrations: []gorp.Migration{v2.RoleMigration},
		})).To(Succeed())
		var edges []ontology.Relationship
		Expect(gorp.NewRetrieve[string, ontology.Relationship]().
			Entries(&edges).
			Exec(ctx, db)).To(Succeed())
		Expect(edges).To(BeEmpty())
		var policies []v2.Policy
		Expect(gorp.NewRetrieve[v2.Key, v2.Policy]().
			Entries(&policies).
			Exec(ctx, db)).To(Succeed())
		return policies
	}

	It("Should set the role of a policy from its parent edge", func(ctx SpecContext) {
		r := createRole(ctx)
		p := createPolicy(ctx, r)
		p.Role = r
		Expect(migrate(ctx)).To(Equal([]v2.Policy{p}))
	})

	It("Should copy a policy under each role after the first", func(ctx SpecContext) {
		roles := []role.Key{createRole(ctx), createRole(ctx)}
		slices.SortFunc(roles, func(a, b role.Key) int { return a.Compare(b) })
		p := createPolicy(ctx, roles[1], roles[0])
		first, copied := p, p
		first.Role = roles[0]
		copied.Key, copied.Role = v2.CopyKey(p.Key, roles[1]), roles[1]
		Expect(migrate(ctx)).To(ConsistOf(first, copied))
	})

	It("Should delete a policy that no stored role holds", func(ctx SpecContext) {
		createPolicy(ctx)
		createPolicy(ctx, uuid.New())
		Expect(migrate(ctx)).To(BeEmpty())
	})
})

var _ = Describe("CopyKey", func() {
	It("Should derive one key for each policy and role", func() {
		p, r := uuid.New(), uuid.New()
		Expect(v2.CopyKey(p, r)).To(Equal(v2.CopyKey(p, r)))
		Expect(v2.CopyKey(p, r)).ToNot(Equal(v2.CopyKey(p, uuid.New())))
		Expect(v2.CopyKey(p, r)).ToNot(Equal(p))
	})
})
