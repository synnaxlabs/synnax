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
	dist   mock.Node
	svc    *svcChannel.Service
	arcSvc *arc.Service
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Channel Suite")
}

var _ = BeforeSuite(func(ctx SpecContext) {
	dist = DeferClose(mock.NewCluster().Provision(ctx))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Signals:  dist.Signals,
		Search:   dist.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Group:    dist.Group,
		Signals:  dist.Signals,
		Ontology: dist.Ontology,
		Label:    labelSvc,
		Search:   dist.Search,
	}))
	rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
		DB:           dist.DB,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		HostProvider: mock.StaticHostKeyProvider(1),
		Status:       statusSvc,
		Search:       dist.Search,
	}))
	taskSvc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Rack:     rackSvc,
		Status:   statusSvc,
		Search:   dist.Search,
	}))
	arcSvc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
		Channel:  dist.Channel,
		Ontology: dist.Ontology,
		DB:       dist.DB,
		Signals:  dist.Signals,
		Task:     taskSvc,
		Search:   dist.Search,
	}))
	svc = MustOpen(svcChannel.OpenService(ctx, svcChannel.ServiceConfig{
		DB:           dist.DB,
		Distribution: dist.Channel,
		Status:       statusSvc,
		Arc:          arcSvc,
	}))
})
