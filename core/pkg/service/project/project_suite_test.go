// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/pagerduty"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/task/config"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

func TestProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Project Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db       *gorp.DB
	otg      *ontology.Ontology
	svc      *project.Service
	groupSvc *group.Service
	panelSvc *panel.Service
	logSvc   *log.Service
	// lineplotSvc registers its exporter in a registry the project service never sees,
	// so line plots stand in for children Export cannot serialize.
	lineplotSvc *lineplot.Service
	imexSvc     *imex.Service
	taskSvc     *task.Service
	testRack    rack.Rack
	writer      project.Writer
	tx          gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := mock.NewNode(ctx)
		db = node.DB
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		imexSvc = imex.NewService()
		panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		logSvc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
			ImEx:     imexSvc,
		}))
		lineplotSvc = MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
			ImEx:     imex.NewService(),
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           db,
			Ontology:     otg,
			Group:        groupSvc,
			Search:       searchIdx,
			Status:       statusSvc,
			HostProvider: node.Cluster,
		}))
		pd := MustOpen(pagerduty.OpenService(ctx, pagerduty.ServiceConfig{DB: db}))
		taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Rack:     rackSvc,
			Status:   statusSvc,
			Search:   searchIdx,
			ImEx:     imexSvc,
			Configs:  MustSucceed(config.NewRegistry(pd.Stores()...)),
		}))
		testRack = rack.Rack{Name: "Project Test Rack"}
		Expect(rackSvc.NewWriter(nil).Create(ctx, &testRack)).To(Succeed())
		svc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    groupSvc,
			Search:   searchIdx,
			ImEx:     imexSvc,
			Panel:    panelSvc,
		}))
		writer = svc.NewWriter(nil)
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
