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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	resourcev0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NameRuleMigration", func() {
	var (
		db     *gorp.DB
		first  = uuid.MustParse("00000000-0000-4000-8000-000000000001")
		second = uuid.MustParse("00000000-0000-4000-8000-000000000002")
		third  = uuid.MustParse("00000000-0000-4000-8000-000000000003")
	)
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	folder := func(key uuid.UUID, name string, parent *resource.ID) v1.Group {
		return v1.Group{
			Key:    key,
			Name:   name,
			Scope:  resource.TypeChannel,
			Parent: parent,
		}
	}
	// migrate stores groups, runs the migration, and returns the name of each group by
	// key.
	migrate := func(ctx context.Context, groups ...v1.Group) map[uuid.UUID]string {
		GinkgoHelper()
		Expect(gorp.WrapWriter[v1.Key, v1.Group](db).Set(ctx, groups...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "group",
			Migrations: []gorp.Migration{v1.NameRuleMigration},
		})).To(Succeed())
		var stored []v1.Group
		Expect(gorp.NewRetrieve[v1.Key, v1.Group]().
			Entries(&stored).
			Exec(ctx, db)).To(Succeed())
		names := make(map[uuid.UUID]string, len(stored))
		for _, g := range stored {
			names[g.Key] = g.Name
		}
		return names
	}

	DescribeTable("Should rewrite a name that a path cannot hold and stage it",
		func(ctx SpecContext, name, expected string) {
			Expect(migrate(ctx, folder(first, name, nil))[first]).To(Equal(expected))
			Expect(MustSucceed(resourcev0.StagedRenames(db))).To(ConsistOf(
				resourcev0.Rename{
					ID:     groupID(first),
					From:   name,
					Reason: resourcev0.RenameReason(name),
				},
			))
		},
		Entry("a separator", "Stand.A", "Stand_A"),
		Entry("each invalid byte", "Stand\xffA", "Stand\uFFFDA"),
		Entry("an empty name", "", "folder"),
	)

	DescribeTable("Should keep a name that breaks the rule but not a path",
		func(ctx SpecContext, name string) {
			Expect(migrate(ctx, folder(first, name, nil))[first]).To(Equal(name))
			Expect(MustSucceed(resourcev0.StagedRenames(db))).To(BeEmpty())
		},
		Entry("a zero-width space", "Stand\u200BA"),
		Entry("a no-break space alone", "\u00A0"),
	)

	It("Should write a kept name in NFC with no staged rename", func(ctx SpecContext) {
		Expect(migrate(ctx, folder(first, "Cafe\u0301", nil))[first]).
			To(Equal("Caf\u00E9"))
		Expect(MustSucceed(resourcev0.StagedRenames(db))).To(BeEmpty())
	})

	It("Should keep each legal name, repeats in other parents included",
		func(ctx SpecContext) {
			parent := groupID(third)
			names := migrate(
				ctx,
				folder(first, "Stand A", nil),
				folder(second, "Stand A", &parent),
				folder(third, "Rigs", nil),
			)
			Expect(names).To(Equal(map[uuid.UUID]string{
				first:  "Stand A",
				second: "Stand A",
				third:  "Rigs",
			}))
			Expect(MustSucceed(resourcev0.StagedRenames(db))).To(BeEmpty())
		})

	It("Should keep the name of the lower key in a parent and move the higher key",
		func(ctx SpecContext) {
			names := migrate(
				ctx,
				folder(second, "Stand A", nil),
				folder(first, "Stand A", nil),
			)
			Expect(names[first]).To(Equal("Stand A"))
			Expect(names[second]).To(Equal("Stand A_2"))
		})

	It("Should keep a legal name when a lower key matches it only after NFC",
		func(ctx SpecContext) {
			names := migrate(
				ctx,
				folder(first, "Cafe\u0301", nil),
				folder(second, "Caf\u00E9", nil),
			)
			Expect(names[second]).To(Equal("Caf\u00E9"))
			Expect(names[first]).To(Equal("Caf\u00E9_2"))
		})

	It("Should leave each top-level Metrics folder to the clash pass",
		func(ctx SpecContext) {
			parent := groupID(third)
			a, b := uuid.New(), uuid.New()
			names := migrate(
				ctx,
				folder(first, v1.MetricsFolder, nil),
				folder(second, v1.MetricsFolder, nil),
				folder(third, "Rigs", nil),
				folder(a, v1.MetricsFolder, &parent),
				folder(b, v1.MetricsFolder, &parent),
			)
			Expect(names[first]).To(Equal(v1.MetricsFolder))
			Expect(names[second]).To(Equal(v1.MetricsFolder))
			Expect([]string{names[a], names[b]}).
				To(ConsistOf(v1.MetricsFolder, v1.MetricsFolder+"_2"))
			Expect(MustSucceed(resourcev0.StagedRenames(db))).To(HaveLen(1))
		})

	It("Should give a new name the first suffix that no sibling holds",
		func(ctx SpecContext) {
			parent := groupID(third)
			names := migrate(
				ctx,
				folder(first, "Stand_A", nil),
				folder(second, "Stand.A", nil),
				folder(third, "Rigs", nil),
				folder(uuid.New(), "Stand_A_2", &parent),
			)
			Expect(names[first]).To(Equal("Stand_A"))
			Expect(names[second]).To(Equal("Stand_A_2"))
		})

	It("Should leave a folder of another scope alone", func(ctx SpecContext) {
		users := folder(first, "a.b", nil)
		users.Scope = resource.TypeUser
		Expect(migrate(ctx, users)[first]).To(Equal("a.b"))
	})
})
