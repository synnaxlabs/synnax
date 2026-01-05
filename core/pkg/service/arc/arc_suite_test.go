// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

func TestArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Arc Suite")
}

var (
	ctx       = context.Background()
	db        *gorp.DB
	otg       *ontology.Ontology
	labelSvc  *label.Service
	svc       *arc.Service
	tx        gorp.Tx
	statusSvc *status.Service
	dist      mock.Node
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		EnableSearch: config.False(),
		DB:           db,
	}))

	// Use mock distribution for simplified testing
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)

	labelSvc = MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: dist.Ontology,
		Group:    dist.Group,
	}))

	statusSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    dist.Group,
		Label:    labelSvc,
	}))

	svc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Channel:  dist.Channel,
		Framer:   dist.Framer,
		Status:   statusSvc,
	}))
})

var (
	_ = AfterSuite(func() {
		Expect(svc.Close()).To(Succeed())
		Expect(labelSvc.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	_ = BeforeEach(func() { tx = db.OpenTx() })
	_ = AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
)
