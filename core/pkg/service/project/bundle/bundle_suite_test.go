// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package bundle_test

import (
	"slices"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	arctask "github.com/synnaxlabs/synnax/pkg/service/arc/task"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/project/bundle"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	racktestutil "github.com/synnaxlabs/synnax/pkg/service/rack/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	secrettestutil "github.com/synnaxlabs/synnax/pkg/service/secret/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	. "github.com/synnaxlabs/x/testutil"
)

func TestBundle(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Project Bundle Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db          *gorp.DB
	resourceSvc *resource.Service
	svc         *bundle.Service
	groupSvc    *group.Service
	panelSvc    *panel.Service
	logSvc      *log.Service
	// lineplotSvc registers its exporter in a registry the bundle service never sees,
	// so line plots stand in for children Export cannot serialize.
	lineplotSvc *lineplot.Service
	imexSvc     *imex.Service
	taskSvc     *task.Service
	arcSvc      *arc.Service
	testRack    rack.Rack
	projectSvc  *project.Service
	tx          gorp.Tx
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	imexSvc = imex.NewService()
	panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB: db,
	}))
	logSvc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
		DB:   db,
		ImEx: imexSvc,
	}))
	lineplotSvc = MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
		DB:   db,
		ImEx: imex.NewService(),
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	roleSvc := MustOpen(role.OpenService(ctx, role.ServiceConfig{
		DB: db,
	}))
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:         db,
		Status:     statusSvc,
		HostKey:    node.Identity.HostKey,
		DriverRole: MustSucceed(racktestutil.CreateDriverRole(ctx, db, roleSvc)),
	}))
	pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{
		DB:     db,
		Secret: secrettestutil.OpenService(ctx, db),
	}))
	arcTable := MustOpen(arc.OpenTable(ctx, arc.TableConfig{DB: db}))
	arcTaskSvc := MustOpen(arctask.OpenService(ctx, arctask.ServiceConfig{DB: db}))
	taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:     db,
		Rack:   rackSvc,
		Status: statusSvc,
		ImEx:   imexSvc,
		Configs: MustSucceed(config.NewRegistry(
			slices.Concat(pd.Stores(), arcTaskSvc.Stores())...,
		)),
		ImExExcluded: []string{arctask.Type},
	}))
	testRack = rack.Rack{Name: "Project Test Rack"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return rackSvc.NewWriter(tx).Create(ctx, &testRack)
	})).To(Succeed())
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	arcSvc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
		DB:         db,
		Table:      arcTable,
		Channel:    channelSvc,
		Task:       taskSvc,
		MatchTasks: arcTaskSvc.MatchArcs,
		Status:     statusSvc,
		ImEx:       imexSvc,
	}))
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB: db,
	}))
	resourceSvc = MustSucceed(resource.NewService(resource.ServiceConfig{
		Providers: map[resource.Type]resource.Provider{
			resource.TypeArc:      arcSvc.Resources(),
			resource.TypeChannel:  channelSvc.Resources(),
			resource.TypeGroup:    groupSvc.Resources(),
			resource.TypeLabel:    labelSvc.Resources(),
			resource.TypeLineplot: lineplotSvc.Resources(),
			resource.TypeLog:      logSvc.Resources(),
			resource.TypePanel:    panelSvc.Resources(),
			resource.TypeProject:  projectSvc.Resources(),
			resource.TypeRack:     rackSvc.Resources(),
			resource.TypeStatus:   statusSvc.Resources(),
			resource.TypeTask:     taskSvc.Resources(),
		},
	}))
	svc = MustSucceed(bundle.NewService(bundle.Config{
		Resource: resourceSvc,
		Project:  projectSvc,
		Group:    groupSvc,
		Panel:    panelSvc,
		ImEx:     imexSvc,
	}))
})

// textOf returns the text of the text-mode Arc b, failing the spec otherwise.
func textOf(b arc.Base) text.Text {
	GinkgoHelper()
	src, ok := b.Source.Variant.(arc.TextSource)
	Expect(ok).To(BeTrue(), "expected a text source, got %T", b.Source.Variant)
	return src.Text
}
