// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/resource/rename"
	versions "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NameRuleMigration", func() {
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	// keys are in ascending order, so a group earlier in a list comes first by key.
	keys := []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
	}
	folder := func(i int, name string, parent *versions.ID) v1.Group {
		return v1.Group{
			Key:    keys[i],
			Name:   name,
			Scope:  resource.TypeChannel,
			Parent: parent,
		}
	}
	names := func(ctx SpecContext, groups ...v1.Group) []string {
		GinkgoHelper()
		Expect(gorp.NewCreate[v1.Key, v1.Group]().
			Entries(&groups).
			Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "group",
			Migrations: []gorp.Migration{v1.NameRuleMigration},
		})).To(Succeed())
		out := make([]string, len(groups))
		for i, g := range groups {
			var stored v1.Group
			Expect(gorp.NewRetrieve[v1.Key, v1.Group]().
				Where(gorp.MatchKeys[v1.Key, v1.Group](g.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			out[i] = stored.Name
		}
		return out
	}

	It("Should never move a legal name", func(ctx SpecContext) {
		Expect(names(ctx, folder(0, "Stand A", nil), folder(1, "PT-01", nil))).
			To(Equal([]string{"Stand A", "PT-01"}))
		Expect(rename.Staged(db)).To(BeEmpty())
	})

	It("Should rewrite each name that breaks the rule", func(ctx SpecContext) {
		Expect(names(
			ctx,
			folder(0, "a.b", nil),
			folder(1, "\u200B", nil),
			folder(2, "e\u0301", nil),
		)).To(Equal([]string{"a_b", "folder", "\u00E9"}))
	})

	It("Should keep a name for the first sibling by key", func(ctx SpecContext) {
		parent := versions.ID{Type: "group", Key: uuid.New().String()}
		Expect(names(
			ctx,
			folder(0, "pt", nil),
			folder(1, "pt", nil),
			folder(2, "pt", &parent),
		)).To(Equal([]string{"pt", "pt_2", "pt"}))
	})

	It("Should leave groups of other scopes alone", func(ctx SpecContext) {
		other := folder(0, "a.b", nil)
		other.Scope = resource.TypeProject
		Expect(names(ctx, other)).To(Equal([]string{"a.b"}))
	})

	It("Should stage each rename", func(ctx SpecContext) {
		names(ctx, folder(0, "a.b", nil))
		Expect(rename.Staged(db)).To(Equal([]rename.Rename{{
			Resource: resource.ID{Type: resource.TypeGroup, Key: keys[0].String()},
			From:     "a.b",
			To:       "a_b",
		}}))
	})
})
