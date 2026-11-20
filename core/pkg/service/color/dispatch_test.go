// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Broadcasts", func() {
	type frame = actions.Scoped[color.Key, color.Action]
	var (
		db  *gorp.DB
		svc *color.Service
		red xcolor.Color
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc = MustOpen(
			color.OpenService(ctx, color.ServiceConfig{DB: db, Group: groups}),
		)
		red = MustSucceed(xcolor.Parse("#ff0000"))
	})
	record := func() *Recorder[color.Key, color.Action] {
		rec := &Recorder[color.Key, color.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	rename := func(name string) color.Action {
		return color.NewSetAction(color.SetPayload{Path: []string{"name"}, Value: name})
	}

	It("Should broadcast a create with the dispatch key after the commit", func(
		ctx SpecContext,
	) {
		rec := record()
		n := color.Named{Name: "Breaker", Light: red}
		tx := DeferClose(db.OpenTx())
		Expect(svc.NewWriter(tx).WithDispatchKey("dk").Create(ctx, &n)).To(Succeed())
		Expect(rec.Snapshot()).To(BeEmpty())
		Expect(tx.Commit(ctx)).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(ConsistOf(frame{
			Key:         n.Key,
			DispatchKey: "dk",
			Seq:         frames[0].Seq,
			Actions: []color.Action{
				color.NewCreateAction(color.CreatePayload{Named: n}),
			},
		}))
	})

	It("Should not broadcast a create that never commits", func(ctx SpecContext) {
		rec := record()
		tx := db.OpenTx()
		n := color.Named{Name: "Breaker", Light: red}
		Expect(svc.NewWriter(tx).Create(ctx, &n)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(rec.Snapshot()).To(BeEmpty())
	})

	Describe("Apply", func() {
		var n color.Named
		BeforeEach(func(ctx SpecContext) {
			n = color.Named{Name: "Breaker", Light: red}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &n)
			})).To(Succeed())
		})
		apply := func(ctx context.Context, acts ...color.Action) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					WithDispatchKey("dk").
					Apply(ctx, n.Key, acts...)
			})
		}

		It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
			rec := record()
			acts := []color.Action{rename("Valve")}
			Expect(apply(ctx, acts...)).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(ConsistOf(frame{
				Key:         n.Key,
				DispatchKey: "dk",
				Seq:         frames[0].Seq,
				Actions:     acts,
			}))
		})

		It("Should broadcast nothing when an action fails", func(ctx SpecContext) {
			rec := record()
			Expect(apply(ctx, rename("Valve"), rename(""))).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("name: required")),
			))
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})
})
