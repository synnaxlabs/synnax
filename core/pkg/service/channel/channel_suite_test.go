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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	db            *gorp.DB
	svc           *channel.Service
	channelWriter channel.Writer
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Channel Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// serviceConfig builds a channel ServiceConfig for the node on db, opening the group,
// label, and status services the channel service depends on. It also returns the group
// service.
func serviceConfig(
	ctx context.Context,
	node testutil.Node,
	db *gorp.DB,
) (channel.ServiceConfig, *group.Service) {
	GinkgoHelper()
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB: db,
	}))
	MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB: db,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB: db,
	}))
	return channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           db,
		HostProvider: node.Cluster,
		Status:       statusSvc,
	}, groupSvc
}

// openService opens a channel service for the node and creates the node's internal
// control channel, mirroring what the service layer's OpenLayer does in production.
// Tests rely on the control channel for their local-key and channel-count expectations.
// Extra configs override the derived distribution-layer fields.
func openService(
	ctx context.Context,
	node testutil.Node,
	cfgs ...channel.ServiceConfig,
) *channel.Service {
	GinkgoHelper()
	cfg, _ := serviceConfig(ctx, node, node.DB)
	return MustOpen(channel.OpenService(
		ctx,
		append([]channel.ServiceConfig{cfg}, cfgs...)...,
	))
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc = openService(ctx, node)
	channelWriter = svc.NewWriter(nil)
})
