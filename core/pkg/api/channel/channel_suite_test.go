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
	"github.com/synnaxlabs/synnax/pkg/service/label"
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
	node       mock.Node
	channelSvc *channel.Service
	apiSvc     *apichannel.Service
)

// openService opens a service-layer channel service for the node and creates the node's
// internal control channel, mirroring what the service layer's OpenLayer does in
// production.
func openService(ctx context.Context, node mock.Node) *channel.Service {
	GinkgoHelper()
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: node.Ontology,
		Group:    node.Group,
		Search:   node.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Group:    node.Group,
		Ontology: node.Ontology,
		Label:    labelSvc,
		Search:   node.Search,
	}))
	channelSvc := MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostResolver: node.Cluster,
		Ontology:     node.Ontology,
		Group:        node.Group,
		Search:       node.Search,
		Status:       statusSvc,
	}))
	controlCh := channel.Channel{
		Name:        fmt.Sprintf("sy_node_%v_control", node.Cluster.HostKey()),
		Leaseholder: node.Cluster.HostKey(),
		Virtual:     true,
		DataType:    telem.StringT,
		Internal:    true,
	}
	Expect(channelSvc.
		NewWriter(nil).
		Create(ctx, &controlCh, channel.RetrieveIfNameExists()),
	).To(Succeed())
	Expect(node.Framer.ConfigureControlUpdateChannel(
		ctx, controlCh.Key(), controlCh.Name,
	)).To(Succeed())
	return channelSvc
}

var _ = BeforeSuite(func(ctx SpecContext) {
	node = mock.NewNode(ctx)
	channelSvc = openService(ctx, node)
	apiSvc = MustSucceed(apichannel.NewService(apicfg.LayerConfig{
		Distribution: &distribution.Layer{DB: node.DB},
		Service:      &svc.Layer{Channel: channelSvc},
	}))
})
