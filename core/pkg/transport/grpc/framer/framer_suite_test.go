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
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
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
	svc := MustOpen(svctestutil.OpenLayer(ctx, testutil.NewNode(ctx)))
	channelSvc = svc.Channel
	apiChannelSvc = MustSucceed(channel.NewService(channel.ServiceConfig{
		RBAC:    svc.RBAC,
		Channel: svc.Channel,
		Status:  svc.Status,
		Alias:   svc.Alias,
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
