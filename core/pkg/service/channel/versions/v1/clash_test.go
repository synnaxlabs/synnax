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
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/telem"
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
				[]v1.Channel{channel(1, v1.MetricsFolder, nil)},
				folder(first, v1.MetricsFolder, nil),
			)
			Expect(channels[1]).To(Equal(v1.MetricsFolder + "_2"))
			Expect(folders[first]).To(Equal(v1.MetricsFolder))
		})

	It("Should rename a top-level channel named for the metrics folder when no "+
		"folder holds the name", func(ctx SpecContext) {
		channels, _ := migrate(
			ctx,
			clash,
			[]v1.Channel{channel(1, v1.MetricsFolder, nil)},
		)
		Expect(channels[1]).To(Equal(v1.MetricsFolder + "_2"))
	})

	It("Should keep the name on the metrics folder that holds the metrics index of "+
		"the host", func(ctx SpecContext) {
		index := channel(1, "sy_node_1_metrics_time", &second)
		index.Internal = true
		_, folders := migrate(
			ctx,
			clash,
			[]v1.Channel{index},
			folder(first, v1.MetricsFolder, nil),
			folder(second, v1.MetricsFolder, nil),
		)
		Expect(folders).To(Equal(map[uuid.UUID]string{
			first:  v1.MetricsFolder + "_2",
			second: v1.MetricsFolder,
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
				folder(second, v1.MetricsFolder, nil),
				folder(first, v1.MetricsFolder, nil),
			)
			Expect(folders).To(Equal(map[uuid.UUID]string{
				first:  v1.MetricsFolder,
				second: v1.MetricsFolder + "_2",
			}))
		})

	It("Should give a channel the first suffix that no channel and no sibling holds",
		func(ctx SpecContext) {
			channels, _ := migrate(
				ctx,
				clash,
				[]v1.Channel{
					channel(1, v1.MetricsFolder, nil),
					channel(2, v1.MetricsFolder+"_2", &second),
				},
				folder(first, v1.MetricsFolder, nil),
				folder(second, v1.MetricsFolder+"_3", nil),
			)
			Expect(channels[1]).To(Equal(v1.MetricsFolder + "_4"))
		})

	Describe("CalculationTextMigration", func() {
		text := []gorp.Migration{
			v1.NameRuleMigration,
			v1.NewClashMigration(host),
			v1.NewCalculationTextMigration(parser.Config{}),
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
				calc.Expression = "// Metrics\nreturn Metrics * 2"
				migrate(
					ctx,
					text,
					[]v1.Channel{channel(1, v1.MetricsFolder, nil), calc},
					folder(first, v1.MetricsFolder, nil),
				)
				Expect(
					expressionOf(ctx, 2),
				).To(Equal("// Metrics\nreturn Metrics_2 * 2"))
				Expect(resource.StagedTextSites(db)).To(BeEmpty())
			})

		It("Should leave a calculation that a v0.62 Core could not parse, and stage it",
			func(ctx SpecContext) {
				// A v0.62 Core wrapped the expression in braces, so the comment hides
				// the closing one.
				calc := channel(2, "doubled", nil)
				calc.Expression = "return Metrics * 2 // Metrics"
				migrate(
					ctx,
					text,
					[]v1.Channel{channel(1, v1.MetricsFolder, nil), calc},
					folder(first, v1.MetricsFolder, nil),
				)
				Expect(expressionOf(ctx, 2)).To(Equal(calc.Expression))
				Expect(resource.StagedTextSites(db)).To(ConsistOf(resource.TextSite{
					Holder:  calc.ResourceID(),
					Name:    v1.MetricsFolder,
					NewName: "Metrics_2",
					Line:    1,
					Column:  7,
				}))
			})

		It("Should leave and stage a read of a renamed channel in a folder that Arc "+
			"cannot read bare yet", func(ctx SpecContext) {
			// A v0.62 Core bound "pt" to the user's channel, the higher key, which the
			// built-in of the name in its folder takes from it.
			builtin := channel(1, "pt", &first)
			builtin.Internal = true
			calc := channel(3, "doubled", nil)
			calc.Expression = "return pt * 2"
			channels, _ := migrate(
				ctx,
				text,
				[]v1.Channel{builtin, channel(2, "pt", &first), calc},
				folder(first, "Stand A", nil),
			)
			Expect(channels[2]).To(Equal("pt_2"))
			Expect(expressionOf(ctx, 3)).To(Equal(calc.Expression))
			Expect(resource.StagedTextSites(db)).To(ConsistOf(resource.TextSite{
				Holder:  calc.ResourceID(),
				Name:    "pt",
				NewName: "pt_2",
				Line:    1,
				Column:  7,
				Folder:  "Stand A",
			}))
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
	var db *gorp.DB
	BeforeEach(func() { db = DeferClose(gorp.Wrap(memkv.New())) })
	id := func(key v1.Key) resource.ID {
		return resource.ID{Type: resource.TypeChannel, Key: key.String()}
	}
	named := func(names map[v1.Key]string) map[v1.Key]v1.TextChannel {
		channels := make(map[v1.Key]v1.TextChannel, len(names))
		for k, n := range names {
			channels[k] = v1.TextChannel{Name: n, DataType: telem.Float32T}
		}
		return channels
	}
	stage := func(ctx SpecContext, renames ...resource.Rename) {
		GinkgoHelper()
		for _, r := range renames {
			Expect(resource.StageRename(ctx, db, r.ID, r.From, resource.ReasonRepeat)).
				To(Succeed())
		}
	}

	It("Should map a name to the renamed channel with the highest key that held it",
		func(ctx SpecContext) {
			stage(
				ctx,
				resource.Rename{ID: id(7), From: "Metrics"},
				resource.Rename{ID: id(3), From: "Metrics"},
			)
			rename := MustSucceed(v1.TextRename(
				db,
				named(map[v1.Key]string{3: "Metrics_2", 7: "Metrics_3", 9: "pt"}),
			))
			to, ok := rename.Rename("Metrics")
			Expect(ok).To(BeTrue())
			Expect(to).To(Equal("Metrics_3"))
			_, ok = rename.Rename("pt")
			Expect(ok).To(BeFalse())
		})

	It("Should map no name when the channel that v0.62 bound keeps it", func(
		ctx SpecContext,
	) {
		stage(ctx, resource.Rename{ID: id(3), From: "Metrics"})
		rename := MustSucceed(v1.TextRename(
			db,
			named(map[v1.Key]string{3: "Metrics_2", 7: "Metrics"}),
		))
		_, ok := rename.Rename("Metrics")
		Expect(ok).To(BeFalse())
	})

	It("Should map a name by the bytes that each channel held before the upgrade",
		func(ctx SpecContext) {
			// v0.62 bound the NFD form to the channel that held it, which the upgrade
			// renamed. The NFC form reads the channel that keeps it.
			stage(ctx, resource.Rename{ID: id(3), From: "cafe\u0301"})
			rename := MustSucceed(v1.TextRename(
				db,
				named(map[v1.Key]string{3: "cafe_2", 7: "café"}),
			))
			to, ok := rename.Rename("cafe\u0301")
			Expect(ok).To(BeTrue())
			Expect(to).To(Equal("cafe_2"))
			_, ok = rename.Rename("café")
			Expect(ok).To(BeFalse())
		})

	It("Should map no name that only a folder held", func(ctx SpecContext) {
		stage(ctx, resource.Rename{
			ID:   resource.ID{Type: resource.TypeGroup, Key: uuid.New().String()},
			From: "Rigs",
		})
		rename := MustSucceed(v1.TextRename(db, named(map[v1.Key]string{3: "pt"})))
		_, ok := rename.Rename("Rigs")
		Expect(ok).To(BeFalse())
	})

	It("Should return no rename when the upgrade staged none", func() {
		Expect(v1.TextRename(db, named(map[v1.Key]string{3: "pt"}))).To(BeNil())
	})

	It("Should fail on a staged rename of a channel that has no record", func(
		ctx SpecContext,
	) {
		stage(ctx, resource.Rename{ID: id(3), From: "pt"})
		Expect(v1.TextRename(db, named(map[v1.Key]string{}))).Error().
			To(MatchError(ContainSubstring("names no record")))
	})
})
