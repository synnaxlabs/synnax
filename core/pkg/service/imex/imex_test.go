// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Service", func() {
	Describe("Import", func() {
		It("Should route to the correct service by type", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: v1.Version,
				Type:    "log",
				Key:     "110e8400-e29b-41d4-a716-446655440000",
				Name:    "Registry Test",
				Data: map[string]any{
					"channels":               []any{},
					"remote_created":         false,
					"timestamp_precision":    0.0,
					"show_channel_names":     true,
					"show_receipt_timestamp": true,
				},
			}}
			Expect(svc.Import(ctx, workspace.OntologyID(ws.Key), envs)).To(Succeed())
		})

		It("Should reject an unregistered type", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: v1.Version,
				Type:    "nonexistent",
				Key:     "220e8400-e29b-41d4-a716-446655440000",
				Name:    "Bad Type",
				Data:    map[string]any{},
			}}
			Expect(svc.Import(
				ctx, workspace.OntologyID(ws.Key), envs,
			)).To(MatchError(ContainSubstring("no importer registered")))
		})

		It("Should roll back the transaction if any import fails", func(ctx SpecContext) {
			envs := []imex.Envelope{
				{
					Version: v1.Version,
					Type:    "log",
					Key:     "330e8400-e29b-41d4-a716-446655440000",
					Name:    "Good Log",
					Data: map[string]any{
						"channels":               []any{},
						"remote_created":         false,
						"timestamp_precision":    0.0,
						"show_channel_names":     true,
						"show_receipt_timestamp": true,
					},
				},
				{
					Version: 99999,
					Type:    "log",
					Key:     "440e8400-e29b-41d4-a716-446655440000",
					Name:    "Bad Version",
					Data:    map[string]any{},
				},
			}
			Expect(svc.Import(ctx, workspace.OntologyID(ws.Key), envs)).To(HaveOccurred())
			Expect(svc.Export(ctx, []ontology.ID{{
				Type: ontology.ResourceTypeLog,
				Key:  "330e8400-e29b-41d4-a716-446655440000",
			}})).Error().To(HaveOccurred())
		})
	})

	Describe("Export", func() {
		It("Should export a registered resource", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: v1.Version,
				Type:    "log",
				Key:     "550e8400-e29b-41d4-a716-446655440001",
				Name:    "Export Registry Test",
				Data: map[string]any{
					"channels":               []any{},
					"remote_created":         false,
					"timestamp_precision":    0.0,
					"show_channel_names":     true,
					"show_receipt_timestamp": true,
				},
			}}
			Expect(svc.Import(ctx, workspace.OntologyID(ws.Key), envs)).To(Succeed())
			result := MustSucceed(svc.Export(ctx, []ontology.ID{{
				Type: ontology.ResourceTypeLog,
				Key:  "550e8400-e29b-41d4-a716-446655440001",
			}}))
			Expect(result).To(HaveLen(1))
			Expect(result[0].Version).To(Equal(54))
			Expect(result[0].Name).To(Equal("Export Registry Test"))
		})

		It("Should reject an unregistered type", func(ctx SpecContext) {
			Expect(svc.Export(ctx, []ontology.ID{{
				Type: "nonexistent",
				Key:  "660e8400-e29b-41d4-a716-446655440000",
			}})).Error().To(MatchError(ContainSubstring("no exporter registered")))
		})
	})
})
