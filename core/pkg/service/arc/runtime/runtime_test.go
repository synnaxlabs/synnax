package runtime_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc/runtime"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Runtime", Ordered, func() {
	var (
		dist      mock.Node
		statusSvc *status.Service
	)

	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		statusSvc = MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Group:    dist.Group,
			Signals:  dist.Signals,
			Ontology: dist.Ontology,
		}))
	})

	AfterAll(func() {
		Expect(dist.Close()).To(Succeed())
		Expect(statusSvc.Close()).To(Succeed())
	})

	It("Should run a basic value printer", func() {
		ch := &channel.Channel{
			Name:     "ox_pt_1",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(dist.Channel.Create(ctx, ch)).To(Succeed())

		cfg := runtime.Config{
			Channel: dist.Channel,
			Framer:  dist.Framer,
			Status:  statusSvc,
		}

		resolver := MustSucceed(runtime.CreateResolver(cfg))

		graph := arc.Graph{
			Nodes: []graph.Node{
				{Node: arc.Node{
					Key:    "first",
					Type:   "on",
					Config: map[string]any{"channel": ch.Key()},
				}},
				{Node: arc.Node{Key: "printer", Type: "printer"}},
			},
			Edges: []arc.Edge{
				{
					Source: arc.Handle{Node: "first"},
					Target: arc.Handle{Node: "printer"},
				},
			},
		}
		cfg.Module = MustSucceed(arc.CompileGraph(ctx, graph, arc.WithResolver(resolver)))
		Expect(cfg.Module.Nodes).To(HaveLen(2))
		Expect(cfg.Module.Edges).To(HaveLen(1))

		r := MustSucceed(runtime.Open(ctx, cfg))
		Expect(r.Close()).To(Succeed())
	})
})
