// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/search"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestArc(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Arc Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()

var (
	db         *gorp.DB
	otg        *ontology.Ontology
	svc        *arc.Service
	channelSvc *channel.Service
	tx         gorp.Tx
	taskSvc    *task.Service
	testRack   *rack.Rack
)

var (
	_ = BeforeSuite(func(ctx SpecContext) {
		searchIdx := MustOpen(search.Open())
		node := mock.MustOpenNode(ctx)
		db = node.DB
		otg = node.Ontology
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    node.Group,
			Search:   searchIdx,
		}))
		statSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    node.Group,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		channelSvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
			Channel:      node.Channel,
			DB:           node.DB,
			HostResolver: node.Cluster,
			Ontology:     node.Ontology,
			Group:        node.Group,
			Search:       node.Search,
			Status:       statSvc,
		}))
		rackSvc := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               node.Group,
			HostProvider:        mock.NewStaticHostProvider(1),
			Status:              statSvc,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		taskSvc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    node.Group,
			Rack:     rackSvc,
			Status:   statSvc,
			Search:   searchIdx,
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(rackSvc.NewWriter(db).Create(ctx, testRack)).To(Succeed())
		svc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Channel:  channelSvc,
			Task:     taskSvc,
			Search:   searchIdx,
		}))
	})
	_ = BeforeEach(func() { tx = DeferClose(db.OpenTx()) })
)
