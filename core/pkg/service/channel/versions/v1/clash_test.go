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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("NewClashMigration", func() {
	const host node.Key = 1
	var (
		db     *gorp.DB
		first  = uuid.MustParse("00000000-0000-4000-8000-000000000001")
		second = uuid.MustParse("00000000-0000-4000-8000-000000000002")
	)
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	channel := func(local v1.LocalKey, name string, folder *uuid.UUID) v1.Channel {
		return v1.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    local,
			LocalIndex:  local,
			IsIndex:     true,
			Virtual:     true,
			Group:       folder,
		}
	}
	folder := func(key uuid.UUID, name string, parent *uuid.UUID) groupv1.Group {
		f := groupv1.Group{Key: key, Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			f.Parent = &resource.ID{Type: resource.TypeGroup, Key: parent.String()}
		}
		return f
	}
	// migrate stores channels and folders, runs chain, and returns the name of each
	// channel by local key and of each folder by key.
	migrate := func(
		ctx context.Context,
		chain []gorp.Migration,
		channels []v1.Channel,
		folders ...groupv1.Group,
	) (map[v1.LocalKey]string, map[uuid.UUID]string) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(ctx, channels...)).
			To(Succeed())
		Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](db).Set(ctx, folders...)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "channel",
			Migrations: chain,
		})).To(Succeed())
		var storedChannels []v1.Channel
		Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
			Entries(&storedChannels).
			Exec(ctx, db)).To(Succeed())
		channelNames := make(map[v1.LocalKey]string, len(storedChannels))
		for _, c := range storedChannels {
			channelNames[c.LocalKey] = c.Name
		}
		var storedFolders []groupv1.Group
		Expect(gorp.NewRetrieve[groupv1.Key, groupv1.Group]().
			Entries(&storedFolders).
			Exec(ctx, db)).To(Succeed())
		folderNames := make(map[uuid.UUID]string, len(storedFolders))
		for _, f := range storedFolders {
			folderNames[f.Key] = f.Name
		}
		return channelNames, folderNames
	}
	clash := []gorp.Migration{v1.NewClashMigration(host)}

	It("Should rename a folder that holds the name of a channel beside it",
		func(ctx SpecContext) {
			channels, folders := migrate(
				ctx,
				clash,
				[]v1.Channel{channel(1, "Rigs", nil), channel(2, "pt", &first)},
				folder(first, "Rigs", nil),
				folder(second, "pt", &first),
			)
			Expect(channels).To(Equal(map[v1.LocalKey]string{1: "Rigs", 2: "pt"}))
			Expect(folders).To(Equal(map[uuid.UUID]string{
				first:  "Rigs_2",
				second: "pt_2",
			}))
			Expect(MustSucceed(resource.StagedRenames(db))).To(ConsistOf(
				resource.Rename{
					ID:     folder(first, "", nil).ResourceID(),
					From:   "Rigs",
					Reason: resource.ReasonClash,
				},
				resource.Rename{
					ID:     folder(second, "", nil).ResourceID(),
					From:   "pt",
					Reason: resource.ReasonClash,
				},
			))
		})

	It("Should leave a channel and a folder with one name in two folders",
		func(ctx SpecContext) {
			channels, folders := migrate(
				ctx,
				clash,
				[]v1.Channel{channel(1, "pt", &first)},
				folder(first, "Rigs", nil),
				folder(second, "pt", nil),
			)
			Expect(channels[1]).To(Equal("pt"))
			Expect(folders[second]).To(Equal("pt"))
			Expect(MustSucceed(resource.StagedRenames(db))).To(BeEmpty())
		})

	It("Should rename a folder that holds the name of a channel only after NFC",
		func(ctx SpecContext) {
			_, folders := migrate(
				ctx,
				clash,
				[]v1.Channel{channel(1, "Café", nil)},
				folder(first, "Café", nil),
			)
			Expect(folders[first]).To(Equal("Café_2"))
		})

	It("Should rename a channel that a name pass renamed and not the folder",
		func(ctx SpecContext) {
			ch := channel(1, "a_b", nil)
			Expect(resource.StageRename(
				ctx,
				db,
				ch.ResourceID(),
				"a.b",
				resource.ReasonSeparator,
			)).To(Succeed())
			channels, folders := migrate(
				ctx,
				clash,
				[]v1.Channel{ch},
				folder(first, "a_b", nil),
			)
			Expect(channels[1]).To(Equal("a_b_2"))
			Expect(folders[first]).To(Equal("a_b"))
			Expect(MustSucceed(resource.StagedRenames(db))).To(ConsistOf(
				resource.Rename{
					ID:     ch.ResourceID(),
					From:   "a.b",
					Reason: resource.ReasonSeparator,
				},
			))
		})

	It("Should rename a top-level channel that holds the name of the metrics folder",
		func(ctx SpecContext) {
			channels, folders := migrate(
				ctx,
				clash,
				[]v1.Channel{channel(1, groupv1.MetricsFolder, nil)},
				folder(first, groupv1.MetricsFolder, nil),
			)
			Expect(channels[1]).To(Equal(groupv1.MetricsFolder + "_2"))
			Expect(folders[first]).To(Equal(groupv1.MetricsFolder))
		})

	It("Should rename a top-level channel named for the metrics folder when no "+
		"folder holds the name", func(ctx SpecContext) {
		channels, _ := migrate(
			ctx,
			clash,
			[]v1.Channel{channel(1, groupv1.MetricsFolder, nil)},
		)
		Expect(channels[1]).To(Equal(groupv1.MetricsFolder + "_2"))
	})

	It("Should keep the name on the metrics folder that holds the metrics index of "+
		"the host", func(ctx SpecContext) {
		index := channel(1, "sy_node_1_metrics_time", &second)
		index.Internal = true
		_, folders := migrate(
			ctx,
			clash,
			[]v1.Channel{index},
			folder(first, groupv1.MetricsFolder, nil),
			folder(second, groupv1.MetricsFolder, nil),
		)
		Expect(folders).To(Equal(map[uuid.UUID]string{
			first:  groupv1.MetricsFolder + "_2",
			second: groupv1.MetricsFolder,
		}))
	})

	It("Should keep the name on the first metrics folder when none holds the index",
		func(ctx SpecContext) {
			index := channel(1, "sy_node_2_metrics_time", &second)
			index.Internal = true
			_, folders := migrate(
				ctx,
				clash,
				[]v1.Channel{index},
				folder(second, groupv1.MetricsFolder, nil),
				folder(first, groupv1.MetricsFolder, nil),
			)
			Expect(folders).To(Equal(map[uuid.UUID]string{
				first:  groupv1.MetricsFolder,
				second: groupv1.MetricsFolder + "_2",
			}))
		})

	It("Should give a channel the first suffix that no channel and no sibling holds",
		func(ctx SpecContext) {
			channels, _ := migrate(
				ctx,
				clash,
				[]v1.Channel{
					channel(1, groupv1.MetricsFolder, nil),
					channel(2, groupv1.MetricsFolder+"_2", &second),
				},
				folder(first, groupv1.MetricsFolder, nil),
				folder(second, groupv1.MetricsFolder+"_3", nil),
			)
			Expect(channels[1]).To(Equal(groupv1.MetricsFolder + "_4"))
		})

	Describe("CalculationTextMigration", func() {
		text := []gorp.Migration{
			v1.NameRuleMigration,
			v1.NewClashMigration(host),
			v1.CalculationTextMigration,
		}
		expressionOf := func(ctx context.Context, local v1.LocalKey) string {
			GinkgoHelper()
			var c v1.Channel
			Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
				Where(gorp.MatchKeys[v1.Key, v1.Channel](channel(local, "", nil).Key())).
				Entry(&c).
				Exec(ctx, db)).To(Succeed())
			return c.Expression
		}

		It("Should write the new name of a renamed channel into a calculation",
			func(ctx SpecContext) {
				calc := channel(2, "doubled", nil)
				calc.Expression = "return Metrics * 2 // Metrics"
				migrate(
					ctx,
					text,
					[]v1.Channel{channel(1, groupv1.MetricsFolder, nil), calc},
					folder(first, groupv1.MetricsFolder, nil),
				)
				Expect(
					expressionOf(ctx, 2),
				).To(Equal("return Metrics_2 * 2 // Metrics"))
			})

		It("Should leave a name that a channel still holds", func(ctx SpecContext) {
			calc := channel(3, "doubled", nil)
			calc.Expression = "return pt * 2"
			channels, _ := migrate(
				ctx,
				text,
				[]v1.Channel{channel(1, "pt", nil), channel(2, "pt", nil), calc},
			)
			Expect(channels[1]).To(Equal("pt_2"))
			Expect(expressionOf(ctx, 3)).To(Equal("return pt * 2"))
		})
	})
})

