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
	"github.com/synnaxlabs/freighter"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	apiproject "github.com/synnaxlabs/synnax/pkg/api/project"
	. "github.com/synnaxlabs/synnax/pkg/api/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/encoding/zip"
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

var (
	projectType = ontology.ID{Type: ontology.ResourceTypeProject}
	logType     = ontology.ID{Type: ontology.ResourceTypeLog}
)

// newBundle returns a version 1 project bundle holding one log.
func newBundle(name string) apiproject.ImportRequest {
	return zip.Files{
		"manifest.json": []byte(
			`{"version":1,"type":"project","name":"` + name + `"}`,
		),
		"Metrics.json": []byte(
			`{"version":2,"type":"log","name":"Metrics","channels":[]}`,
		),
	}
}

// importCtx returns an authed context carrying the file_name import param.
func importCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := AuthedCtx(ctx, u)
	fctx.Set("params", `{"file_name":"Imported.zip"}`)
	return fctx
}

var _ = Describe("Import", func() {
	It("Should import the bundle when create is granted on every kind", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.OntologyID(), access.ActionCreate, projectType, logType)
		res := MustSucceed(apiSvc.Import(importCtx(ctx, u), nil, newBundle("Imported")))
		Expect(res.Project.Name).To(Equal("Imported"))
	})
	It("Should reject the request when create is not granted on the project", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.OntologyID(), access.ActionCreate, logType)
		Expect(apiSvc.Import(importCtx(ctx, u), nil, newBundle("Imported"))).Error().
			To(MatchError(access.ErrDenied))
	})
	It("Should reject the request when create is granted on the project alone", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		grantOn(ctx, u.OntologyID(), access.ActionCreate, projectType)
		Expect(apiSvc.Import(importCtx(ctx, u), nil, newBundle("Imported"))).Error().
			To(MatchError(access.ErrDenied))
	})
	It("Should reject a request with no params", func(ctx SpecContext) {
		u := createUser(ctx)
		grantOn(ctx, u.OntologyID(), access.ActionCreate, projectType, logType)
		Expect(apiSvc.Import(AuthedCtx(ctx, u), nil, newBundle("Imported"))).Error().
			To(MatchError(ContainSubstring("params")))
	})
	It("Should reject an invalid bundle before any access check", func(
		ctx SpecContext,
	) {
		u := createUser(ctx)
		files := zip.Files{"Metrics.json": []byte("{}")}
		Expect(apiSvc.Import(importCtx(ctx, u), nil, files)).Error().
			To(MatchError(ContainSubstring("bundle holds no manifest.json")))
	})
})
