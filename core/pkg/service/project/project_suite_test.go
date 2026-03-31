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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestProject(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Project Suite")
}

var (
	db  *gorp.DB
	otg *ontology.Ontology
	svc *project.Service
	tx  gorp.Tx
)

var _ = BeforeSuite(func(ctx SpecContext) {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		DB: db,
	}))
	searchIdx := MustSucceed(search.Open())
	DeferCleanup(func() {
		Expect(searchIdx.Close()).To(Succeed())
	})
	g := MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Search:   searchIdx,
	}))
	svc = MustSucceed(project.OpenService(ctx, project.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    g,
		Search:   searchIdx,
	}))
})

var (
	_ = AfterSuite(func(ctx SpecContext) {
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	_ = BeforeEach(func(ctx SpecContext) { tx = db.OpenTx() })
	_ = AfterEach(func(ctx SpecContext) { Expect(tx.Close()).To(Succeed()) })
)
