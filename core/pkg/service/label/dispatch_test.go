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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Dispatch", func() {
	type frame = actions.Scoped[label.Key, label.Action]
	var (
		db      *gorp.DB
		svc     *label.Service
		nominal label.Label
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		svc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB: db,
		}))
		nominal = label.Label{
			Name:  "Nominal",
			Color: MustSucceed(color.Parse("#00ff00")),
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &nominal)
		})).To(Succeed())
	})
	dispatch := func(
		ctx context.Context,
		key label.Key,
		dispatchKey string,
		acts []label.Action,
	) error {
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).
				WithDispatchKey(dispatchKey).
				Apply(ctx, key, acts...)
		})
	}
	record := func() *Recorder[label.Key, label.Action] {
		rec := &Recorder[label.Key, label.Action]{}
		DeferCleanup(svc.Actions().OnChange(rec.Record))
		return rec
	}
	set := func(path []string, value any) label.Action {
		return label.NewSetAction(label.SetPayload{Path: path, Value: value})
	}
	retrieve := func(ctx context.Context) label.Label {
		GinkgoHelper()
		var res label.Label
		Expect(svc.NewRetrieve().
			Where(label.MatchKeys(nominal.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		return res
	}

	It("Should store the edited name and color", func(ctx SpecContext) {
		Expect(dispatch(ctx, nominal.Key, "dk", []label.Action{
			set([]string{"name"}, "Anomaly"),
			set([]string{"color"}, "#ff0000"),
		})).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(label.Label{
			Key:   nominal.Key,
			Name:  "Anomaly",
			Color: MustSucceed(color.Parse("#ff0000")),
		}))
	})

	It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
		rec := record()
		acts := []label.Action{set([]string{"name"}, "Anomaly")}
		Expect(dispatch(ctx, nominal.Key, "dk-1", acts)).To(Succeed())
		frames := rec.Snapshot()
		Expect(frames).To(HaveLen(1))
		f := frames[0]
		Expect(f.Key).To(Equal(nominal.Key))
		Expect(f.DispatchKey).To(Equal("dk-1"))
		Expect(f.Seq).To(BeNumerically(">", 0))
		Expect(f.Actions).To(Equal(acts))
	})

	It("Should apply none of the actions when one fails", func(ctx SpecContext) {
		rec := record()
		Expect(dispatch(ctx, nominal.Key, "dk", []label.Action{
			set([]string{"name"}, "Anomaly"),
			set([]string{"key"}, uuid.New().String()),
		})).To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx)).To(Equal(nominal))
		Expect(rec.Snapshot()).To(BeEmpty())
	})

	It("Should reject an empty name", func(ctx SpecContext) {
		Expect(dispatch(ctx, nominal.Key, "dk", []label.Action{
			set([]string{"name"}, ""),
		})).To(MatchError(ContainSubstring("name: required")))
		Expect(retrieve(ctx)).To(Equal(nominal))
	})

	It("Should reject a create", func(ctx SpecContext) {
		overwritten := nominal
		overwritten.Name = "Overwritten"
		Expect(dispatch(ctx, nominal.Key, "dk", []label.Action{
			label.NewCreateAction(label.CreatePayload{Label: overwritten}),
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("only the Core dispatches a label create")),
		))
		Expect(retrieve(ctx)).To(Equal(nominal))
	})

	It("Should return not found for a missing label", func(ctx SpecContext) {
		Expect(dispatch(ctx, uuid.New(), "dk", []label.Action{
			set([]string{"name"}, "Anomaly"),
		})).To(MatchError(query.ErrNotFound))
	})

	Describe("Writer", func() {
		It("Should broadcast a create with the dispatch key after the commit", func(
			ctx SpecContext,
		) {
			rec := record()
			l := label.Label{
				Name:  "Anomaly",
				Color: MustSucceed(color.Parse("#ff0000")),
			}
			tx := DeferClose(db.OpenTx())
			w := svc.NewWriter(tx).WithDispatchKey("dk-create")
			Expect(w.Create(ctx, &l)).To(Succeed())
			Expect(rec.Snapshot()).To(BeEmpty())
			Expect(tx.Commit(ctx)).To(Succeed())
			frames := rec.Snapshot()
			Expect(frames).To(ConsistOf(frame{
				Key:         l.Key,
				DispatchKey: "dk-create",
				Seq:         frames[0].Seq,
				Actions: []label.Action{
					label.NewCreateAction(label.CreatePayload{Label: l}),
				},
			}))
		})

		It("Should not broadcast a create that never commits", func(ctx SpecContext) {
			rec := record()
			tx := db.OpenTx()
			l := label.Label{Name: "Anomaly"}
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(rec.Snapshot()).To(BeEmpty())
		})
	})
})
