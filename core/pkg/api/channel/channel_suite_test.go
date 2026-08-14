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
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	apicfg "github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	svc "github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAPIChannel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Channel Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	channelSvc *channel.Service
	apiSvc     *apichannel.Service
)

// openService opens a service-layer channel service for the node and creates the node's
// internal control channel, mirroring what the service layer's OpenLayer does in
// production.
func openService(ctx context.Context, node mock.Node) *channel.Service {
	GinkgoHelper()
	otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: node.DB}))
	searchIdx := MustOpen(search.OpenIndex())
	groupSvc := MustOpen(group.OpenService(ctx, group.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Search:   searchIdx,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: otg,
		Group:    groupSvc,
		Search:   searchIdx,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Group:    groupSvc,
		Ontology: otg,
		Label:    labelSvc,
		Search:   searchIdx,
	}))
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostProvider: node.Cluster,
		Ontology:     otg,
		Group:        groupSvc,
		Search:       searchIdx,
		Status:       statusSvc,
	}))
	controlCh := channel.Channel{
		Name:        fmt.Sprintf("sy_node_%v_control", node.Cluster.HostKey()),
		Leaseholder: node.Cluster.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}
	Expect(channelSvc.NewWriter(nil).Create(ctx, &controlCh)).To(Succeed())
	Expect(node.Framer.ConfigureControlUpdateChannel(
		ctx, controlCh.Key(), controlCh.Name,
	)).To(Succeed())
	return channelSvc
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node := mock.NewNode(ctx)
	channelSvc = openService(ctx, node)
	apiSvc = MustSucceed(apichannel.NewService(apicfg.LayerConfig{
		Distribution: &distribution.Layer{DB: node.DB},
		Service:      &svc.Layer{Channel: channelSvc},
	}))
})
