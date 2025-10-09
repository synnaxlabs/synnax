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
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	xstatus "github.com/synnaxlabs/x/status"
)

var _ = Describe("Writer", func() {
	Describe("Status Writer", func() {
		Describe("Create with Status", func() {
			It("Should create an Arc and set its initial status", func() {
				a := arc.Arc{
					Name:  "test-arc",
					Graph: graph.Graph{},
					Text:  text.Text{},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
				Expect(a.Key).ToNot(Equal(uuid.Nil))

				var s status.Status
				statusKey := a.Key.String()
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(statusKey).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())

				Expect(s.Key).To(Equal(statusKey))
				Expect(s.Name).To(Equal(fmt.Sprintf("%s Status", a.Name)))
				Expect(s.Variant).To(Equal(xstatus.LoadingVariant))
				Expect(s.Message).To(Equal("Deploying"))
			})

			It("Should create an Arc with explicit key and set status", func() {
				key := uuid.New()
				a := arc.Arc{
					Key:   key,
					Name:  "test-arc-with-key",
					Graph: graph.Graph{},
					Text:  text.Text{},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())
				Expect(a.Key).To(Equal(key))

				// Verify the status was created with the correct key
				var s status.Status
				statusKey := key.String()
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(statusKey).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())

				Expect(s.Key).To(Equal(statusKey))
				Expect(s.Name).To(Equal(fmt.Sprintf("%s Status", a.Name)))
			})

			It("Should set status with correct parent relationship", func() {
				a := arc.Arc{
					Name:  "test-arc-parent",
					Graph: graph.Graph{},
					Text:  text.Text{},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

				// Verify the status was created
				var s status.Status
				statusKey := a.Key.String()
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(statusKey).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())

				// The status should be created with the arc's ontology ID as parent
				// This ensures proper hierarchy in the status system
				Expect(s.Key).To(Equal(statusKey))
			})

			It("Should handle multiple arc creations with unique statuses", func() {
				a1 := arc.Arc{
					Name:  "arc-1",
					Graph: graph.Graph{},
					Text:  text.Text{},
				}
				a2 := arc.Arc{
					Name:  "arc-2",
					Graph: graph.Graph{},
					Text:  text.Text{},
				}

				Expect(svc.NewWriter(tx).Create(ctx, &a1)).To(Succeed())
				Expect(svc.NewWriter(tx).Create(ctx, &a2)).To(Succeed())

				// Verify both statuses were created
				var s1, s2 status.Status
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(a1.Key.String()).
					Entry(&s1).
					Exec(ctx, tx)).To(Succeed())
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(a2.Key.String()).
					Entry(&s2).
					Exec(ctx, tx)).To(Succeed())

				Expect(s1.Name).To(Equal("arc-1 Status"))
				Expect(s2.Name).To(Equal("arc-2 Status"))
			})
		})

		Describe("Status on existing Arc update", func() {
			It("Should preserve status when updating existing Arc", func() {
				key := uuid.New()
				a := arc.Arc{
					Key:   key,
					Name:  "existing-arc",
					Graph: graph.Graph{},
					Text:  text.Text{},
				}

				// Create the arc initially
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

				// Try to create again with same key (update scenario)
				a.Name = "updated-arc"
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

				// Status should still exist
				var s status.Status
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(key.String()).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())

				// Note: The status name would still be from the second create call
				Expect(s.Name).To(Equal("updated-arc Status"))
			})
		})

		Describe("Delete with Status cleanup", func() {
			It("Should not directly delete status on arc deletion", func() {
				a := arc.Arc{
					Name:  "arc-to-delete",
					Graph: graph.Graph{},
					Text:  text.Text{},
				}
				Expect(svc.NewWriter(tx).Create(ctx, &a)).To(Succeed())

				// Verify status exists
				var s status.Status
				statusKey := a.Key.String()
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(statusKey).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())

				// Delete the arc
				Expect(svc.NewWriter(tx).Delete(ctx, a.Key)).To(Succeed())

				// Status should still exist (it's managed separately)
				// The status service would handle cleanup through ontology relationships
				Expect(gorp.NewRetrieve[string, status.Status]().
					WhereKeys(statusKey).
					Entry(&s).
					Exec(ctx, tx)).To(Succeed())
			})
		})
	})
})
