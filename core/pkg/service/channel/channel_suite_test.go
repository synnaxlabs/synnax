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
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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

// serviceConfig builds a channel ServiceConfig for the node, opening the ontology,
// group, label, and status services the channel service depends on.
func serviceConfig(ctx context.Context, node testutil.Node) channel.ServiceConfig {
	GinkgoHelper()
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Group:    groupSvc,
		Ontology: otg,
		Label:    labelSvc,
	}))
	return channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
	}
}

// openService opens a channel service for the node and creates the node's internal
// control channel, mirroring what the service layer's OpenLayer does in production.
// Tests rely on the control channel for their local-key and channel-count expectations.
// Extra configs override the derived distribution-layer fields.
func openService(
	ctx context.Context,
	node testutil.Node,
	cfgs ...channel.ServiceConfig,
) (*channel.Service, *ontology.Ontology) {
	GinkgoHelper()
	cfg := serviceConfig(ctx, node)
	channelSvc := MustOpen(channel.OpenService(
		ctx,
		append([]channel.ServiceConfig{cfg}, cfgs...)...,
	))
	return channelSvc, cfg.Ontology
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
	db = node.DB
	svc, _ = openService(ctx, node)
	channelWriter = svc.NewWriter(nil)
})
