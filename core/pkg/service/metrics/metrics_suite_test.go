// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	builder   *mock.Cluster
	dist      mock.Node
	svcFramer *framer.Service
	labelSvc  *label.Service
	statusSvc *status.Service
	arcSvc    *arc.Service
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metrics Suite")
}

var _ = BeforeSuite(func() {
	builder = mock.NewCluster()
	ctx := context.Background()
	dist = builder.Provision(ctx)
	labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Label:    labelSvc,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
	}))
	arcSvc := MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Framer:   dist.Framer,
		Status:   statusSvc,
		Signals:  dist.Signals,
	}))
	svcFramer = MustSucceed(framer.OpenService(ctx, framer.Config{
		Framer:  dist.Framer,
		Channel: dist.Channel,
		Arc:     arcSvc,
		DB:      dist.DB,
	}))
})

var _ = AfterSuite(func() {
	Expect(svcFramer.Close()).To(Succeed())
	Expect(arcSvc.Close()).To(Succeed())
	Expect(statusSvc.Close()).To(Succeed())
	Expect(labelSvc.Close()).To(Succeed())
	Expect(builder.Close()).To(Succeed())
})
