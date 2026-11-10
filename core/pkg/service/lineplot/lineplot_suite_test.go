// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lineplot_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

func TestLinePlot(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Line Plot Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	channelSvc *channel.Service
	groupSvc   *group.Service
	proj       project.Project
	projectSvc *project.Service
	svc        *lineplot.Service
	imexSvc    *imex.Service
	// ch1 through ch99 are channels that the specs plot.
	ch1, ch2, ch3, ch4, ch5, ch6, ch7, ch9, ch10, ch20, ch42, ch43, ch99 channel.Key
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel: node.Channel,
		DB:      db,
		HostKey: node.Identity.HostKey,
		Status:  statusSvc,
	}))
	chs := []*channel.Key{
		&ch1,
		&ch2,
		&ch3,
		&ch4,
		&ch5,
		&ch6,
		&ch7,
		&ch9,
		&ch10,
		&ch20,
		&ch42,
		&ch43,
		&ch99,
	}
	keys := MustSucceed(channeltestutil.CreateVirtual(ctx, db, channelSvc, len(chs)))
	for i, ch := range chs {
		*ch = keys[i]
	}
	imexSvc = imex.NewService()
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB: db,
	}))
	svc = MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
		DB:   db,
		ImEx: imexSvc,
	}))
	proj.Name = "test-project"
	Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
		return projectSvc.NewWriter(tx).Create(ctx, &proj)
	})).To(Succeed())
})

// dispatch runs Writer.Apply in a transaction of its own.
func dispatch(
	ctx context.Context,
	key lineplot.Key,
	dispatchKey string,
	acts []lineplot.Action,
) error {
	return db.WithTx(ctx, func(tx gorp.Tx) error {
		_, err := svc.NewWriter(tx).
			WithDispatchKey(dispatchKey).
			Apply(ctx, key, acts...)
		return err
	})
}
