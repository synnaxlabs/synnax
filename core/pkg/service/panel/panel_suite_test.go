// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"context"
	"testing"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
)

func TestPanel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Panel Suite", SuiteConfig())
}

var (
	node       testutil.Node
	db         *gorp.DB
	groupSvc   *group.Service
	svc        *panel.Service
	framerSvc  *framer.Service
	channelSvc *channel.Service
	projectSvc *project.Service
	plotSvc    *lineplot.Service
	proj       project.Project
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = testutil.NewNode(ctx)
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: node.DB,
	}))
	db = node.DB
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: node.DB,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      node.DB,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           node.DB,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	sigs := MustSucceed(signals.New(signals.Config{
		DB:      db,
		Channel: channelSvc,
		Framer:  node.Framer,
	}))
	svc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB:      node.DB,
		Signals: sigs,
	}))
	MustOpen(sigs.Start(ctx))
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB: node.DB,
	}))
	proj = project.Project{Name: "panel-project"}
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
	plotSvc = MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
		DB:   node.DB,
		ImEx: imex.NewService(),
	}))
})

// createPlot stores the line plot that tab(key) displays.
func createPlot(ctx context.Context, tx gorp.Tx, key uuid.UUID) {
	GinkgoHelper()
	p := defaults.Fill(lineplot.LinePlot{Key: key, Name: "plot", Project: &proj.Key})
	Expect(plotSvc.NewWriter(tx).Create(ctx, &p)).To(Succeed())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// dispatch runs Writer.Apply in a transaction of its own.
func dispatch(
	ctx context.Context,
	key panel.Key,
	dispatchKey string,
	acts []panel.Action,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := svc.NewWriter(tx).
			WithDispatchKey(dispatchKey).
			Apply(ctx, key, acts...)
		return err
	})
}
