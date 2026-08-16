// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	apiproject "github.com/synnaxlabs/synnax/pkg/api/project"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Export", func() {
	It(
		"Should export the bundle when retrieve is granted on the project and members",
		func(ctx SpecContext) {
			proj := createProject(ctx, "granted")
			l := createLog(ctx, proj.Key, "Metrics")
			grantRetrieveOn(
				ctx,
				author.OntologyID(),
				project.OntologyID(proj.Key),
				log.OntologyID(l.Key),
			)
			Expect(MustSucceed(apiSvc.Export(
				AuthedCtx(ctx, author),
				apiproject.ExportRequest{Key: proj.Key, Encoding: apiimex.EncodingJSON},
			))).To(SatisfyAll(HaveKey("manifest.json"), HaveKey("Metrics.json")))
		},
	)
	It("Should reject an unsupported encoding", func(ctx SpecContext) {
		proj := createProject(ctx, "bad-encoding")
		grantRetrieveOn(ctx, author.OntologyID(), project.OntologyID(proj.Key))
		Expect(apiSvc.Export(
			AuthedCtx(ctx, author),
			apiproject.ExportRequest{Key: proj.Key, Encoding: "YAML"},
		)).Error().To(MatchError(ContainSubstring("unsupported encoding")))
	})
	It("Should reject the request when retrieve is not granted on the project", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "ungranted-project")
		Expect(apiSvc.Export(
			AuthedCtx(ctx, author),
			apiproject.ExportRequest{Key: proj.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when retrieve is not granted on a member", func(
		ctx SpecContext,
	) {
		proj := createProject(ctx, "ungranted-member")
		createLog(ctx, proj.Key, "Metrics")
		grantRetrieveOn(ctx, author.OntologyID(), project.OntologyID(proj.Key))
		Expect(apiSvc.Export(
			AuthedCtx(ctx, author),
			apiproject.ExportRequest{Key: proj.Key, Encoding: apiimex.EncodingJSON},
		)).Error().To(MatchError(access.ErrDenied))
	})
})
