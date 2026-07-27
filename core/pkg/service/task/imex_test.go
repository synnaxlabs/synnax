// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/search"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("ImEx", Ordered, func() {
	var (
		svc      *task.Service
		rackSvc  *rack.Service
		testRack *rack.Rack
	)
	BeforeAll(func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		otg := MustOpen(ontology.Open(ctx, ontology.Config{DB: db}))
		searchIdx := MustOpen(search.OpenIndex())
		g := MustOpen(group.OpenService(ctx, group.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Search:   searchIdx,
		}))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Search:   searchIdx,
		}))
		statSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Label:    labelSvc,
			Search:   searchIdx,
		}))
		rackSvc = MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:                  db,
			Ontology:            otg,
			Group:               g,
			HostProvider:        mock.NewStaticHostProvider(1),
			Status:              statSvc,
			HealthCheckInterval: 10 * telem.Millisecond,
			Search:              searchIdx,
		}))
		svc = MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       db,
			Ontology: otg,
			Group:    g,
			Rack:     rackSvc,
			Status:   statSvc,
			Search:   searchIdx,
			ImEx:     imex.NewService(),
		}))
		testRack = &rack.Rack{Name: "Test Rack"}
		Expect(rackSvc.NewWriter(nil).Create(ctx, testRack)).To(Succeed())
	})

	Describe("Export", func() {
		It("Should export a task's config flat with version, type, and name", func(ctx SpecContext) {
			t := &task.Task{
				Key:  task.NewKey(testRack.Key, 0),
				Name: "Exported Task",
				Type: "opc_read",
				Config: msgpack.EncodedJSON{
					"sample_rate": float64(25),
					"channels":    []any{"a", "b"},
				},
			}
			Expect(svc.NewWriter(nil).Create(ctx, t)).To(Succeed())

			env := MustSucceed(svc.Export(ctx, t.Key.OntologyID()))
			Expect(env.Version).To(Equal(task.Version))
			Expect(env.Type).To(Equal("opc_read"))
			Expect(env.Name).To(Equal("Exported Task"))

			var body map[string]any
			Expect(json.Unmarshal(MustSucceed(json.Marshal(env)), &body)).To(Succeed())
			// The driver reads the file as its config, so config fields sit flat at the
			// top level rather than nested under a "config" key.
			Expect(body).ToNot(HaveKey("config"))
			Expect(body["sample_rate"]).To(BeEquivalentTo(25))
			Expect(body["type"]).To(Equal("opc_read"))
			Expect(body["name"]).To(Equal("Exported Task"))
			Expect(body["version"]).To(BeEquivalentTo(1))
		})

		It("Should return not found for a missing key", func(ctx SpecContext) {
			id := task.NewKey(testRack.Key, 9999).OntologyID()
			Expect(svc.Export(ctx, id)).Error().To(MatchError(query.ErrNotFound))
		})
	})
})
