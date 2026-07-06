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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	dist       mock.Node
	otg        *ontology.Ontology
	searchIdx  *search.Index
	groupSvc   *group.Service
	framerSvc  *framer.Service
	channelSvc *channel.Service
)

func TestMetrics(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Metrics Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	dist = mock.NewNode(ctx)
	otg = MustOpen(ontology.Open(ctx, ontology.Config{DB: dist.DB}))
	searchIdx = MustOpen(search.Open())
	groupSvc = MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       dist.DB,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Label:    labelSvc,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostResolver: dist.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:       dist.Framer,
		Channel:      channelSvc,
		Status:       statusSvc,
		HostResolver: dist.Cluster,
	}))
})
