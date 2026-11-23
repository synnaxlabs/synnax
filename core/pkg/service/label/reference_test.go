// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Color references", func() {
	It("Should inline a deleted named color into the labels that use it", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		colors := MustOpen(color.OpenService(ctx, color.ServiceConfig{
			DB:    db,
			Group: groups,
		}))
		svc := MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
		gone := color.Named{Name: "Fault", Light: MustSucceed(xcolor.Parse("#ff0000"))}
		kept := color.Named{
			Name:  "Nominal",
			Light: MustSucceed(xcolor.Parse("#00ff00")),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := colors.NewWriter(tx)
			if err := w.Create(ctx, &gone); err != nil {
				return err
			}
			return w.Create(ctx, &kept)
		})).To(Succeed())
		named := func(n color.Named) color.Ref {
			return color.Ref{Variant: color.NamedRef{Key: n.Key}}
		}
		fault := label.Label{Name: "Fault", Color: named(gone)}
		nominal := label.Label{Name: "Nominal", Color: named(kept)}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			if err := w.Create(ctx, &fault); err != nil {
				return err
			}
			return w.Create(ctx, &nominal)
		})).To(Succeed())
		rec := &Recorder[label.Key, label.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return colors.NewWriter(tx).Delete(ctx, gone.Key)
		})).To(Succeed())
		var res []label.Label
		Expect(svc.NewRetrieve().
			Where(label.MatchKeys(fault.Key, nominal.Key)).
			Entries(&res).
			Exec(ctx, db)).To(Succeed())
		Expect(res).To(ConsistOf(
			label.Label{Key: fault.Key, Name: "Fault", Color: fixed("#ff0000")},
			nominal,
		))
		seen := rec.Snapshot()
		Expect(seen).To(HaveLen(1))
		Expect(seen[0].Key).To(Equal(fault.Key))
		Expect(seen[0].Actions).To(Equal([]label.Action{label.NewSetAction(
			label.SetPayload{Path: []string{"color"}, Value: fixed("#ff0000")},
		)}))
	})
})
