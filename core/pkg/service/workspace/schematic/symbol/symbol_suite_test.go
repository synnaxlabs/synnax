// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	"context"
	"testing"

	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/synnax/pkg/service/workspace/schematic/symbol"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

func TestSymbol(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Symbol Suite")
}

var (
	ctx     = context.Background()
	db      *gorp.DB
	otg     *ontology.Ontology
	ws      workspace.Workspace
	userSvc *user.Service
	svc     *symbol.Service
	tx      gorp.Tx
)

var _ = BeforeSuite(func() {
	var err error
	db = gorp.Wrap(memkv.New())
	Expect(err).ToNot(HaveOccurred())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		EnableSearch: config.False(),
		DB:           db,
	}))
	g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	workspaceSvc := MustSucceed(workspace.OpenService(ctx, workspace.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	userSvc = MustSucceed(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
	var author user.User
	author.Username = "test"
	Expect(userSvc.NewWriter(nil).Create(ctx, &author)).To(Succeed())
	ws.Author = author.Key
	Expect(workspaceSvc.NewWriter(nil).Create(ctx, &ws)).To(Succeed())
	svc = MustSucceed(symbol.OpenService(ctx, symbol.Config{
		DB:       db,
		Ontology: otg,
		Group:    g,
	}))
})

var (
	_ = AfterSuite(func() {
		Expect(svc.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	_ = BeforeEach(func() { tx = db.OpenTx() })
	_ = AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
)
