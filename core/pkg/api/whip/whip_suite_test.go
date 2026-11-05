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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac"
	"github.com/synnaxlabs/synnax/pkg/service/auth/apikey"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/whip"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	channelSvc *channel.Service
	userSvc    *user.Service
	apiKeySvc  *apikey.Service
	rbacSvc    *rbac.Service
	factory    *whip.Factory
)

func TestWHIP(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API WHIP Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db := node.DB
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
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      db,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           db,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	userSvc = MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	rbacSvc = MustOpen(rbac.OpenService(ctx, rbac.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		User:     userSvc,
	}))
	apiKeySvc = MustOpen(apikey.OpenService(ctx, apikey.ServiceConfig{DB: db}))
	factory = MustOpen(whip.OpenFactory(whip.FactoryConfig{
		DB:            db,
		Channel:       channelSvc,
		Framer:        framerSvc,
		Status:        statusSvc,
		KeyframeLimit: time.Second,
	}))
})
