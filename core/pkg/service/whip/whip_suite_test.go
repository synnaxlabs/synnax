// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package whip_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/synnax/pkg/service/whip"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db         *gorp.DB
	channelSvc *channel.Service
	framerSvc  *framer.Service
	statusSvc  *status.Service
	taskSvc    *task.Service
	rackKey    rack.Key
)

func TestWHIP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service WHIP Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	core := testutil.NewNode(ctx)
	db = core.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: db}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{DB: db}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: core.Channel,
		DB:      db,
		HostKey: core.Identity.HostKey,
		Status:  statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      db,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           db,
		Framer:       core.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{DB: db}))
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:         db,
		HostKey:    core.Identity.HostKey,
		Status:     statusSvc,
		DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
		// No Driver reports for the rack, so a check would overwrite task statuses.
		HealthCheckInterval: telem.Hour,
	}))
	configSvc := MustOpen(whip.OpenService(ctx, whip.ServiceConfig{DB: db}))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:      db,
		Rack:    rackSvc,
		Status:  statusSvc,
		Channel: channelSvc,
		ImEx:    imex.NewService(),
		Configs: MustSucceed(taskconfig.NewRegistry(configSvc.Stores()...)),
	}))
	r := rack.Rack{Name: "whip-rack"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rackSvc.NewWriter(tx).Create(ctx, &r)
	})).To(Succeed())
	rackKey = r.Key
})
