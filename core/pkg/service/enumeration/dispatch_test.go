// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	xcolor "github.com/synnaxlabs/x/color"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Broadcasts", func() {
	type frame = actions.Scoped[enumeration.Key, enumeration.Action]
	var (
		db     *gorp.DB
		svc    *enumeration.Service
		colors *color.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups := MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		colors = MustOpen(
			color.OpenService(ctx, color.ServiceConfig{DB: db, Group: groups}),
		)
		svc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
	})
	record := func() *Recorder[enumeration.Key, enumeration.Action] {
		rec := &Recorder[enumeration.Key, enumeration.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	rename := func(name string) enumeration.Action {
		return enumeration.NewSetAction(
			enumeration.SetPayload{Path: []string{"name"}, Value: name},
		)
	}

	It("Should broadcast a create with the dispatch key after the commit", func(
		ctx SpecContext,
	) {
		rec := record()
		e := enumeration.Enumeration{Name: "Breaker"}
		tx := DeferClose(db.OpenTx())
		Expect(
			createOne(ctx, svc.NewWriter(tx).WithDispatchKey("dk"), &e),
		).To(Succeed())
		Expect(rec.Snapshot()).To(BeEmpty())
		Expect(tx.Commit(ctx)).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(ConsistOf(frame{
			Key:         e.Key,
			DispatchKey: "dk",
			Seq:         frames[0].Seq,
			Actions: []enumeration.Action{
				enumeration.NewCreateAction(enumeration.CreatePayload{Enumeration: e}),
			},
		}))
	})

	It("Should not broadcast a create that never commits", func(ctx SpecContext) {
		rec := record()
		tx := db.OpenTx()
		e := enumeration.Enumeration{Name: "Breaker"}
		Expect(createOne(ctx, svc.NewWriter(tx), &e)).To(Succeed())
		Expect(tx.Close()).To(Succeed())
		Expect(rec.Snapshot()).To(BeEmpty())
	})

	Describe("Apply", func() {
		var e enumeration.Enumeration
		BeforeEach(func(ctx SpecContext) {
			e = enumeration.Enumeration{Name: "Breaker"}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return createOne(ctx, svc.NewWriter(tx), &e)
			})).To(Succeed())
		})
		apply := func(ctx context.Context, acts ...enumeration.Action) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					WithDispatchKey("dk").
					Apply(ctx, e.Key, acts...)
			})
		}

		It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
			rec := record()
			acts := []enumeration.Action{rename("Valve")}
			Expect(apply(ctx, acts...)).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(ConsistOf(frame{
				Key:         e.Key,
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

	Describe("Named color delete", func() {
		It("Should inline the light value and broadcast the enumeration", func(
			ctx SpecContext,
		) {
			red := MustSucceed(xcolor.Parse("#ff0000"))
			trip := color.Named{Name: "Trip", Light: red}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return colors.NewWriter(tx).Create(ctx, &trip)
			})).To(Succeed())
			named := color.Ref{Variant: color.NamedRef{Key: trip.Key}}
			e := enumeration.Enumeration{
				Name: "Breaker",
				Entries: []enumeration.Entry{
					{Key: "closed", Value: 0, Name: "Closed"},
					{Key: "tripped", Value: 1, Name: "Tripped", Color: &named},
				},
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return createOne(ctx, svc.NewWriter(tx), &e)
			})).To(Succeed())
			rec := record()
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return colors.NewWriter(tx).Delete(ctx, trip.Key)
			})).To(Succeed())
			fixed := color.Ref{Variant: color.FixedRef{Value: red}}
			expected := e
			expected.Entries = []enumeration.Entry{
				e.Entries[0],
				{Key: "tripped", Value: 1, Name: "Tripped", Color: &fixed},
			}
			var stored enumeration.Enumeration
			Expect(svc.NewRetrieve().
				Where(enumeration.MatchKeys(e.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored).To(Equal(expected))
			frames := rec.Snapshot()
			Expect(frames).To(ConsistOf(frame{
				Key: e.Key,
				Seq: frames[0].Seq,
				Actions: []enumeration.Action{enumeration.NewCreateAction(
					enumeration.CreatePayload{Enumeration: expected},
				)},
			}))
		})
	})
})
