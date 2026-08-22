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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apiconfig "github.com/synnaxlabs/synnax/pkg/api/config"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/security"
	secmock "github.com/synnaxlabs/synnax/pkg/security/mock"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/auth"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
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
	fCtx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fCtx.Set("Subject", root.OntologyID())
	return fCtx
}
