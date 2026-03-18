// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	svcChannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx    context.Context
	dist   mock.Node
	svc    *svcChannel.Service
	arcSvc *arc.Service
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Channel Suite")
}

var _ = BeforeSuite(func() {
	ctx = context.Background()
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)
	labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	DeferCleanup(func() {
		Expect(labelSvc.Close()).To(Succeed())
	})
	statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Group:    dist.Group,
		Signals:  dist.Signals,
		Ontology: dist.Ontology,
		Label:    labelSvc,
	}))
	DeferCleanup(func() {
		Expect(statusSvc.Close()).To(Succeed())
	})
	rackSvc := MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           dist.DB,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		HostProvider: mock.StaticHostKeyProvider(1),
		Status:       statusSvc,
	}))
	DeferCleanup(func() {
		Expect(rackSvc.Close()).To(Succeed())
	})
	taskSvc := MustSucceed(task.OpenService(ctx, task.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Rack:     rackSvc,
		Status:   statusSvc,
	}))
	DeferCleanup(func() {
		Expect(taskSvc.Close()).To(Succeed())
	})
	arcSvc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Signals:  dist.Signals,
		Task:     taskSvc,
	}))
	DeferCleanup(func() {
		Expect(arcSvc.Close()).To(Succeed())
	})
	svc = MustSucceed(svcChannel.OpenService(ctx, svcChannel.ServiceConfig{
		DB:           dist.DB,
		Distribution: dist.Channel,
		Status:       statusSvc,
		Arc:          arcSvc,
	}))
	DeferCleanup(func() {
		Expect(svc.Close()).To(Succeed())
	})
})

var _ = AfterSuite(func() {
	Expect(dist.Close()).To(Succeed())
})
