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
	"context"
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

func TestFramer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Framer Suite")
}

var (
	node          mock.Node
	channelSvc    *channel.Service
	channelWriter channel.Writer
	statusSvc     *status.Service
	framerSvc     *framer.Service
	validCfg      framer.ServiceConfig
)

// newFramerConfig opens the label, status, and channel services backing a framer
// Service on n and returns a valid framer.ServiceConfig. It is used both to build the
// shared suite-level service and to spin up isolated services within individual specs
// (e.g. to exercise Close without conflicting with the shared service's control update
// channel).
func newFramerConfig(ctx context.Context, n mock.Node) framer.ServiceConfig {
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: n.DB}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       n.DB,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       n.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       n.DB,
		Label:    labelSvc,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      n.Channel,
		DB:           n.DB,
		HostProvider: n.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	}))
	return framer.ServiceConfig{
		Framer:       n.Framer,
		Channel:      channelSvc,
		Status:       statusSvc,
		HostProvider: n.Cluster,
	}
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = mock.NewNode(ctx)
	validCfg = newFramerConfig(ctx, node)
	channelSvc = validCfg.Channel
	channelWriter = channelSvc.NewWriter(nil)
	statusSvc = validCfg.Status
	framerSvc = MustOpen(framer.OpenService(ctx, validCfg))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
