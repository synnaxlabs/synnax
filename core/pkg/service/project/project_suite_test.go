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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
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
	writer      project.Writer
	tx          gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
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
