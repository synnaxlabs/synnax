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
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

// recordingImporter captures the ImportOptions the API layer hands to the registry so
// specs can assert on how the out-of-band request params were parsed.
type recordingImporter struct {
	opts imex.ImportOptions
}

func (*recordingImporter) Type() ontology.ResourceType {
	return ontology.ResourceTypeChannel
}

func (r *recordingImporter) Import(
	_ context.Context, _ gorp.Tx, env imex.Envelope, opts imex.ImportOptions,
) (ontology.ID, error) {
	r.opts = opts
	return ontology.ID{Type: ontology.ResourceTypeChannel, Key: env.Name}, nil
}

func testEnvelope(name string) apiimex.ImportRequest {
	var env imex.Envelope
	Expect(json.Unmarshal(fmt.Appendf(nil,
		`{"version":1,"type":%q,"name":%q}`, ontology.ResourceTypeChannel, name,
	), &env)).To(Succeed())
	return env
}

var _ = Describe("Import", func() {
	It("Should hand the file_name and project params to the importer", func(ctx SpecContext) {
		key := uuid.New()
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(
			`{"file_name":"Metrics Log.json","project":%q}`, key,
		))
		id := MustSucceed(apiSvc.Import(fctx, db, testEnvelope("with-options")))
		Expect(id.Key).To(Equal("with-options"))
		Expect(importer.opts.FileName).To(Equal("Metrics Log.json"))
		Expect(importer.opts.Project).To(Equal(key))
	})

	It("Should reject a request with no params", func(ctx SpecContext) {
		Expect(apiSvc.Import(rootCtx(ctx), db, testEnvelope("no-params"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("params")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject empty params", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", "")
		Expect(apiSvc.Import(fctx, db, testEnvelope("empty-params"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("params")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject params without a file_name", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(`{"project":%q}`, uuid.New()))
		Expect(apiSvc.Import(fctx, db, testEnvelope("no-file-name"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("file_name")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject params without a project", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", `{"file_name":"Metrics Log.json"}`)
		Expect(apiSvc.Import(fctx, db, testEnvelope("no-project"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("project")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject a zero project key", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(
			`{"file_name":"Metrics Log.json","project":%q}`, uuid.Nil,
		))
		Expect(apiSvc.Import(fctx, db, testEnvelope("zero-project"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("project")),
				MatchError(ContainSubstring("must be non-zero")),
			))
	})

	It("Should reject a project param that is not a valid UUID", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", `{"file_name":"Metrics Log.json","project":"not-a-uuid"}`)
		Expect(apiSvc.Import(fctx, db, testEnvelope("bad-key"))).Error().To(SatisfyAll(
			MatchError(ContainSubstring("invalid project key")),
			MatchError(ContainSubstring("validation error")),
		))
	})

	It("Should reject params that are not valid JSON", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", "not-json")
		Expect(apiSvc.Import(fctx, db, testEnvelope("bad-params"))).Error().To(SatisfyAll(
			MatchError(ContainSubstring("params must be a valid JSON object")),
			MatchError(ContainSubstring("validation error")),
		))
	})
})