var _ = Describe("TextRename", func() {
	id := func(key v1.Key) resource.ID {
		return resource.ID{Type: resource.TypeChannel, Key: key.String()}
	}

	It("Should map a name to the renamed channel with the highest key that held it",
		func() {
			rename := MustSucceed(v1.TextRename(
				map[v1.Key]string{3: "Metrics_2", 7: "Metrics_3", 9: "pt"},
				[]resource.Rename{
					{ID: id(7), From: "Metrics"},
					{ID: id(3), From: "Metrics"},
				},
			))
			to, ok := rename("Metrics")
			Expect(ok).To(BeTrue())
			Expect(to).To(Equal("Metrics_3"))
			_, ok = rename("pt")
			Expect(ok).To(BeFalse())
		})

	It("Should map no name that a channel holds now in NFC", func() {
		rename := MustSucceed(v1.TextRename(
			map[v1.Key]string{3: "cafe_2", 7: "café"},
			[]resource.Rename{{ID: id(3), From: "café"}},
		))
		_, ok := rename("café")
		Expect(ok).To(BeFalse())
	})

	It("Should map no name that only a folder held", func() {
		rename := MustSucceed(v1.TextRename(
			map[v1.Key]string{3: "pt"},
			[]resource.Rename{{
				ID:   resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()},
				From: "Rigs",
			}},
		))
		_, ok := rename("Rigs")
		Expect(ok).To(BeFalse())
	})

	It("Should fail on a staged rename of a channel that has no record", func() {
		Expect(v1.TextRename(
			map[v1.Key]string{},
			[]resource.Rename{{ID: id(3), From: "pt"}},
		)).Error().To(MatchError(ContainSubstring("names no record")))
	})
})
