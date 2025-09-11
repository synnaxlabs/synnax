// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
)

var _ = Describe("Retrieve", func() {
	Describe("Status Retrieve", func() {
		It("Should retrieve an Arc and its associated status", func() {
			// Create an arc
			a := arc.Arc{
				Name:  "test-retrieve",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			// Retrieve the arc
			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, tx)).To(Succeed())
			Expect(retrievedArc).To(Equal(a))

			// Retrieve the associated status
			var s status.Status
			statusKey := a.Key.String()
			Expect(gorp.NewRetrieve[string, status.Status]().
				WhereKeys(statusKey).
				Entry(&s).
				Exec(ctx, tx)).To(Succeed())
			
			Expect(s.Key).To(Equal(statusKey))
			Expect(s.Name).To(Equal("test-retrieve Status"))
		})

		It("Should retrieve multiple Arcs with their statuses", func() {
			// Create multiple arcs
			arcs := []arc.Arc{
				{Name: "arc-multi-1", Graph: graph.Graph{}, Text: text.Text{}},
				{Name: "arc-multi-2", Graph: graph.Graph{}, Text: text.Text{}},
				{Name: "arc-multi-3", Graph: graph.Graph{}, Text: text.Text{}},
			}

			keys := make([]uuid.UUID, 0, len(arcs))
			for i := range arcs {
				Expect(svc.NewWriter(tx).Create(ctx, &arcs[i])).To(Succeed())
				keys = append(keys, arcs[i].Key)
			}

			// Retrieve all arcs
			var retrievedArcs []arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(keys...).Entries(&retrievedArcs).Exec(ctx, tx)).To(Succeed())
			Expect(retrievedArcs).To(HaveLen(3))

			// Verify each arc has an associated status
			for _, a := range retrievedArcs {
				var s status.Status
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(a.Key.String()).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())
				
				Expect(s.Key).To(Equal(a.Key.String()))
				Expect(s.Name).To(ContainSubstring(a.Name))
				Expect(s.Name).To(ContainSubstring("Status"))
			}
		})

		It("Should handle retrieving non-existent Arc status", func() {
			nonExistentKey := uuid.New()
			
			// Try to retrieve non-existent arc
			var a arc.Arc
			err := svc.NewRetrieve().WhereKeys(nonExistentKey).Entry(&a).Exec(ctx, tx)
			Expect(err).ToNot(HaveOccurred()) // Retrieve returns no error for non-existent keys
			Expect(a.Key).To(Equal(uuid.Nil)) // But the arc should be empty

			// Try to retrieve non-existent status
			var s status.Status
			err = gorp.NewRetrieve[string, status.Status]().
				WhereKeys(nonExistentKey.String()).
				Entry(&s).
				Exec(ctx, tx)
			Expect(err).ToNot(HaveOccurred())
			Expect(s.Key).To(BeEmpty()) // Status should be empty
		})

		It("Should retrieve Arc with status after transaction commit", func() {
			// Create an arc in a transaction
			localTx := db.OpenTx()
			a := arc.Arc{
				Name:  "tx-test-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			// Retrieve the arc in a new transaction
			newTx := db.OpenTx()
			defer newTx.Close()

			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, newTx)).To(Succeed())
			Expect(retrievedArc.Name).To(Equal("tx-test-arc"))

			// Retrieve the status
			var s status.Status
			Expect(gorp.NewRetrieve[string, status.Status]().
				WhereKeys(a.Key.String()).
				Entry(&s).
				Exec(ctx, newTx)).To(Succeed())
			
			Expect(s.Key).To(Equal(a.Key.String()))
			Expect(s.Name).To(Equal("tx-test-arc Status"))
		})

		It("Should retrieve Arc without transaction", func() {
			// Create an arc with direct DB write
			localTx := db.OpenTx()
			a := arc.Arc{
				Name:  "no-tx-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(localTx).Create(ctx, &a)).To(Succeed())
			Expect(localTx.Commit(ctx)).To(Succeed())

			// Retrieve without providing a transaction (uses base DB)
			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, nil)).To(Succeed())
			Expect(retrievedArc.Name).To(Equal("no-tx-arc"))

			// Retrieve status without transaction
			var s status.Status
			Expect(gorp.NewRetrieve[string, status.Status]().
				WhereKeys(a.Key.String()).
				Entry(&s).
				Exec(ctx, db)).To(Succeed())
			
			Expect(s.Key).To(Equal(a.Key.String()))
		})

		It("Should retrieve Arc and verify status metadata integrity", func() {
			a := arc.Arc{
				Name:  "metadata-arc",
				Graph: graph.Graph{},
				Text:  text.Text{},
			}
			Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

			// Retrieve and verify arc
			var retrievedArc arc.Arc
			Expect(svc.NewRetrieve().WhereKeys(a.Key).Entry(&retrievedArc).Exec(ctx, tx)).To(Succeed())

			// Retrieve and verify status has correct metadata
			var s status.Status
			Expect(gorp.NewRetrieve[string, status.Status]().
				WhereKeys(a.Key.String()).
				Entry(&s).
				Exec(ctx, tx)).To(Succeed())

			// Verify status metadata
			Expect(s.Key).To(Equal(a.Key.String()))
			Expect(s.Name).To(Equal("metadata-arc Status"))
			Expect(s.Message).To(Equal("Status created successfully"))
			Expect(s.Time).ToNot(BeZero()) // Time should be set
		})
	})
})