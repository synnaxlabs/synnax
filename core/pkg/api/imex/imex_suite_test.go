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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	apiimex "github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIImEx(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API ImEx Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db       *gorp.DB
	imexSvc  *imex.Service
	apiSvc   *apiimex.Service
	importer *recordingImporter
	root     user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg, Search: searchIdx,
	}))
	authSvc := MustOpen(auth.OpenService(ctx, auth.ServiceConfig{DB: db}))
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:              db,
		Ontology:        otg,
		Group:           groupSvc,
		Search:          searchIdx,
		Auth:            authSvc,
		RootCredentials: auth.Credentials{Username: "api-imex-suite-root", Password: "p"},
	}))
	rbacSvc := MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
		User:     userSvc,
	}))
	imexSvc = imex.NewService()
	importer = &recordingImporter{}
	imexSvc.RegisterImporter(string(importer.Type()), importer)
	apiSvc = MustSucceed(apiimex.NewService(apicfg.LayerConfig{
		Distribution: &distribution.Layer{DB: db},
		Service: &service.Layer{
			ImEx: imexSvc,
			RBAC: rbacSvc,
		},
	}))
	Expect(userSvc.NewRetrieve().
		Where(user.MatchUsernames("api-imex-suite-root")).
		Entry(&root).Exec(ctx, nil)).To(Succeed())
})

// rootCtx returns a freighter.Context with the suite-level root user installed as the
// request subject. The root user holds the Owner role, so every RBAC enforcement call
// passes.
func rootCtx(ctx SpecContext) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", root.OntologyID())
	return fctx
}
