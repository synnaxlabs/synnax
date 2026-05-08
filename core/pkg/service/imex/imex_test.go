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
	. "github.com/synnaxlabs/x/testutil"
)

func logV1Data() map[string]any {
	return map[string]any{
		"channels":               []any{},
		"remote_created":         false,
		"timestamp_precision":    0,
		"show_channel_names":     true,
		"show_receipt_timestamp": true,
	}
}

var _ = Describe("Service", func() {
	Describe("Import", func() {
		It("Should route to the correct service by type", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: v1.Version,
				Type:    "log",
				Name:    "Registry Test",
				Data:    logV1Data(),
			}}
			keys := MustSucceed(svc.Import(ctx, envs))
			Expect(keys).To(HaveLen(1))
			Expect(keys[0]).NotTo(BeEmpty())
		})

		It("Should reject an unregistered type", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: v1.Version,
				Type:    "nonexistent",
				Name:    "Bad Type",
				Data:    map[string]any{},
			}}
			Expect(svc.Import(ctx, envs)).Error().To(
				MatchError(ContainSubstring("no importer registered")),
			)
		})

		It("Should roll back the transaction if any import fails", func(ctx SpecContext) {
			envs := []imex.Envelope{
				{
					Version: v1.Version,
					Type:    "log",
					Name:    "Good Log",
					Data:    logV1Data(),
				},
				{
					Version: 99999,
					Type:    "nonexistent",
					Name:    "Bad Type",
					Data:    map[string]any{},
				},
			}
			Expect(svc.Import(ctx, envs)).Error().To(
				MatchError(ContainSubstring("no importer registered")),
			)
		})
	})

	Describe("Export", func() {
		It("Should export a registered resource", func(ctx SpecContext) {
			envs := []imex.Envelope{{
				Version: v1.Version,
				Type:    "log",
				Name:    "Export Registry Test",
				Data:    logV1Data(),
			}}
			keys := MustSucceed(svc.Import(ctx, envs))
			Expect(keys).To(HaveLen(1))
			result := MustSucceed(svc.Export(ctx, []ontology.ID{{
				Type: ontology.ResourceTypeLog,
				Key:  keys[0],
			}}))
			Expect(result).To(HaveLen(1))
			Expect(result[0].Version).To(Equal(v1.Version))
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
