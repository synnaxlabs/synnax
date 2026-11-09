// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/server"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// healthTransport serves the gRPC health service, whose Watch stream stays open.
type healthTransport struct{ *health.Server }

func (healthTransport) Use(...freighter.Middleware) {}

func (healthTransport) Report() alamos.Report { return nil }

func (t healthTransport) BindTo(reg grpc.ServiceRegistrar) {
	healthpb.RegisterHealthServer(reg, t.Server)
}

var _ = Describe("gRPC", func() {
	It("Should start a gRPC server", func() {
		b := MustSucceed(server.Serve(server.Config{
			Listeners: []server.Listener{{Address: "localhost:0"}},
			Security:  server.SecurityConfig{Insecure: new(true)},
			Branches:  []server.Branch{&server.GRPCBranch{}},
		}))
		Expect(b.Close()).To(Succeed())
	})

	It("Should hold a stream past MaxConcurrentStreams until one closes", func(
		ctx SpecContext,
	) {
		b := MustSucceed(server.Serve(server.Config{
			Listeners: []server.Listener{{Address: "localhost:0"}},
			Security:  server.SecurityConfig{Insecure: new(true)},
			Branches: []server.Branch{&server.GRPCBranch{
				Transports: []fgrpc.BindableTransport{
					healthTransport{Server: health.NewServer()},
				},
				MaxConcurrentStreams: 1,
			}},
		}))
		DeferCleanup(b.Close)
		conn := MustSucceed(grpc.NewClient(
			b.Addresses()[0].String(),
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		))
		DeferCleanup(conn.Close)
		client := healthpb.NewHealthClient(conn)
		watch := func(ctx context.Context) error {
			stream, err := client.Watch(ctx, &healthpb.HealthCheckRequest{})
			if err != nil {
				return err
			}
			_, err = stream.Recv()
			return err
		}
		firstCtx, cancelFirst := context.WithCancel(ctx)
		DeferCleanup(cancelFirst)
		Expect(watch(firstCtx)).To(Succeed())
		secondCtx, cancelSecond := context.WithTimeout(ctx, 200*time.Millisecond)
		DeferCleanup(cancelSecond)
		Expect(watch(secondCtx)).
			To(MatchError(ContainSubstring("code = DeadlineExceeded")))
		cancelFirst()
		Expect(watch(ctx)).To(Succeed())
	})
})
