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
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	dist mock.Node
	svc  *channel.Service
)

func TestChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Channel Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// openService opens a channel service for the node and creates the node's internal
// control channel, mirroring what the service layer's OpenLayer does in production.
// Tests rely on the control channel for their local-key and channel-count expectations.
// Extra configs override the derived distribution-layer fields.
func openService(ctx context.Context, n mock.Node, cfgs ...channel.ServiceConfig) *channel.Service {
	GinkgoHelper()
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       n.DB,
		Ontology: n.Ontology,
		Group:    n.Group,
		Search:   n.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       n.DB,
		Group:    n.Group,
		Ontology: n.Ontology,
		Label:    labelSvc,
		Search:   n.Search,
	}))
	base := channel.ServiceConfig{
		Channel:      n.Channel,
		DB:           n.DB,
		HostResolver: n.Cluster,
		Ontology:     n.Ontology,
		Group:        n.Group,
		Search:       n.Search,
		Status:       statusSvc,
	}
	channelSvc := MustOpen(channel.OpenService(ctx, append([]channel.ServiceConfig{base}, cfgs...)...))
	controlCh := channel.Channel{
		Name:        fmt.Sprintf("sy_node_%v_control", n.Cluster.HostKey()),
		Leaseholder: n.Cluster.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}
	Expect(channelSvc.Create(ctx, &controlCh, channel.RetrieveIfNameExists())).To(Succeed())
	Expect(n.Framer.ConfigureControlUpdateChannel(ctx, controlCh.Key(), controlCh.Name)).To(Succeed())
	return channelSvc
}

var _ = BeforeSuite(func(ctx SpecContext) {
	dist = mock.MustOpenNode(ctx)
	svc = openService(ctx, dist)
})
