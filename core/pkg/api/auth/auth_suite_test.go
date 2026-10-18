// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	svc "github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/auth/token"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIAuth(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Auth Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	node        mock.Node
	tokenSvc    *token.Service
	passwordSvc *password.Service
	userSvc     *user.Service
	apiSvc      *apiauth.Service
	root        user.User
	creds       = auth.Credentials{Username: "api-auth-suite-root", Password: "p"}
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = mock.NewNode(ctx)
	db := node.DB
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db, Ontology: otg, Search: searchIdx,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: db, Ontology: otg, Group: groupSvc, Search: searchIdx,
	}))
	passwordSvc = MustOpen(password.OpenService(ctx, password.ServiceConfig{DB: db}))
	root = MustSucceed(user.ProvisionRoot(ctx, user.RootConfig{
		Users:       userSvc,
		Passwords:   passwordSvc,
		Credentials: creds,
	}))
	sec := MustSucceed(security.NewProvider(security.ProviderConfig{
		Insecure: new(true),
		KeySize:  secmock.SmallKeySize,
	}))
	tokenSvc = MustSucceed(token.NewService(token.ServiceConfig{KeyProvider: sec}))
	apiSvc = MustSucceed(apiauth.NewService(apicfg.LayerConfig{
		Distribution: node.Layer,
		Service: &svc.Layer{
			Token:    tokenSvc,
			Password: passwordSvc,
			User:     userSvc,
		},
	}))
})
