// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
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

func TestWorkspace(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Workspace Suite")
}

var (
	ctx          = context.Background()
	db           *gorp.DB
	otg          *ontology.Ontology
	groupSvc     *group.Service
	svc          *workspace.Service
	schematicSvc *schematic.Service
	lineplotSvc  *lineplot.Service
	logSvc       *log.Service
	tableSvc     *table.Service
	userSvc      *user.Service
	author       user.User
	tx           gorp.Tx
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		EnableSearch: new(false),
		DB:           db,
	}))
	groupSvc = MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	schematicSvc = MustSucceed(schematic.OpenService(ctx, schematic.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	lineplotSvc = MustSucceed(lineplot.NewService(lineplot.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	logSvc = MustSucceed(log.NewService(log.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	tableSvc = MustSucceed(table.NewService(table.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	svc = MustSucceed(workspace.OpenService(ctx, workspace.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		ChildDeleters: []workspace.ChildDeleter{
			schematicSvc, lineplotSvc, logSvc, tableSvc,
		},
	}))
	userSvc = MustSucceed(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	author.Username = "test"
	Expect(userSvc.NewWriter(nil).Create(ctx, &author)).To(Succeed())
})

var (
	_ = AfterSuite(func() {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	_ = BeforeEach(func() { tx = db.OpenTx() })
	_ = AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
)
