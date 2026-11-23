// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Color references", func() {
	It("Should inline a deleted named color into the panels that use it", func(
		ctx SpecContext,
	) {
		c := color.Named{Name: "Fault", Light: MustSucceed(xcolor.Parse("#ff0000"))}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return colorSvc.NewWriter(tx).Create(ctx, &c)
		})).To(Succeed())
		p := panel.Panel{
			Name:    "colored",
			Project: &proj.Key,
			Color:   &color.Ref{Variant: color.NamedRef{Key: c.Key}},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &p)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, p.Key)
			})).To(Succeed())
		})
		rec := &Recorder[panel.Key, panel.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return colorSvc.NewWriter(tx).Delete(ctx, c.Key)
		})).To(Succeed())
		var res panel.Panel
		Expect(svc.NewRetrieve().
			Where(panel.MatchKeys(p.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		Expect(res.Color).To(Equal(&color.Ref{Variant: color.FixedRef{Value: c.Light}}))
		seen := rec.Snapshot()
		Expect(seen).To(HaveLen(1))
		Expect(seen[0].Key).To(Equal(p.Key))
		Expect(seen[0].Actions).To(HaveLen(1))
		Expect(seen[0].Actions[0].Create).ToNot(BeNil())
		Expect(seen[0].Actions[0].Create.Panel).To(Equal(res))
	})

	DescribeTable(
		"Should drop a named color only when the Core does not hold it",
		func(ctx SpecContext, held bool) {
			c := color.Named{
				Name:  "Custom",
				Light: MustSucceed(xcolor.Parse("#123456")),
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return colorSvc.NewWriter(tx).Create(ctx, &c)
			})).To(Succeed())
			ref := &color.Ref{Variant: color.NamedRef{Key: c.Key}}
			if !held {
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return colorSvc.NewWriter(tx).Delete(ctx, c.Key)
				})).To(Succeed())
			} else {
				DeferCleanup(func(ctx SpecContext) {
					Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
						return colorSvc.NewWriter(tx).Delete(ctx, c.Key)
					})).To(Succeed())
				})
			}
			p := panel.Panel{Name: "imported", Color: ref}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.DropDangling(ctx, tx, &p)
			})).To(Succeed())
			if held {
				Expect(p.Color).To(Equal(ref))
			} else {
				Expect(p.Color).To(BeNil())
			}
		},
		Entry("held", true),
		Entry("missing", false),
	)
})
