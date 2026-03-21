// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package export_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/export"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestExport(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Export Suite")
}

var (
	ctx        = context.Background()
	db         *gorp.DB
	otg        *ontology.Ontology
	svc        *export.Service
	svcLayer   *service.Layer
	testAuthor user.User
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		EnableSearch: new(false),
		DB:           db,
	}))
	g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))

	userSvc := MustSucceed(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	testAuthor = user.User{Username: "test_export_user"}
	Expect(userSvc.NewWriter(nil).Create(ctx, &testAuthor)).To(Succeed())

	svcLayer = &service.Layer{
		User: userSvc,
		Workspace: MustSucceed(workspace.OpenService(ctx, workspace.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		})),
		LinePlot: MustSucceed(lineplot.NewService(lineplot.ServiceConfig{
			DB:       db,
			Ontology: otg,
		})),
		Schematic: MustSucceed(schematic.OpenService(ctx, schematic.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
		})),
		Table: MustSucceed(table.NewService(table.ServiceConfig{
			DB:       db,
			Ontology: otg,
		})),
		Log: MustSucceed(log.NewService(log.ServiceConfig{
			DB:       db,
			Ontology: otg,
		})),
	}

	svc = export.NewService(export.ServiceConfig{
		Service:      svcLayer,
		Distribution: &distribution.Layer{Ontology: otg},
	})
})

var _ = AfterSuite(func() {
	Expect(otg.Close()).To(Succeed())
	Expect(db.Close()).To(Succeed())
})
