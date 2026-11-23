// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Color references", func() {
	It(
		"Should inline a deleted named color into the nodes and edges that use it",
		func(ctx SpecContext) {
			gone := color.Named{
				Name:  "Fault",
				Light: MustSucceed(xcolor.Parse("#ff0000")),
			}
			kept := color.Named{
				Name:  "Nominal",
				Light: MustSucceed(xcolor.Parse("#00ff00")),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				w := colorSvc.NewWriter(tx)
				if err := w.Create(ctx, &gone); err != nil {
					return err
				}
				return w.Create(ctx, &kept)
			})).To(Succeed())
			named := func(n color.Named) *color.Ref {
				return &color.Ref{Variant: color.NamedRef{Key: n.Key}}
			}
			box := defaults.Fill(schematic.NodeConfig{Variant: schematic.BoxNodeConfig{
				StrokeColor: named(gone),
				FillColor:   named(kept),
			}})
			scale := defaults.Fill(schematic.NodeConfig{
				Variant: schematic.ScaleNodeConfig{
					ScaleIndicatorConfig: schematic.ScaleIndicatorConfig{
						LevelColor: named(gone),
					},
				},
			})
			e := edge("e", "a", "o", "b", "i")
			e.Config = pipeCfg("")
			pipe := as[schematic.PipeEdgeConfig](e.Config.Variant)
			pipe.StrokeColor = named(gone)
			e.Config.Variant = pipe
			s := schematic.Schematic{
				Project: &proj.Key,
				Name:    "colored",
				Nodes: []schematic.Node{
					configured(node("a", 0, 0), box),
					configured(node("b", 0, 0), scale),
				},
				Edges: []schematic.Edge{e},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &s)
			})).To(Succeed())
			DeferCleanup(func(ctx SpecContext) {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					if err := svc.NewWriter(tx).Delete(ctx, s.Key); err != nil {
						return err
					}
					return colorSvc.NewWriter(tx).Delete(ctx, kept.Key)
				})).To(Succeed())
			})
			rec := &Recorder[schematic.Key, schematic.Action]{}
			DeferCleanup(svc.OnAction(rec.Record))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return colorSvc.NewWriter(tx).Delete(ctx, gone.Key)
			})).To(Succeed())
			var res schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&res).
				Exec(ctx, db)).To(Succeed())
			fixed := &color.Ref{Variant: color.FixedRef{Value: gone.Light}}
			b := as[schematic.BoxNodeConfig](res.Nodes[0].Config.Variant)
			Expect(b.StrokeColor).To(Equal(fixed))
			Expect(b.FillColor).To(Equal(named(kept)))
			sc := as[schematic.ScaleNodeConfig](res.Nodes[1].Config.Variant)
			Expect(sc.LevelColor).To(Equal(fixed))
			p := as[schematic.PipeEdgeConfig](res.Edges[0].Config.Variant)
			Expect(p.StrokeColor).To(Equal(fixed))
			seen := rec.Snapshot()
			Expect(seen).To(HaveLen(1))
			Expect(seen[0].Key).To(Equal(s.Key))
			Expect(seen[0].Actions).To(HaveLen(1))
			Expect(seen[0].Actions[0].Create).ToNot(BeNil())
			Expect(seen[0].Actions[0].Create.Schematic).To(Equal(res))
		},
	)
})

// as returns v as a T and fails the spec when v is not a T.
func as[T any](v any) T {
	GinkgoHelper()
	t, ok := v.(T)
	Expect(ok).To(BeTrue())
	return t
}
