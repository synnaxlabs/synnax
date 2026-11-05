// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	rbactestutil "github.com/synnaxlabs/synnax/pkg/service/access/rbac/testutil"
	svcchannel "github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	svcframer "github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	channelSvc    *svcchannel.Service
	channelWriter svcchannel.Writer
	apiChannelSvc *channel.Service
	framerSvc     *svcframer.Service
)

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Transport HTTP Framer Suite", SuiteConfig())
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: node.DB,
	}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:    node.DB,
		Group: groupSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: node.DB,
	}))
	channelSvc = MustOpen(svcchannel.OpenService(ctx, svcchannel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Status:       statusSvc,
	}))
	channelWriter = channelSvc.NewWriter(nil)
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(svcframer.OpenService(ctx, svcframer.ServiceConfig{
		DB:           node.DB,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:    node.DB,
		Group: groupSvc,
	}))
	rbacSvc := MustOpen(rbactestutil.Open(ctx, rbactestutil.Config{
		DB:     node.DB,
		User:   userSvc,
		Status: statusSvc,
	})).RBAC
	rangerSvc := MustOpen(ranger.OpenService(ctx, ranger.ServiceConfig{
		DB:    node.DB,
		Group: groupSvc,
	}))
	aliasSvc := MustOpen(alias.OpenService(ctx, alias.ServiceConfig{
		DB:              node.DB,
		Channel:         channelSvc,
		ParentRetriever: rangerSvc,
	}))
	apiChannelSvc = MustSucceed(channel.NewService(channel.ServiceConfig{
		RBAC:    rbacSvc,
		Channel: channelSvc,
		Ranger:  rangerSvc,
		Alias:   aliasSvc,
		Status:  statusSvc,
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
