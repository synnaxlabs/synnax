// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
	. "github.com/synnaxlabs/x/testutil"
)

func loadPayload(path string) imex.ImportPayload {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	payload := imex.ImportPayload{Version: env.Version, Name: env.Name}
	Expect(json.Unmarshal(env.Data, &payload.Data)).To(Succeed())
	return payload
}

var _ = Describe("ImportExport", func() {
	Describe("Import", func() {
		It("Should import a v1 envelope", func(ctx SpecContext) {
			payload := loadPayload("testdata/import_v1.json")
			Expect(svc.Import(ctx, tx, payload)).Error().NotTo(HaveOccurred())
		})

		It("Should import and migrate a v0 envelope", func(ctx SpecContext) {
			payload := loadPayload("testdata/import_v0.json")
			Expect(svc.Import(ctx, tx, payload)).Error().NotTo(HaveOccurred())
		})

		It("Should reject a future version", func(ctx SpecContext) {
			payload := loadPayload("testdata/import_v1.json")
			payload.Version = 999
			Expect(svc.Import(ctx, tx, payload)).Error().To(
				MatchError(ContainSubstring("newer than this Core supports")),
			)
		})

		It("Should reject invalid data with a user-friendly error", func(ctx SpecContext) {
			payload := loadPayload("testdata/import_bad_data.json")
			Expect(svc.Import(ctx, tx, payload)).Error().To(
				MatchError(ContainSubstring("channels")),
			)
		})
	})

	Describe("Export", func() {
		It("Should export an existing log", func(ctx SpecContext) {
			payload := loadPayload("testdata/import_v1.json")
			key := MustSucceed(svc.Import(ctx, tx, payload))
			Expect(tx.Commit(ctx)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, key))
			Expect(exported.Type).To(Equal("log"))
			Expect(exported.Name).To(Equal(payload.Name))
			Expect(exported.Key).To(Equal(key))
			Expect(exported.Version).To(Equal(v1.Version))
		})
	})

	Describe("Round-trip", func() {
		It("Should import v0 and export as v1", func(ctx SpecContext) {
			payload := loadPayload("testdata/import_v0.json")
			key := MustSucceed(svc.Import(ctx, tx, payload))
			Expect(tx.Commit(ctx)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, key))
			var d v1.Data
			Expect(json.Unmarshal(exported.Data, &d)).To(Succeed())
			Expect(d.Channels).To(HaveLen(3))
		})
	})
})
