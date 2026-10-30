// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestTable(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Table Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db      *gorp.DB
	proj    project.Project
	svc     *table.Service
	imexSvc *imex.Service
	tx      gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		var (
			otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
			g   = MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
			panelSvc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
				DB:       db,
				Ontology: otg,
			}))
		)
		imexSvc = imex.NewService()
		projectSvc := MustOpen(project.OpenService(ctx, project.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			ImEx:     imexSvc,
			Panel:    panelSvc,
		}))
		svc = MustOpen(table.OpenService(ctx, table.ServiceConfig{
			DB:       db,
			Ontology: otg,
			ImEx:     imexSvc,
		}))
		proj.Name = "test-project"
		Expect(projectSvc.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
