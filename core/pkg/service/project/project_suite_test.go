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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	. "github.com/synnaxlabs/x/testutil"
)

func TestProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Project Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db           *gorp.DB
	otg          *ontology.Ontology
	svc          *project.Service
	groupSvc     *group.Service
	schematicSvc *schematic.Service
	lineplotSvc  *lineplot.Service
	logSvc       *log.Service
	tableSvc     *table.Service
	panelSvc     *panel.Service
	writer       project.Writer
	tx           gorp.Tx
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	svc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	writer = svc.NewWriter(nil)
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	schematicSvc = MustOpen(schematic.OpenService(ctx, schematic.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		ImEx:     imex.NewService(),
	}))
	lineplotSvc = MustOpen(lineplot.OpenService(ctx, lineplot.ServiceConfig{
		DB:       db,
		Ontology: otg,
		ImEx:     imex.NewService(),
	}))
	logSvc = MustOpen(log.OpenService(ctx, log.ServiceConfig{
		DB:       db,
		Ontology: otg,
		ImEx:     imex.NewService(),
	}))
	tableSvc = MustOpen(table.OpenService(ctx, table.ServiceConfig{
		DB:       db,
		Ontology: otg,
		ImEx:     imex.NewService(),
	}))
	panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
})
