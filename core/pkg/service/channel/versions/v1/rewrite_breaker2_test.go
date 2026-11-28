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
	// rewriter returns a rewriter with cfg over the rename of each channel that held
	// one of olds before the upgrade, which renamed it to that name with "_2", keyed
	// from 1 in order.
	rewriter := func(
		ctx context.Context,
		cfg parser.Config,
		olds ...string,
	) v1.Rewriter {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		byKey := make(map[v1.Key]v1.TextChannel, len(olds))
		for i, old := range olds {
			key := v1.Key(i + 1)
			byKey[key] = v1.TextChannel{Name: old + "_2", DataType: telem.Float64T}
			Expect(resource.StageRename(
				ctx,
				db,
				resource.ID{Type: resource.TypeChannel, Key: key.String()},
				old,
				resource.ReasonClash,
			)).To(Succeed())
		}
		renames := MustSucceed(v1.TextRename(db, byKey))
		return v1.Rewriter{Renames: renames, Config: cfg, Ambient: stl.NewSymbols()}
	}
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
		It("Should keep a read beside a captured read of another channel", func(
			ctx SpecContext,
		) {
			w := rewriter(ctx, parser.Config{}, "pt", "Metrics")
			text := "pt_2 f64 := 2.0\nreturn pt * pt_2 + Metrics"
			res := MustSucceed(w.Rewrite(ctx, calculation(text)))
			Expect(res.Text).To(Equal(text))
			Expect(byName(res, "Metrics")).To(ConsistOf(SatisfyAll(
				HaveField("Line", 2),
				HaveField("Column", 19),
				HaveField("Kept", true),
				HaveField("Captured", false),
			)))
		})

		It("Should keep a captured read", func(ctx SpecContext) {
			w := rewriter(ctx, parser.Config{}, "Metrics")
			text := "Metrics_2 f64 := 2.0\nreturn Metrics * Metrics_2"
			res := MustSucceed(w.Rewrite(ctx, calculation(text)))
			Expect(res.Text).To(Equal(text))
			Expect(byName(res, "Metrics")).To(ConsistOf(SatisfyAll(
				HaveField("Kept", true),
				HaveField("Captured", true),
			)))
		})
	})

	Describe("The splice of written reads", func() {
		It("Should write each read of each renamed channel", func(ctx SpecContext) {
			w := rewriter(ctx, parser.Config{}, "Metrics", "pt")
			res := MustSucceed(w.Rewrite(
				ctx,
				calculation("x := Metrics + pt\nreturn Metrics * pt + Metrics"),
			))
			Expect(res.Text).To(Equal(
				"x := Metrics_2 + pt_2\nreturn Metrics_2 * pt_2 + Metrics_2",
			))
			Expect(res.References).To(HaveEach(HaveField("Kept", false)))
		})

		It("Should write each read after a multi-byte character in a format string",
			func(ctx SpecContext) {
				w := rewriter(ctx, parser.Config{}, "Metrics", "pt")
				res := MustSucceed(w.Rewrite(
					ctx,
					calculation("s := f\"日本{Metrics}é{pt}\"\nreturn 1.0"),
				))
				Expect(res.Text).
					To(Equal("s := f\"日本{Metrics_2}é{pt_2}\"\nreturn 1.0"))
			})

		It("Should write the new name over a read written with an escape in a "+
			"placeholder", func(ctx SpecContext) {
			w := rewriter(ctx, parser.Config{}, "Metrics", "pt")
			text := "s := f\"{Metr\\u0069cs}{pt}\"\nreturn 1.0"
			res := MustSucceed(w.Rewrite(ctx, calculation(text)))
			Expect(res.Text).To(Equal("s := f\"{Metrics_2}{pt_2}\"\nreturn 1.0"))
		})
	})

	DescribeTable("Should write the new name into a write to a renamed channel",
		func(ctx SpecContext, src v1.Source, want string) {
			w := rewriter(ctx, parser.Config{}, "out", "sensor")
			res := MustSucceed(w.Rewrite(ctx, src))
			Expect(res.Text).To(Equal(want))
			Expect(byName(res, "out")).ToNot(BeEmpty())
			Expect(res.References).To(HaveEach(HaveField("Kept", false)))
		},
		Entry("a flow target",
			program("sensor -> scale{factor: 2.0} -> out"),
			"sensor_2 -> scale{factor: 2.0} -> out_2"),
		Entry("an assignment in a calculation",
			calculation("out = sensor\nreturn 1.0"),
			"out_2 = sensor_2\nreturn 1.0"),
		Entry("an authority entry",
			program("authority (out 200)\nsensor -> out"),
			"authority (out_2 200)\nsensor_2 -> out_2"),
		Entry("an assignment in a function",
			program("func f() {\n\tout = sensor\n}"),
			"func f() {\n\tout_2 = sensor_2\n}"),
	)

	It("Should write the new name into a read in a graph body", func(ctx SpecContext) {
		w := rewriter(ctx, parser.Config{}, "Metrics")
		res := MustSucceed(w.Rewrite(ctx, v1.Source{
			Text:     "{ return Metrics * x }",
			Form:     v1.FormBody,
			Function: "scale",
			Inputs:   types.Params{{Name: "x", Type: types.F64()}},
			Outputs:  types.Params{{Name: "output", Type: types.F64()}},
		}))
		Expect(res.Text).To(Equal("{ return Metrics_2 * x }"))
	})

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
		// migrate stores channels, a calculation keyed 4 that reads "pt", and folders,
		// runs the text chain, and returns the expression of the calculation.
		migrate := func(
			ctx context.Context,
			channels []v1.Channel,
			folders ...groupv1.Group,
		) string {
			GinkgoHelper()
			calc := channel(4, "doubled", nil)
			calc.Expression = "return pt * 2"
			Expect(gorp.WrapWriter[v1.Key, v1.Channel](db).
				Set(ctx, append(channels, calc)...)).To(Succeed())
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
		// run stores a built-in "pt" and the user's "pt" in the folder at, then
		// folders, and returns the expression of the calculation after the text chain.
		run := func(
			ctx context.Context,
			at uuid.UUID,
			folders ...groupv1.Group,
		) string {
			GinkgoHelper()
			builtin := channel(1, "pt", &at)
			builtin.Internal = true
			stored := []v1.Channel{builtin, channel(2, "pt", &at)}
			return migrate(ctx, stored, folders...)
		}

		It("Should write the new name of a channel under a folder that Arc cannot "+
			"read bare", func(ctx SpecContext) {
			Expect(run(
				ctx,
				third,
				folder(first, "Stand A", nil),
				folder(second, "plant", &first),
				folder(third, "rig", &second),
			)).To(Equal("return pt_2 * 2"))
			Expect(resource.StagedTextSites(db)).To(BeEmpty())
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

		It("Should write the new name of the channel that v0.62 bound when a "+
			"top-level channel keeps the old name", func(ctx SpecContext) {
			builtin := channel(2, "pt", &first)
			builtin.Internal = true
			Expect(migrate(
				ctx,
				[]v1.Channel{channel(1, "pt", nil), builtin, channel(3, "pt", &first)},
				folder(first, "Stand A", nil),
			)).To(Equal("return pt_2 * 2"))
			Expect(resource.StagedTextSites(db)).To(BeEmpty())
		})

		It("Should write the new name through a folder that the name pass made "+
			"readable", func(ctx SpecContext) {
			Expect(run(ctx, first, folder(first, "plant.a", nil))).
				To(Equal("return pt_2 * 2"))
			Expect(resource.StagedTextSites(db)).To(BeEmpty())
		})
	})
})
