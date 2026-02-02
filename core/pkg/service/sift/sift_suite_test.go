// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	builder   *mock.Cluster
	dist      mock.Node
	framerSvc *framer.Service
	taskSvc   *task.Service
	statusSvc *status.Service
	deviceSvc *device.Service
	rackSvc   *rack.Service
	testRack  *rack.Rack
)

func TestSift(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sift Suite")
}

var _ = BeforeSuite(func() {
	builder = mock.NewCluster()
	ctx := context.Background()
	dist = builder.Provision(ctx)

	labelSvc := MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))

	statusSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Label:    labelSvc,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))

	rackSvc = MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           dist.DB,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		HostProvider: mock.StaticHostKeyProvider(1),
		Status:       statusSvc,
	}))

	testRack = &rack.Rack{Name: "Test Rack"}
	Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())

	taskSvc = MustSucceed(task.OpenService(ctx, task.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Rack:     rackSvc,
		Status:   statusSvc,
	}))

	arcSvc := MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Signals:  dist.Signals,
		Task:     taskSvc,
	}))

	framerSvc = MustSucceed(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:  dist.Framer,
		Channel: dist.Channel,
		Arc:     arcSvc,
		Status:  statusSvc,
		DB:      dist.DB,
	}))

	deviceSvc = MustSucceed(device.OpenService(ctx, device.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Status:   statusSvc,
		Rack:     rackSvc,
	}))
})

var _ = AfterSuite(func() {
	Expect(framerSvc.Close()).To(Succeed())
	Expect(builder.Close()).To(Succeed())
})
