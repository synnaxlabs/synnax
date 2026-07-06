// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSchematic(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Schematic Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db   *gorp.DB
	otg  *ontology.Ontology
	proj project.Project
	svc  *schematic.Service
	tx   gorp.Tx
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		db = DeferClose(gorp.Wrap(memkv.New()))
		otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		var (
			searchIdx = MustOpen(search.Open())
			g         = MustOpen(group.OpenService(ctx, group.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Search:   searchIdx,
			}))
			projectSvc = MustOpen(project.OpenService(ctx, project.ServiceConfig{
				DB:       db,
				Ontology: otg,
				Group:    g,
				Search:   searchIdx,
			}))
		)
		svc = MustOpen(schematic.OpenService(ctx, schematic.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		proj.Name = "test-project"
		Expect(projectSvc.NewWriter(nil).Create(ctx, &proj)).To(Succeed())
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
