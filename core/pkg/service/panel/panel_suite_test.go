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
	RunSpecs(t, "Panel Suite")
}

var (
	dist       mock.Node
	db         *gorp.DB
	otg        *ontology.Ontology
	svc        *panel.Service
	channelSvc *channel.Service
	parentID   ontology.ID
	tx         gorp.Tx
)

var _ = BeforeSuite(func(ctx SpecContext) {
	dist = mock.NewNode(ctx)
	db = dist.DB
	otg = dist.Ontology
	channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
		Channel:      dist.Channel,
		DB:           dist.DB,
		HostResolver: dist.Cluster,
		Ontology:     dist.Ontology,
		Group:        dist.Group,
		Search:       dist.Search,
	}))
	labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Search:   dist.Search,
	}))
	statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Label:    labelSvc,
		Search:   dist.Search,
	}))
	framerSvc := MustOpen(framer.OpenService(ctx, framer.ServiceConfig{
		Framer:       dist.Framer,
		Channel:      channelSvc,
		DB:           dist.DB,
		Status:       statusSvc,
		HostResolver: dist.Cluster,
	}))
	sigs := MustSucceed(signals.New(signals.Config{
		Channel: channelSvc,
		Framer:  framerSvc,
	}))
	svc = MustOpen(panel.OpenService(ctx, panel.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Search:   dist.Search,
		Signals:  sigs,
	}))
	userSvc := MustOpen(user.OpenService(ctx, user.ServiceConfig{
		DB:       dist.DB,
		Ontology: dist.Ontology,
		Group:    dist.Group,
		Search:   dist.Search,
	}))
	parent := MustSucceed(userSvc.NewWriter(nil).Create(ctx, user.User{Username: "panel-parent"}))
	parentID = user.OntologyID(parent.Key)
})

var _ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
