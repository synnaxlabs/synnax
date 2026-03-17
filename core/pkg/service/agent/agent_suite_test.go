// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package agent_test

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/agent"
	"github.com/synnaxlabs/synnax/pkg/service/agent/llm"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func TestAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}

var (
	ctx      = context.Background()
	db       *gorp.DB
	otg      *ontology.Ontology
	dist     mock.Node
	groupSvc *group.Service
	labelSvc *label.Service
	statSvc  *status.Service
	rackSvc  *rack.Service
	taskSvc  *task.Service
	arcSvc   *arc.Service
	agentSvc *agent.Service
	testRack *rack.Rack
	tx       gorp.Tx
)

var _ = BeforeSuite(func() {
	db = gorp.Wrap(memkv.New())
	otg = MustSucceed(ontology.Open(ctx, ontology.Config{
		EnableSearch: new(true),
		DB:           db,
	}))
	distB := mock.NewCluster()
	dist = distB.Provision(ctx)

	groupSvc = MustSucceed(group.OpenService(ctx, group.ServiceConfig{
		DB:       db,
		Ontology: otg,
	}))
	labelSvc = MustSucceed(label.OpenService(ctx, label.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
	}))
	statSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Label:    labelSvc,
	}))
	rackSvc = MustSucceed(rack.OpenService(ctx, rack.ServiceConfig{
		DB:                  db,
		Ontology:            otg,
		Group:               groupSvc,
		HostProvider:        mock.StaticHostKeyProvider(1),
		Status:              statSvc,
		HealthCheckInterval: 10 * telem.Millisecond,
	}))
	taskSvc = MustSucceed(task.OpenService(ctx, task.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Group:    groupSvc,
		Rack:     rackSvc,
		Status:   statSvc,
	}))
	testRack = &rack.Rack{Name: "Test Rack"}
	Expect(rackSvc.NewWriter(db).Create(ctx, testRack)).To(Succeed())

	arcSvc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Channel:  dist.Channel,
		Task:     taskSvc,
	}))

	agentSvc = MustSucceed(agent.OpenService(ctx, agent.ServiceConfig{
		DB:       db,
		Ontology: otg,
		Channel:  dist.Channel,
		Arc:      arcSvc,
		Task:     taskSvc,
		Framer:   dist.Framer,
		LLM: llm.Config{
			APIKey:  os.Getenv("SYNNAX_LLM_API_KEY"),
			Model:   "gpt-4o",
			BaseURL: "https://api.openai.com/v1",
		},
	}))

	// Create test channels that the agent will discover
	ch1 := &channel.Channel{Name: "pump_3_current", DataType: telem.Float64T, Virtual: true}
	Expect(dist.Channel.Create(ctx, ch1)).To(Succeed())
	ch2 := &channel.Channel{Name: "pump_3_status", DataType: telem.Uint8T, Virtual: true}
	Expect(dist.Channel.Create(ctx, ch2)).To(Succeed())

})

var (
	_ = AfterSuite(func() {
		Expect(agentSvc.Close()).To(Succeed())
		Expect(arcSvc.Close()).To(Succeed())
		Expect(taskSvc.Close()).To(Succeed())
		Expect(rackSvc.Close()).To(Succeed())
		Expect(statSvc.Close()).To(Succeed())
		Expect(labelSvc.Close()).To(Succeed())
		Expect(groupSvc.Close()).To(Succeed())
		Expect(dist.Close()).To(Succeed())
		Expect(otg.Close()).To(Succeed())
		Expect(db.Close()).To(Succeed())
	})
	_ = BeforeEach(func() { tx = db.OpenTx() })
	_ = AfterEach(func() { Expect(tx.Close()).To(Succeed()) })
)
