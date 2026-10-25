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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
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
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
		}))
		svc = MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		}))
		nominal = label.Label{Name: "Nominal", Color: color.MustFromHex("#00ff00")}
		Expect(svc.NewWriter(nil).Create(ctx, &nominal)).To(Succeed())
	})
	collect := func() *[]frame {
		GinkgoHelper()
		var frames []frame
		DeferCleanup(svc.ObserveActions().OnChange(func(_ context.Context, f frame) {
			frames = append(frames, f)
		}))
		return &frames
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
			Exec(ctx, nil)).To(Succeed())
		return res
	}

	It("Should store the edited name and color", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, nominal.Key, "dk", []label.Action{
			set([]string{"name"}, "Anomaly"),
			set([]string{"color"}, "#ff0000"),
		})).To(Succeed())
		Expect(retrieve(ctx)).To(Equal(label.Label{
			Key:   nominal.Key,
			Name:  "Anomaly",
			Color: color.MustFromHex("#ff0000"),
		}))
	})

	It("Should broadcast the actions with the dispatch key", func(ctx SpecContext) {
		frames := collect()
		acts := []label.Action{set([]string{"name"}, "Anomaly")}
		Expect(svc.Dispatch(ctx, nominal.Key, "dk-1", acts)).To(Succeed())
		Expect(*frames).To(HaveLen(1))
		f := (*frames)[0]
		Expect(f.Key).To(Equal(nominal.Key))
		Expect(f.DispatchKey).To(Equal("dk-1"))
		Expect(f.Seq).To(BeNumerically(">", 0))
		Expect(f.Actions).To(Equal(acts))
	})

	It("Should apply none of the actions when one fails", func(ctx SpecContext) {
		frames := collect()
		Expect(svc.Dispatch(ctx, nominal.Key, "dk", []label.Action{
			set([]string{"name"}, "Anomaly"),
			set([]string{"key"}, uuid.New().String()),
		})).To(MatchError(validate.ErrValidation))
		Expect(retrieve(ctx)).To(Equal(nominal))
		Expect(*frames).To(BeEmpty())
	})

	It("Should reject an empty name", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, nominal.Key, "dk", []label.Action{
			set([]string{"name"}, ""),
		})).To(MatchError(ContainSubstring("name: required")))
		Expect(retrieve(ctx)).To(Equal(nominal))
	})

	It("Should reject a create", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, nominal.Key, "dk", []label.Action{
			label.NewCreateAction(label.CreatePayload{Label: nominal}),
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("only the Core dispatches a label create")),
		))
	})

	It("Should return not found for a missing label", func(ctx SpecContext) {
		Expect(svc.Dispatch(ctx, uuid.New(), "dk", []label.Action{
			set([]string{"name"}, "Anomaly"),
		})).To(MatchError(query.ErrNotFound))
	})

	Describe("Writer", func() {
		It("Should broadcast a create with the dispatch key after the commit", func(
			ctx SpecContext,
		) {
			frames := collect()
			l := label.Label{Name: "Anomaly", Color: color.MustFromHex("#ff0000")}
			tx := db.OpenTx()
			DeferCleanup(func() { Expect(tx.Close()).To(Succeed()) })
			w := svc.NewWriter(tx).WithDispatchKey("dk-create")
			Expect(w.Create(ctx, &l)).To(Succeed())
			Expect(*frames).To(BeEmpty())
			Expect(tx.Commit(ctx)).To(Succeed())
			Expect(*frames).To(ConsistOf(frame{
				Key:         l.Key,
				DispatchKey: "dk-create",
				Seq:         (*frames)[0].Seq,
				Actions: []label.Action{
					label.NewCreateAction(label.CreatePayload{Label: l}),
				},
			}))
		})

		It("Should not broadcast a create that never commits", func(ctx SpecContext) {
			frames := collect()
			tx := db.OpenTx()
			l := label.Label{Name: "Anomaly"}
			Expect(svc.NewWriter(tx).Create(ctx, &l)).To(Succeed())
			Expect(tx.Close()).To(Succeed())
			Expect(*frames).To(BeEmpty())
		})
	})
})
