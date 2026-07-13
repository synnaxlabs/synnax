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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/project"
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
		fctx.Set("file_name", "Metrics Log.json")
		fctx.Set("project", project.OntologyID(key).String())
		id := MustSucceed(apiSvc.Import(fctx, db, testEnvelope("with-options")))
		Expect(id.Key).To(Equal("with-options"))
		Expect(importer.opts.FileName).To(Equal("Metrics Log.json"))
		Expect(importer.opts.Project).To(Equal(key))
	})

	It("Should import with zero options when no params are set", func(ctx SpecContext) {
		MustSucceed(apiSvc.Import(rootCtx(ctx), db, testEnvelope("no-options")))
		Expect(importer.opts).To(Equal(imex.ImportOptions{}))
	})

	It("Should treat an empty project param as no project", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("project", "")
		MustSucceed(apiSvc.Import(fctx, db, testEnvelope("empty-project")))
		Expect(importer.opts.Project).To(Equal(uuid.Nil))
	})

	It("Should reject a project param that does not reference a project", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("project", "group:"+uuid.NewString())
		Expect(apiSvc.Import(fctx, db, testEnvelope("wrong-type"))).Error().To(SatisfyAll(
			MatchError(ContainSubstring("can only be imported under a project")),
			MatchError(ContainSubstring("validation error")),
		))
	})

	It("Should reject a project param whose key is not a valid UUID", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("project", "project:not-a-uuid")
		Expect(apiSvc.Import(fctx, db, testEnvelope("bad-key"))).Error().To(
			MatchError(ContainSubstring("invalid project key")),
		)
	})

	It("Should reject a malformed project param", func(ctx SpecContext) {
		fctx := rootCtx(ctx)
		fctx.Set("project", "not-an-ontology-id")
		Expect(apiSvc.Import(fctx, db, testEnvelope("malformed"))).Error().To(
			MatchError(ContainSubstring("failed to parse id")),
		)
	})
})
