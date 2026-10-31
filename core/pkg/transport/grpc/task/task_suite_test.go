// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	. "github.com/synnaxlabs/freighter/grpc/testutil"
	"github.com/synnaxlabs/synnax/pkg/api"
	grpctask "github.com/synnaxlabs/synnax/pkg/transport/grpc/task"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/grpc"
)

var (
	transport api.Transport
	conn      *fgrpc.ClientConn
)

func TestTask(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport gRPC Task Suite", SuiteConfig())
}

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	var pool *fgrpc.Pool
	addr := StartServer(func(reg grpc.ServiceRegistrar, p *fgrpc.Pool) {
		grpctask.New(&transport).BindTo(reg)
		pool = p
	})
	conn = MustSucceed(pool.Acquire(addr))
	DeferCleanup(conn.Release)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
