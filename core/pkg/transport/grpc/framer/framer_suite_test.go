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
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	grpcframer "github.com/synnaxlabs/synnax/pkg/transport/grpc/framer"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

var (
	channelSvc    *svcchannel.Service
	apiChannelSvc *channel.Service
	transport     api.Transport
	bindable      fgrpc.BindableTransport
	conn          *fgrpc.ClientConn
)

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport gRPC Framer Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: node.DB,
	}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: node.DB,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc = MustOpen(svcchannel.OpenService(ctx, svcchannel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Status:       statusSvc,
	}))
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB: node.DB,
	}))
	rbacSvc := MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     node.DB,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	rangerSvc := MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
		DB:    node.DB,
		Group: groupSvc,
	}))
	aliasSvc := MustOpen(alias.OpenService(ctx, alias.ServiceConfig{
		DB:              node.DB,
		Channel:         channelSvc,
		ParentRetriever: rangerSvc,
	}))
	apiChannelSvc = MustSucceed(channel.NewService(channel.ServiceConfig{
		RBAC:    rbacSvc,
		Channel: channelSvc,
		Ranger:  rangerSvc,
		Alias:   aliasSvc,
		Status:  statusSvc,
	}))
	var pool *fgrpc.Pool
	addr := StartServer(func(reg grpc.ServiceRegistrar, p *fgrpc.Pool) {
		bindable = grpcframer.New(&transport, apiChannelSvc)
		bindable.BindTo(reg)
		pool = p
	})
	conn = MustSucceed(pool.Acquire(addr))
	DeferCleanup(conn.Release)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
