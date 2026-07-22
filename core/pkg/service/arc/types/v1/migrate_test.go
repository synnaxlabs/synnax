// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	graph "github.com/synnaxlabs/arc/graph/types/v0"
	ir "github.com/synnaxlabs/arc/ir/types/v0"
	text "github.com/synnaxlabs/arc/text/types/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/arc/types/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateArc", func() {
	It("Should carry the arc's key, name, mode, graph, and text", func(ctx SpecContext) {
		key := uuid.New()
		migrated := MustSucceed(v1.MigrateArc(ctx, v0.Arc{
			Key:  key,
			Name: "my-arc",
			Mode: v0.ModeText,
			Text: text.Text{Raw: "x := 1"},
			Graph: graph.Graph{
				Functions: ir.Functions{{Key: "scale", Body: ir.Body{Raw: "x * 2"}}},
			},
		}))
		Expect(migrated.Key).To(Equal(key))
		Expect(migrated.Name).To(Equal("my-arc"))
		Expect(migrated.Mode).To(Equal(v1.ModeText))
		Expect(migrated.Text.Raw).To(Equal("x := 1"))
		Expect(migrated.Graph.Functions).To(HaveLen(1))
		Expect(migrated.Graph.Functions[0].Key).To(Equal("scale"))
	})

	It("Should drop the persisted program and status", func(ctx SpecContext) {
		migrated := MustSucceed(v1.MigrateArc(ctx, v0.Arc{
			Key:    uuid.New(),
			Name:   "loaded",
			Status: &v0.Status{Name: "running", Variant: "success"},
		}))
		Expect(migrated.Status).To(BeNil())
		Expect(migrated.Program).To(BeNil())
	})
})

var _ = Describe("Migrations", func() {
	migrateSeed := func(ctx SpecContext, seed v0.Arc) v1.Arc {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v0.Key, v0.Arc]{DB: db}))
		Expect(gorp.NewCreate[v0.Key, v0.Arc]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Arc",
			Migrations: v1.Migrations,
		})).To(Succeed())
		var got v1.Arc
		Expect(gorp.NewRetrieve[v1.Key, v1.Arc]().
			Where(gorp.MatchKeys[v1.Key, v1.Arc](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		return got
	}

	It("Should rename deprecated set_status nodes to status.set", func(ctx SpecContext) {
		got := migrateSeed(ctx, v0.Arc{
			Key:  uuid.New(),
			Name: "Legacy Status Graph",
			Mode: v0.ModeGraph,
			Graph: graph.Graph{
				Nodes: graph.Nodes{
					{
						Key:  "alarm",
						Type: "set_status",
						Config: msgpack.EncodedJSON{
							"statusKey":   "ox_alarm",
							"variant":     "error",
							"message":     "Overpressure",
							"description": "dropped on migrate",
						},
					},
					{Key: "scale", Type: "scale", Config: msgpack.EncodedJSON{"factor": "2"}},
				},
			},
		})
		Expect(got.Graph.Nodes).To(HaveLen(2))
		alarm := got.Graph.Nodes[0]
		Expect(alarm.Type).To(Equal("status.set"))
		Expect(alarm.Config).To(Equal(msgpack.EncodedJSON{
			"key_or_name": "ox_alarm",
			"variant":     "error",
			"message":     "Overpressure",
		}))
		scale := got.Graph.Nodes[1]
		Expect(scale.Type).To(Equal("scale"))
		Expect(scale.Config).To(Equal(msgpack.EncodedJSON{"factor": "2"}))
	})

	It("Should default missing set_status config parameters", func(ctx SpecContext) {
		got := migrateSeed(ctx, v0.Arc{
			Key:   uuid.New(),
			Name:  "Bare Status Node",
			Mode:  v0.ModeGraph,
			Graph: graph.Graph{Nodes: graph.Nodes{{Key: "alarm", Type: "set_status"}}},
		})
		Expect(got.Graph.Nodes).To(HaveLen(1))
		Expect(got.Graph.Nodes[0].Type).To(Equal("status.set"))
		Expect(got.Graph.Nodes[0].Config).To(Equal(msgpack.EncodedJSON{
			"key_or_name": "",
			"variant":     "success",
			"message":     "",
		}))
	})
})
