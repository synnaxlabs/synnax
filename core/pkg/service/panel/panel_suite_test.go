// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package panel_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/panel"
	"github.com/synnaxlabs/synnax/pkg/service/signals"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
	. "github.com/synnaxlabs/x/testutil"
)

func TestPanel(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Panel Suite")
}

var (
	node       mock.Node
	db         *gorp.DB
	otg        *ontology.Ontology
	svc        *panel.Service
	writer     panel.Writer
	framerSvc  *framer.Service
	channelSvc *channel.Service
	parentID   ontology.ID
	tx         gorp.Tx
)

var _ = BeforeSuite(func(ctx SpecContext) {
	ShouldNotLeakGoroutines()
	node = mock.NewNode(ctx)
	db = node.DB
	otg = node.Ontology
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       node.DB,
		Ontology: node.Ontology,
		Group:    node.Group,
		Search:   node.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       node.DB,
		Ontology: node.Ontology,
		Group:    node.Group,
		Label:    labelSvc,
		Search:   node.Search,
	}))
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      node.Channel,
		DB:           node.DB,
		HostResolver: node.Cluster,
		Ontology:     node.Ontology,
		Group:        node.Group,
		Search:       node.Search,
		Status:       statusSvc,
	}))
	framerSvc = MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:       node.Framer,
		Channel:      channelSvc,
		Status:       statusSvc,
		HostResolver: node.Cluster,
	}))
	sigs := MustSucceed(signals.New(signals.Config{
		Channel: channelSvc,
		Framer:  framerSvc,
	}))
	svc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB:       node.DB,
		Ontology: node.Ontology,
		Search:   node.Search,
		Signals:  sigs,
	}))
	writer = svc.NewWriter(nil)
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       node.DB,
		Ontology: node.Ontology,
		Group:    node.Group,
		Search:   node.Search,
	}))
	parent := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Username: "panel-parent"}))
	parentID = user.OntologyID(parent.Key)
})

var _ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })

var _ = ShouldNotLeakGoroutinesPerSpec()
