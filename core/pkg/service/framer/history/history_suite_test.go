// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package history_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	calcgraph "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/graph"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/framer/history"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	svcsignals "github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	timetestutil "github.com/synnaxlabs/x/time/testutil"
)

func TestHistory(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Framer History Suite", SuiteConfig())
}

const flushInterval = 100 * telem.Millisecond

var (
	channelSvc *channel.Service
	framerSvc  *framer.Service
	// clock only advances when a spec flushes, so every change before it merges.
	clock = timetestutil.NewClock(time.Now())
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := testutil.NewNode(ctx)
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
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Status:       statusSvc,
	}))
	channelGraph := MustOpen(calcgraph.Open(ctx, calcgraph.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Status:  statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		DB:           node.DB,
		Framer:       node.Framer,
		Channel:      channelSvc,
		ChannelGraph: channelGraph,
	}))
	sigs := MustSucceed(svcsignals.New(svcsignals.Config{
		DB:      node.DB,
		Channel: channelSvc,
		Framer:  framerSvc,
	}))
	MustOpen(history.Publish(ctx, history.Config{
		Framer:        node.Framer,
		Signals:       sigs,
		FlushInterval: flushInterval,
		Clock:         clock,
	}))
})

var _ = ShouldNotLeakGoroutinesPerSpec()
