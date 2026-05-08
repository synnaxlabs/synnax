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

func loadEnvelope(path string) imex.Envelope {
	raw := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(raw, &env)).To(Succeed())
	return env
}

var _ = Describe("ImportExport", func() {
	Describe("Import", func() {
		It("Should import a v1 envelope", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			Expect(svc.Import(ctx, tx, env)).Error().NotTo(HaveOccurred())
		})

		It("Should import and migrate a v0 envelope", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v0.json")
			Expect(svc.Import(ctx, tx, env)).Error().NotTo(HaveOccurred())
		})

		It("Should reject a future version", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			env.Version = 999
			Expect(svc.Import(ctx, tx, env)).Error().To(
				MatchError(ContainSubstring("newer than this Core supports")),
			)
		})

		It("Should reject invalid data with a user-friendly error", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_bad_data.json")
			Expect(svc.Import(ctx, tx, env)).Error().To(
				MatchError(ContainSubstring("channels")),
			)
		})
	})

	Describe("Export", func() {
		It("Should export an existing log", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			key := MustSucceed(svc.Import(ctx, tx, env))
			Expect(tx.Commit(ctx)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, key))
			Expect(exported.Type).To(Equal("log"))
			Expect(exported.Name).To(Equal(env.Name))
			Expect(exported.Version).To(Equal(v1.Version))
		})
	})

	Describe("Round-trip", func() {
		It("Should import v0 and export as v1", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v0.json")
			key := MustSucceed(svc.Import(ctx, tx, env))
			Expect(tx.Commit(ctx)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, key))
			Expect(exported.Data).To(HaveKey("channels"))
			Expect(exported.Data["channels"]).To(HaveLen(3))
		})
	})
})
