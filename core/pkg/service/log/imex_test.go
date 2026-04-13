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
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	. "github.com/synnaxlabs/x/testutil"
)

func loadEnvelope(path string) imex.Envelope {
	data := MustSucceed(os.ReadFile(path))
	var env imex.Envelope
	Expect(json.Unmarshal(data, &env)).To(Succeed())
	return env
}

var _ = Describe("ImportExport", func() {
	Describe("Import", func() {
		It("Should import a v1 envelope", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			Expect(svc.Import(ctx, tx, workspace.OntologyID(ws.Key), env)).To(Succeed())
		})

		It("Should import and migrate a v0 envelope", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v0.json")
			Expect(svc.Import(ctx, tx, workspace.OntologyID(ws.Key), env)).To(Succeed())
		})

		It("Should treat a future version as current", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			env.Version = 999
			Expect(svc.Import(ctx, tx, workspace.OntologyID(ws.Key), env)).To(Succeed())
		})

		It("Should reject invalid data", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_bad_data.json")
			Expect(svc.Import(ctx, tx, workspace.OntologyID(ws.Key), env)).To(HaveOccurred())
		})
	})

	Describe("Export", func() {
		It("Should export an existing log", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v1.json")
			Expect(svc.Import(ctx, tx, workspace.OntologyID(ws.Key), env)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, tx, env.Key))
			Expect(exported.Type).To(Equal("log"))
			Expect(exported.Name).To(Equal(env.Name))
			Expect(exported.Key).To(Equal(env.Key))
		})
	})

	Describe("Round-trip", func() {
		It("Should import v0 and export as v1", func(ctx SpecContext) {
			env := loadEnvelope("testdata/import_v0.json")
			Expect(svc.Import(ctx, tx, workspace.OntologyID(ws.Key), env)).To(Succeed())
			exported := MustSucceed(svc.Export(ctx, tx, env.Key))
			channels, ok := exported.Data["channels"].([]any)
			Expect(ok).To(BeTrue())

			Expect(channels).To(HaveLen(3))
		})
	})
})
