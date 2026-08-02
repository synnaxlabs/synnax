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

// Match claims typeless envelopes carrying the legacy_marker key so specs can exercise
// type resolution ahead of access control.
func (*recordingImporter) Match(body map[string]any) bool {
	_, ok := body["legacy_marker"]
	return ok
}

func testEnvelope(name string) apiimex.ImportRequest {
	var env imex.Envelope
	Expect(json.Unmarshal(fmt.Appendf(nil,
		`{"version":1,"type":%q,"name":%q}`, ontology.ResourceTypeChannel, name,
	), &env)).To(Succeed())
	return env
}

var _ = Describe("Import", func() {
	It("Should hand the file_name and parent params to the importer", func(ctx SpecContext) {
		key := uuid.New()
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(
			`{"file_name":"Metrics Log.json","parent":"project:%s"}`, key,
		))
		id := MustSucceed(apiSvc.Import(fctx, db, testEnvelope("with-options")))
		Expect(id.Key).To(Equal("with-options"))
		Expect(importer.opts.FileName).To(Equal("Metrics Log.json"))
		Expect(importer.opts.Parent).To(Equal(ontology.ID{
			Type: ontology.ResourceTypeProject, Key: key.String(),
		}))
	})

	It("Should resolve a typeless envelope through the registered matcher", func(ctx SpecContext) {
		var env imex.Envelope
		Expect(json.Unmarshal(
			[]byte(`{"version":"1.0.0","legacy_marker":true}`), &env,
		)).To(Succeed())
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(
			`{"file_name":"Legacy State.json","parent":"project:%s"}`, uuid.New(),
		))
		id := MustSucceed(apiSvc.Import(fctx, db, env))
		Expect(id.Key).To(Equal("Legacy State"))
	})

	It("Should reject a typeless envelope no matcher claims", func(ctx SpecContext) {
		var env imex.Envelope
		Expect(json.Unmarshal(
			[]byte(`{"version":"1.0.0","unclaimed":true}`), &env,
		)).To(Succeed())
		fctx := rootCtx(ctx)
		fctx.Set("params", fmt.Sprintf(
			`{"file_name":"Mystery.json","parent":"project:%s"}`, uuid.New(),
		))
		Expect(apiSvc.Import(fctx, db, env)).Error().To(
			MatchError(ContainSubstring("does not match any known resource format")),
		)
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
		fctx.Set("params", fmt.Sprintf(`{"parent":"project:%s"}`, uuid.New()))
		Expect(apiSvc.Import(fctx, db, testEnvelope("no-file-name"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("file_name")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject params without a parent", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", `{"file_name":"Metrics Log.json"}`)
		Expect(apiSvc.Import(fctx, db, testEnvelope("no-parent"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("parent")),
				MatchError(ContainSubstring("required")),
			))
	})

	It("Should reject a parent with an empty key", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", `{"file_name":"Metrics Log.json","parent":"project:"}`)
		Expect(apiSvc.Import(fctx, db, testEnvelope("empty-parent-key"))).Error().
			To(SatisfyAll(
				MatchError(ContainSubstring("parent")),
				MatchError(ContainSubstring("must carry a non-empty key")),
			))
	})

	It("Should reject a parent that is not a valid ontology ID", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", `{"file_name":"Metrics Log.json","parent":"no-colon"}`)
		Expect(apiSvc.Import(fctx, db, testEnvelope("bad-parent"))).Error().To(SatisfyAll(
			MatchError(ContainSubstring("failed to parse id: no-colon")),
			MatchError(ContainSubstring("validation error")),
		))
	})

	It("Should reject params that are not valid JSON", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("params", "not-json")
		Expect(apiSvc.Import(fctx, db, testEnvelope("bad-params"))).Error().To(SatisfyAll(
			MatchError(ContainSubstring("invalid params")),
			MatchError(ContainSubstring("validation error")),
		))
	})
})
