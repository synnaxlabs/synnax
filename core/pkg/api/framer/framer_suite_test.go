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

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiconfig "github.com/synnaxlabs/synnax/pkg/api/config"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/policy"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Framer Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

const rootUsername = "api-framer-suite-root"

var (
	apiSvc        *apiframer.Service
	framerSvc     *svcframer.Service
	channelWriter channel.Writer
	rbacSvc       *rbac.Service
	userSvc       *user.Service
	root          user.User
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := mock.NewNode(ctx)
	sec := MustSucceed(security.NewProvider(security.ProviderConfig{
		Insecure: new(true),
		KeySize:  secmock.SmallKeySize,
	}))
	svc := MustOpen(service.OpenLayer(ctx, service.LayerConfig{
		Distribution: node.Layer,
		Security:     sec,
		Storage:      node.Storage,
		RootCredentials: auth.Credentials{
			Username: rootUsername,
			Password: "p",
		},
	}))
	framerSvc = svc.Framer
	channelWriter = svc.Channel.NewWriter(nil)
	rbacSvc = svc.RBAC
	userSvc = svc.User
	apiSvc = MustSucceed(apiframer.NewService(apiconfig.LayerConfig{
		Distribution: node.Layer,
		Service:      svc,
	}))
	Expect(svc.User.NewRetrieve().
		Where(user.MatchUsernames(rootUsername)).
		Entry(&root).
		Exec(ctx, nil)).To(Succeed())
})

// rootCtx returns a context with the suite's root user installed as the request
// subject. The root user holds the Owner role, so every access check passes.
func rootCtx(ctx SpecContext) freighter.Context {
	return subjectCtx(ctx, root.OntologyID())
}

// subjectCtx returns a context with the given subject installed, so auth.GetSubject
// resolves it.
func subjectCtx(ctx SpecContext, subject ontology.ID) freighter.Context {
	fCtx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fCtx.Set("Subject", subject)
	return fCtx
}

// createUserGranted creates a user holding a fresh role that permits the given action
// on the given objects, and nothing else. Writes commit directly, since the enforcers
// read committed state with no transaction.
func createUserGranted(
	ctx SpecContext,
	action access.Action,
	objects ...ontology.ID,
) user.User {
	u := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{
		Username: "api-framer-" + uuid.NewString(),
	}))
	roleWriter := rbacSvc.Role.NewWriter(nil, true)
	r := &role.Role{Name: "api-framer-" + uuid.NewString()}
	Expect(roleWriter.Create(ctx, r)).To(Succeed())
	policyWriter := rbacSvc.Policy.NewWriter(nil, true)
	p := &policy.Policy{
		Name:    "api-framer-" + uuid.NewString(),
		Objects: objects,
		Actions: []access.Action{action},
	}
	Expect(policyWriter.Create(ctx, p)).To(Succeed())
	Expect(policyWriter.SetOnRole(ctx, r.Key, p.Key)).To(Succeed())
	Expect(roleWriter.AssignRole(ctx, u.OntologyID(), r.Key)).To(Succeed())
	return u
}
