// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/gorp"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/security"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/secret"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestAPIFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Framer Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

const rootUsername = "api-framer-suite-root"

var (
	apiSvc     *apiframer.Service
	framerSvc  *svcframer.Service
	channelSvc *channel.Service
	creator    calctestutil.Creator
	rbacSvc    *rbac.Service
	userSvc    *user.Service
	svc        *service.Layer
	db         *gorp.DB
	root       user.User
	// faultyFS holds the Core's time-series data and fails the operations a spec sets.
	faultyFS *FaultyFS
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	faultyFS = WrapFaultyFS(xfs.NewMem())
	kv := DeferClose(memkv.New())
	store := &storage.Layer{
		KV:   kv,
		Gorp: gorp.Wrap(kv),
		TS:   DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(faultyFS)))),
	}
	StartRun(store.Run)
	node := testutil.NewNode(ctx, distribution.LayerConfig{Storage: store})
	sec := MustSucceed(security.NewProvider(security.ProviderConfig{
		Insecure: new(true),
	}))
	db = node.DB
	svc = MustOpen(service.OpenLayer(ctx, service.LayerConfig{
		Distribution: node.Layer,
		Security:     sec,
		Storage:      store,
		RootCredentials: auth.Credentials{
			Username: rootUsername,
			Password: "p",
		},
		SecretsKey: secret.NewEncryptionKey(),
	}))
	framerSvc = svc.Framer
	channelSvc = svc.Channel
	creator = calctestutil.Creator{DB: db, Channel: channelSvc}
	rbacSvc = svc.RBAC
	userSvc = svc.User
	apiSvc = MustSucceed(apiframer.NewService(apiframer.ServiceConfig{
		DB:      db,
		Framer:  svc.Framer,
		Channel: svc.Channel,
		RBAC:    svc.RBAC,
		Records: svc.Signals.Records(),
		Errors:  errors.NewCodec(),
	}))
	Expect(svc.User.NewRetrieve().
		Where(user.MatchRootUser(true)).
		Entry(&root).
		Exec(ctx, db)).To(Succeed())
})

// rootCtx returns a context with the suite's root user installed as the request
// subject. The root user holds the Owner role, so every access check passes.
func rootCtx(ctx SpecContext) freighter.Context {
	return subjectCtx(ctx, root.ResourceID())
}

// subjectCtx returns a context with the given subject installed, so auth.GetSubject
// resolves it.
func subjectCtx(ctx SpecContext, subject resource.ID) freighter.Context {
	fCtx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fCtx.Set("Subject", subject)
	return fCtx
}

// createUserGranted creates a user that may retrieve the objects.
func createUserGranted(ctx SpecContext, objects ...resource.ID) user.User {
	GinkgoHelper()
	u := MustSucceed(rbactestutil.CreateUser(ctx, db, rbacSvc, userSvc, user.User{
		Name: "api-framer-" + uuid.New().String(),
	}))
	actions := []access.Action{access.ActionRetrieve}
	MustSucceed(
		rbactestutil.Grant(ctx, db, rbacSvc, userSvc, u.Key, actions, objects...),
	)
	return u
}
