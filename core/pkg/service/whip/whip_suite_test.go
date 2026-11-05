// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package whip_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db         *gorp.DB
	channelSvc *channel.Service
	framerSvc  *framer.Service
	statusSvc  *status.Service
)

func TestWHIP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service WHIP Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	core := testutil.NewCluster(ctx, 0).Provision(ctx)
	db = core.DB
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statusSvc = MustOpen(status.OpenService(ctx, status.ServiceConfig{
		Ontology: otg,
		DB:       db,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      core.Channel,
		DB:           db,
		HostProvider: core.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      db,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           db,
		Framer:       core.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
})
