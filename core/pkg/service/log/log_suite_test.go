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
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func TestLog(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Log Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	channels   *channeltestutil.Table
	groupSvc   *group.Service
	otg        *ontology.Ontology
	proj       project.Project
	projectSvc *project.Service
	svc        *log.Service
	imexSvc    *imex.Service
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	channels = DeferClose(MustOpen(channeltestutil.OpenTable(ctx, db)))
	Expect(channels.Create(ctx, 1, 2, 3, 4, 5, 10, 42, 99)).To(Succeed())
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
