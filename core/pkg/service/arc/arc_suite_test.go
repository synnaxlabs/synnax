// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskconfig "github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Arc Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	svc        *arc.Service
	writer     arc.Writer
	channelSvc *channel.Service
	taskSvc    *task.Service
	arcTaskSvc *arctask.Service
	statusSvc  *status.Service
	rackSvc    *rack.Service
	testRack   *rack.Rack
	imexSvc    *imex.Service
	arcClock   = telem.Now
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:    db,
		Group: groupSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Status:       statusSvc,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		HostProvider:        testutil.NewStaticHostProvider(1),
		Status:              statusSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
		DriverRole:          MustSucceed(racktestutil.CreateDriverRole(ctx, roleSvc)),
	}))
	imexSvc = imex.NewService()
	arcTable := MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
	arcTaskSvc = MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{
		DB: db,
	}))
	configs := MustSucceed(taskconfig.NewRegistry(arcTaskSvc.Stores()...))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:           db,
		Rack:         rackSvc,
		Status:       statusSvc,
		ImEx:         imexSvc,
		Configs:      configs,
		ImExExcluded: []string{arctask.Type},
	}))
	testRack = &rack.Rack{Name: "Test Rack"}
	Expect(rackSvc.NewWriter(db).Create(ctx, testRack)).To(Succeed())
	svc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
		DB:                  db,
		Table:               arcTable,
		Channel:             channelSvc,
		Task:                taskSvc,
		MatchTasks:          arcTaskSvc.MatchArcs,
		Status:              statusSvc,
		ImEx:                imexSvc,
		TextSweepQuiescence: 5 * telem.Second,
		TextSweepThreshold:  1,
		TaskSyncDebounce:    5 * telem.Millisecond,
		Now:                 func() telem.TimeStamp { return arcClock() },
	}))
	writer = svc.NewWriter(nil)
})
