// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

func TestLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Log Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	channelSvc *channel.Service
	groupSvc   *group.Service
	otg        *ontology.Ontology
	proj       project.Project
	projectSvc *project.Service
	svc        *log.Service
	imexSvc    *imex.Service
	// ch1 through ch99 are channels that the specs log.
	ch1, ch2, ch3, ch4, ch5, ch10, ch42, ch99 channel.Key
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Status:       statusSvc,
	}))
	chs := []*channel.Key{&ch1, &ch2, &ch3, &ch4, &ch5, &ch10, &ch42, &ch99}
	keys := MustSucceed(channeltestutil.CreateVirtual(ctx, channelSvc, len(chs)))
	for i, ch := range chs {
		*ch = keys[i]
	}
	imexSvc = imex.NewService()
	projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	svc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
		DB:       db,
		Ontology: otg,
		ImEx:     imexSvc,
	}))
	proj.Name = "test-project"
	Expect(projectSvc.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
})
