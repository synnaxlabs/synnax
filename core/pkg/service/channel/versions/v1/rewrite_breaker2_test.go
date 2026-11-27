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
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
	groupv1 "github.com/synnaxlabs/synnax/pkg/service/group/versions/v1"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Folder rewrite breaker", func() {
	// held is a channel that held old before the upgrade, which renamed it to old with
	// "_2", in folders from the top level down.
	type held struct {
		old     string
		folders []string
	}
	// rewriter returns a rewriter with cfg over the rename of each of channels, keyed
	// from 1 in order.
	rewriter := func(
		ctx context.Context,
		cfg parser.Config,
		channels ...held,
	) v1.Rewriter {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		byKey := make(map[v1.Key]v1.TextChannel, len(channels))
		for i, c := range channels {
			key := v1.Key(i + 1)
			byKey[key] = v1.TextChannel{
				Name:     c.old + "_2",
				DataType: telem.Float64T,
				Folders:  c.folders,
			}
			Expect(resource.StageRename(
				ctx,
				db,
				resource.ID{Type: resource.TypeChannel, Key: key.String()},
				c.old,
				resource.ReasonClash,
			)).To(Succeed())
		}
		renames := MustSucceed(v1.TextRename(db, byKey))
		return v1.Rewriter{Renames: renames, Config: cfg, Ambient: stl.NewSymbols()}
	}
	standA := []string{"Stand A"}
	byName := func(res v1.Result, name string) []v1.Reference {
		var refs []v1.Reference
		for _, r := range res.References {
			if r.Name == name {
				refs = append(refs, r)
			}
		}
		return refs
	}
	calculation := func(text string) v1.Source {
		return v1.Source{Text: text, Form: v1.FormCalculation}
	}
	program := func(text string) v1.Source {
		return v1.Source{Text: text, Form: v1.FormProgram}
	}

	Describe("A text that a capture keeps", func() {
		It("Should mark a read of a channel in a folder that Arc cannot read when a "+
			"local captures a read of another channel", func(ctx SpecContext) {
			w := rewriter(
				ctx,
				parser.Config{},
				held{old: "pt"},
				held{old: "Metrics", folders: standA},
			)
			text := "pt_2 f64 := 2.0\nreturn pt * pt_2 + Metrics"
			res := MustSucceed(w.Rewrite(ctx, calculation(text)))
			Expect(res.Text).To(Equal(text))
			Expect(byName(res, "Metrics")).To(ConsistOf(SatisfyAll(
				HaveField("Line", 2),
				HaveField("Column", 19),
				HaveField("Kept", true),
				HaveField("Captured", false),
				HaveField("FolderUnreadable", true),
			)))
		})

		It("Should mark a captured read of a channel in a folder that Arc cannot read",
			func(ctx SpecContext) {
				w := rewriter(
					ctx,
					parser.Config{},
					held{old: "Metrics", folders: standA},
				)
				text := "Metrics_2 f64 := 2.0\nreturn Metrics * Metrics_2"
				res := MustSucceed(w.Rewrite(ctx, calculation(text)))
				Expect(res.Text).To(Equal(text))
				Expect(byName(res, "Metrics")).To(ConsistOf(SatisfyAll(
					HaveField("Kept", true),
					HaveField("FolderUnreadable", true),
				)))
			})
	})

	Describe("The splice after a kept read", func() {
		It("Should write each read of a readable channel between kept reads",
			func(ctx SpecContext) {
				w := rewriter(
					ctx,
					parser.Config{},
					held{old: "Metrics", folders: standA},
					held{old: "pt"},
				)
				res := MustSucceed(w.Rewrite(
					ctx,
					calculation("x := Metrics + pt\nreturn Metrics * pt + Metrics"),
				))
				Expect(res.Text).
					To(Equal("x := Metrics + pt_2\nreturn Metrics * pt_2 + Metrics"))
				Expect(byName(res, "Metrics")).To(ConsistOf(
					SatisfyAll(HaveField("Line", 1), HaveField("Column", 5),
						HaveField("Kept", true), HaveField("FolderUnreadable", true)),
					SatisfyAll(HaveField("Line", 2), HaveField("Column", 7),
						HaveField("Kept", true), HaveField("FolderUnreadable", true)),
					SatisfyAll(HaveField("Line", 2), HaveField("Column", 22),
						HaveField("Kept", true), HaveField("FolderUnreadable", true)),
				))
				Expect(byName(res, "pt")).To(HaveEach(HaveField("Kept", false)))
			})

		It("Should keep a read after a multi-byte character in a format string and "+
			"write the read after it", func(ctx SpecContext) {
			w := rewriter(
				ctx,
				parser.Config{},
				held{old: "Metrics", folders: standA},
				held{old: "pt"},
			)
			res := MustSucceed(w.Rewrite(
				ctx,
				calculation("s := f\"日本{Metrics}é{pt}\"\nreturn 1.0"),
			))
			Expect(res.Text).To(Equal("s := f\"日本{Metrics}é{pt_2}\"\nreturn 1.0"))
			Expect(byName(res, "Metrics")).To(ConsistOf(SatisfyAll(
				HaveField("Line", 1),
				HaveField("Column", 10),
				HaveField("Kept", true),
				HaveField("FolderUnreadable", true),
			)))
		})

		It("Should keep a read written with an escape in a placeholder", func(
			ctx SpecContext,
		) {
			w := rewriter(
				ctx,
				parser.Config{},
				held{old: "Metrics", folders: standA},
				held{old: "pt"},
			)
			text := "s := f\"{Metr\\u0069cs}{pt}\"\nreturn 1.0"
			res := MustSucceed(w.Rewrite(ctx, calculation(text)))
			Expect(res.Text).To(Equal("s := f\"{Metr\\u0069cs}{pt_2}\"\nreturn 1.0"))
		})
	})

	DescribeTable("Should keep a write to a channel in a folder that Arc cannot read",
		func(ctx SpecContext, src v1.Source, want string) {
			w := rewriter(
				ctx,
				parser.Config{},
				held{old: "out", folders: standA},
				held{old: "sensor"},
			)
			res := MustSucceed(w.Rewrite(ctx, src))
			Expect(res.Text).To(Equal(want))
			Expect(byName(res, "out")).To(HaveEach(SatisfyAll(
				HaveField("Kept", true),
				HaveField("FolderUnreadable", true),
			)))
			Expect(byName(res, "out")).ToNot(BeEmpty())
		},
		Entry("a flow target",
			program("sensor -> scale{factor: 2.0} -> out"),
			"sensor_2 -> scale{factor: 2.0} -> out"),
		Entry("an assignment in a calculation",
			calculation("out = sensor\nreturn 1.0"),
			"out = sensor_2\nreturn 1.0"),
		Entry("an authority entry",
			program("authority (out 200)\nsensor -> out"),
			"authority (out 200)\nsensor_2 -> out"),
		Entry("an assignment in a function",
			program("func f() {\n\tout = sensor\n}"),
			"func f() {\n\tout = sensor_2\n}"),
	)

	DescribeTable("Should keep a read in a graph body of a channel in a folder that "+
		"Arc cannot read", func(ctx SpecContext, folders ...string) {
		w := rewriter(ctx, parser.Config{}, held{old: "Metrics", folders: folders})
		src := v1.Source{
			Text:     "{ return Metrics * x }",
			Form:     v1.FormBody,
			Function: "scale",
			Inputs:   types.Params{{Name: "x", Type: types.F64()}},
			Outputs:  types.Params{{Name: "output", Type: types.F64()}},
		}
		res := MustSucceed(w.Rewrite(ctx, src))
		Expect(res.Text).To(Equal(src.Text))
		Expect(res.References).To(ConsistOf(HaveField("FolderUnreadable", true)))
	},
		Entry("a tab", "Stand\tA"),
		Entry("a keyword type", "f64"),
		Entry("a boolean", "true"),
		Entry("a logical keyword", "and"),
		Entry("an authority keyword", "authority"),
		Entry("a slash", "a/b"),
		Entry("a comment", "a//b"),
		Entry("a non-ASCII letter", "Ström"),
		Entry("a combining mark", "plant\u0301"),
		Entry("a leading digit after a readable folder", "plant", "2nd"),
		Entry("an unreadable ancestor of readable folders", "Stand A", "plant", "x"),
	)

	DescribeTable("Should keep a read through a dashed folder that the dashed lexer "+
		"splits", func(ctx SpecContext, folder string) {
		w := rewriter(
			ctx,
			parser.Config{AllowDashedNames: true},
			held{old: "Metrics", folders: []string{folder}},
		)
		res := MustSucceed(w.Rewrite(ctx, calculation("return Metrics")))
		Expect(res.Text).To(Equal("return Metrics"))
	},
		Entry("an arrow", "a->b"),
		Entry("a compound assignment", "a-=b"),
		Entry("a leading dash", "-a"),
	)

	Describe("CalculationTextMigration", func() {
		const host node.Key = 1
		var (
			db     *gorp.DB
			first  = uuid.MustParse("00000000-0000-4000-8000-000000000001")
			second = uuid.MustParse("00000000-0000-4000-8000-000000000002")
			third  = uuid.MustParse("00000000-0000-4000-8000-000000000003")
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
				DataType:    telem.Float64T,
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
		// run stores a built-in "pt" and the user's "pt" in the folder at, and a
		// calculation that reads "pt", runs the text chain, and returns the
		// expression of the calculation.
		run := func(
			ctx context.Context,
			at uuid.UUID,
			folders ...groupv1.Group,
		) string {
			GinkgoHelper()
			builtin := channel(1, "pt", &at)
			builtin.Internal = true
			calc := channel(3, "doubled", nil)
			calc.Expression = "return pt * 2"
			channels := []v1.Channel{builtin, channel(2, "pt", &at), calc}
			Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(ctx, channels...)).
				To(Succeed())
			Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](db).
				Set(ctx, folders...)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:        db,
				Namespace: "channel",
				Migrations: []gorp.Migration{
					groupv1.NameRuleMigration,
					v1.NameRuleMigration,
					v1.NewClashMigration(host),
					v1.NewCalculationTextMigration(parser.Config{}),
				},
			})).To(Succeed())
			var c v1.Channel
			Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
				Where(gorp.MatchKeys[v1.Key, v1.Channel](calc.Key())).
				Entry(&c).
				Exec(ctx, db)).To(Succeed())
			return c.Expression
		}

		It("Should keep and stage a read of a channel in a readable folder under a "+
			"folder that Arc cannot read", func(ctx SpecContext) {
			Expect(run(
				ctx,
				third,
				folder(first, "Stand A", nil),
				folder(second, "plant", &first),
				folder(third, "rig", &second),
			)).To(Equal("return pt * 2"))
			Expect(resource.StagedTextSites(db)).To(ConsistOf(
				HaveField("FolderUnreadable", true),
			))
		})

		It("Should write the new name through nested readable folders",
			func(ctx SpecContext) {
				Expect(run(
					ctx,
					second,
					folder(first, "plant", nil),
					folder(second, "rig", &first),
				)).To(Equal("return pt_2 * 2"))
				Expect(resource.StagedTextSites(db)).To(BeEmpty())
			})

		It("Should keep and stage a read of a renamed channel in a folder that Arc "+
			"cannot read when a top-level channel holds the old name",
			func(ctx SpecContext) {
				builtin := channel(2, "pt", &first)
				builtin.Internal = true
				calc := channel(4, "doubled", nil)
				calc.Expression = "return pt * 2"
				Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).Set(
					ctx,
					channel(1, "pt", nil),
					builtin,
					channel(3, "pt", &first),
					calc,
				)).To(Succeed())
				Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](db).
					Set(ctx, folder(first, "Stand A", nil))).To(Succeed())
				Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
					DB:        db,
					Namespace: "channel",
					Migrations: []gorp.Migration{
						groupv1.NameRuleMigration,
						v1.NameRuleMigration,
						v1.NewClashMigration(host),
						v1.NewCalculationTextMigration(parser.Config{}),
					},
				})).To(Succeed())
				var c v1.Channel
				Expect(gorp.NewRetrieve[v1.Key, v1.Channel]().
					Where(gorp.MatchKeys[v1.Key, v1.Channel](calc.Key())).
					Entry(&c).
					Exec(ctx, db)).To(Succeed())
				Expect(c.Expression).To(Equal("return pt * 2"))
				Expect(resource.StagedTextSites(db)).To(ConsistOf(SatisfyAll(
					HaveField("Name", "pt"),
					HaveField("FolderUnreadable", true),
				)))
			})

		// A v0.62 Core moved a folder with no check of its own chain, so a store can
		// hold a folder whose chain comes back to it.
		It("Should return the folders of a channel whose folder chain comes back to "+
			"itself", func(ctx SpecContext) {
			loop := []groupv1.Group{
				folder(first, "plant", &second),
				folder(second, "rig", &first),
			}
			Expect(gorp.WrapWriter[groupv1.Key, groupv1.Group](db).Set(ctx, loop...)).
				To(Succeed())
			done := make(chan error, 1)
			go func() {
				tx := db.OpenTx()
				_, err := v1.TextChannels(
					ctx,
					tx,
					[]v1.Channel{channel(1, "pt", &first)},
				)
				if closeErr := tx.Close(); err == nil {
					err = closeErr
				}
				done <- err
			}()
			Eventually(done).WithTimeout(300 * time.Millisecond).Should(Receive())
		})

		It("Should write the new name through a folder that the name pass made "+
			"readable", func(ctx SpecContext) {
			Expect(run(ctx, first, folder(first, "plant.a", nil))).
				To(Equal("return pt_2 * 2"))
			Expect(resource.StagedTextSites(db)).To(BeEmpty())
		})
	})
})
