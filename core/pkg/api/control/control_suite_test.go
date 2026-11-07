// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control_test

import (
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/freighter"
	apicontrol "github.com/synnaxlabs/synnax/pkg/api/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/auth/password"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/control"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIControl(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Control Suite", SuiteConfig())
}

var (
	channelSvc *channel.Service
	framerSvc  *framer.Service
	rbacSvc    *rbac.Service
	userSvc    *user.Service
	apiSvc     *apicontrol.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	dist := testutil.NewNode(ctx)
	passwordSvc := MustOpen(password.OpenService(ctx, password.ServiceConfig{
		DB: dist.DB,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: dist.DB, Password: passwordSvc,
	}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: dist.DB,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: dist.DB,
	}))
	rbacSvc = MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     dist.DB,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostProvider: dist.Cluster,
		Status:       statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      dist.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           dist.DB,
		Framer:       dist.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	sigs := MustSucceed(signals.New(signals.Config{
		DB:      dist.DB,
		Channel: channelSvc,
		Framer:  framerSvc,
	}))
	controlSvc := MustOpen(control.OpenService(ctx, control.ServiceConfig{
		Control: dist.Control,
		Signals: sigs,
	}))
	apiSvc = MustSucceed(apicontrol.NewService(apicontrol.ServiceConfig{
		Control: controlSvc,
		RBAC:    rbacSvc,
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()

// authedCtx returns a freighter context carrying u as the authenticated subject.
func authedCtx(ctx SpecContext, u user.User) freighter.Context {
	fctx := freighter.Context{Context: ctx, Params: freighter.Params{}}
	fctx.Set("Subject", u.ResourceID())
	return fctx
}

// newUser creates a user with no role assignments.
func newUser(ctx SpecContext) user.User {
	GinkgoHelper()
	return MustSucceed(userSvc.NewWriter(nil).
		Create(ctx, user.User{Name: "user-" + uuid.New().String()}))
}

// grantOn grants the actions on the objects to the subject through a new role.
func grantOn(
	ctx SpecContext,
	subject resource.ID,
	actions []access.Action,
	objects ...resource.ID,
) {
	GinkgoHelper()
	key := MustSucceed(user.KeyFromResourceID(subject))
	MustSucceed(rbactestutil.Grant(ctx, rbacSvc, userSvc, key, actions, objects...))
}
