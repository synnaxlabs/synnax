// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package metrics_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	servicechannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	builder    *mock.Cluster
	dist       mock.Node
	framerSvc  *framer.Service
	channelSvc *servicechannel.Service
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

var _ = BeforeSuite(func() {
	bgCtx := context.Background()
	builder = mock.NewCluster()
	dist = builder.Provision(bgCtx)
	labelSvc := MustSucceed(label.OpenService(bgCtx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	statusSvc := MustSucceed(status.OpenService(bgCtx, status.ServiceConfig{
		DB:       dist.DB,
		Label:    labelSvc,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	rackSvc := MustSucceed(rack.OpenService(bgCtx, rack.ServiceConfig{
		DB:           dist.DB,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		HostProvider: mock.StaticHostKeyProvider(1),
		Status:       statusSvc,
	}))
	taskSvc := MustSucceed(task.OpenService(bgCtx, task.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Rack:     rackSvc,
		Status:   statusSvc,
	}))
	arcSvc := MustSucceed(arc.OpenService(bgCtx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Signals:  dist.Signals,
		Task:     taskSvc,
	}))
	channelSvc = MustSucceed(servicechannel.OpenService(bgCtx, servicechannel.ServiceConfig{
		DB:           dist.DB,
		Distribution: dist.Channel,
		Status:       statusSvc,
		Arc:          arcSvc,
	}))
	framerSvc = MustSucceed(framer.OpenService(bgCtx, framer.ServiceConfig{
		Framer:  dist.Framer,
		Channel: channelSvc,
		Arc:     arcSvc,
		Status:  statusSvc,
		DB:      dist.DB,
	}))
})

var _ = AfterSuite(func(ctx SpecContext) {
	Expect(channelSvc.Close()).To(Succeed())
	Expect(framerSvc.Close()).To(Succeed())
	Expect(builder.Close()).To(Succeed())
})
