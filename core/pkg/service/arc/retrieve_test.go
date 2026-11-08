// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/directory"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Retrieve", func() {
	Describe("Arc Retrieve", func() {
		It("Should retrieve an Arc", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			a := newAutomation(arc.Base{
				Name:   "test-retrieve",
				Source: arc.Source{Variant: arc.TextSource{Text: text.Text{}}},
			})
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&retrievedArc).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(retrievedArc.Base().Key).To(Equal(a.Base().Key))
			Expect(retrievedArc.Base().Name).To(Equal(a.Base().Name))
		})

		It("Should retrieve multiple Arcs", func(ctx SpecContext) {
			tx := DeferClose(db.OpenTx())
			arcs := []arc.Arc{
				newAutomation(arc.Base{
					Name:   "arc-multi-1",
					Source: arc.Source{Variant: arc.TextSource{Text: text.Text{}}},
				}),
				newAutomation(arc.Base{
					Name:   "arc-multi-2",
					Source: arc.Source{Variant: arc.TextSource{Text: text.Text{}}},
				}),
				newAutomation(arc.Base{
					Name:   "arc-multi-3",
					Source: arc.Source{Variant: arc.TextSource{Text: text.Text{}}},
				}),
			}

			keys := make([]uuid.UUID, 0, len(arcs))
			for i := range arcs {
				Expect(svc.NewWriter(tx).Create(ctx, &arcs[i])).To(Succeed())
				keys = append(keys, arcs[i].Base().Key)
			}

			var retrievedArcs []arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(keys...)).
					Entries(&retrievedArcs).
					Exec(ctx, tx),
			).To(Succeed())
			Expect(retrievedArcs).To(HaveLen(3))
		})

		It(
			"Should return only the Arcs in the given directories",
			func(ctx SpecContext) {
				tx := DeferClose(db.OpenTx())
				w := svc.NewWriter(tx)
				root := rootDirectory(ctx, tx).Key
				dirs := []directory.Directory{{Name: "lib"}, {Name: "other"}}
				arcs := make([]arc.Arc, 3)
				for i, name := range []string{"in_lib", "in_other", "in_root"} {
					arcs[i] = newModule(
						arc.Base{
							Name:   name,
							Source: arc.Source{Variant: arc.TextSource{}},
						},
					)
					Expect(w.Create(ctx, &arcs[i])).To(Succeed())
				}
				for i := range dirs {
					Expect(w.CreateDirectory(ctx, &dirs[i], root)).To(Succeed())
					Expect(w.Move(ctx, root, dirs[i].Key, arcs[i].ResourceID())).
						To(Succeed())
				}
				var res []arc.Arc
				Expect(svc.NewRetrieve().
					Where(arc.MatchDirectories(dirs[0].Key, dirs[1].Key)).
					Entries(&res).
					Exec(ctx, tx)).To(Succeed())
				Expect(res).To(ConsistOf(
					WithTransform(arc.Arc.Base, HaveField("Name", "in_lib")),
					WithTransform(arc.Arc.Base, HaveField("Name", "in_other")),
				))
			},
		)

		It("Should retrieve Arc after transaction commit", func(ctx SpecContext) {
			localTx := db.OpenTx()
			a := newAutomation(arc.Base{
				Name:   "tx-test-arc",
				Source: arc.Source{Variant: arc.TextSource{Text: text.Text{}}},
			})
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			newTx := db.OpenTx()

			var retrievedArc arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&retrievedArc).
					Exec(ctx, newTx),
			).To(Succeed())
			Expect(retrievedArc.Base().Name).To(Equal("tx-test-arc"))
			Expect(newTx.Close()).To(Succeed())
		})

		It("Should retrieve Arc without transaction", func(ctx SpecContext) {
			localTx := db.OpenTx()
			a := newAutomation(arc.Base{
				Name:   "no-tx-arc",
				Source: arc.Source{Variant: arc.TextSource{Text: text.Text{}}},
			})
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			var retrievedArc arc.Arc
			Expect(
				svc.NewRetrieve().
					Where(arc.MatchKeys(a.Base().Key)).
					Entry(&retrievedArc).
					Exec(ctx, nil),
			).To(Succeed())
			Expect(retrievedArc.Base().Name).To(Equal("no-tx-arc"))
		})
	})
})
