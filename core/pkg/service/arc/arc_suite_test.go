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
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Arc Suite")
}

var (
	ctx      = context.Background()
	db       *gorp.DB
	otg      *ontology.Ontology
	svc      *arc.Service
	tx       gorp.Tx
	dist     mock.Node
	groupSvc *group.Service
	labelSvc *label.Service
	statSvc  *status.Service
	rackSvc  *rack.Service
	taskSvc  *task.Service
	testRack *rack.Rack
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		EnableSearch: new(false),
		DB:           db,
	}))

	// Use mock distribution for simplified testing
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)

	groupSvc = MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	labelSvc = MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	rackSvc = MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		Ontology:            otg,
		Group:               groupSvc,
		HostProvider:        mock.StaticHostKeyProvider(1),
		Status:              statSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
	}))
	taskSvc = MustSucceed(task.OpenService(ctx, task.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Rack:     rackSvc,
		Status:   statSvc,
	}))
	testRack = &rack.Rack{Name: "Test Rack"}
	Expect(rackSvc.NewWriter(db).Create(ctx, testRack)).To(Succeed())

	svc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Channel:  dist.Channel,
		Task:     taskSvc,
	}))
})

var (
	_ = AfterSuite(func() {
		Expect(svc.Close()).To(Succeed())
		Expect(taskSvc.Close()).To(Succeed())
		Expect(rackSvc.Close()).To(Succeed())
		Expect(statSvc.Close()).To(Succeed())
		Expect(labelSvc.Close()).To(Succeed())
		Expect(groupSvc.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	_ = BeforeEach(func() { tx = db.OpenTx() })
	_ = AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
)
