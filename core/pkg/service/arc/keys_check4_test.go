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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Compile keys check 4", func() {
	It("Should read the stored deployed lists of a graph whose Driver runs the "+
		"current config", func(ctx SpecContext) {
		in := channel.Channel{Name: "kc4_in", Virtual: true, DataType: telem.Float64T}
		out := channel.Channel{Name: "kc4_out", Virtual: true, DataType: telem.Float64T}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := channelSvc.NewWriter(tx)
			if err := w.Create(ctx, &in); err != nil {
				return err
			}
			return w.Create(ctx, &out)
		})).To(Succeed())
		a := newAutomation(arc.Base{
			Name: "kc4_graph",
			Source: arc.Source{Variant: arc.GraphSource{Graph: graph.Graph{
				Nodes: graph.Nodes{{Key: "source"}, {Key: "sink"}},
				Edges: graph.Edges{{
					Key:    "edge",
					Source: ir.Handle{Node: "source", Param: ir.DefaultOutputParam},
					Target: ir.Handle{Node: "sink", Param: ir.DefaultInputParam},
				}},
				Inputs: map[string]msgpack.EncodedJSON{
					"source": {"type": "on", "channel": float64(in.Key())},
					"sink":   {"type": "write", "channel": float64(out.Key())},
				},
			}}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		key := a.Base().Key
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, key)
			})).To(Succeed())
		})
		var tsk *task.Task
		Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
			tsk, err = svc.NewWriter(tx).UpdateTask(
				ctx,
				key,
				arc.TaskUpdate{Rack: new(testRack.Key)},
			)
			return err
		})).To(Succeed())
		MustSucceed(svc.DeployProgram(ctx, key, testRack.Key))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			var stored task.Task
			if err := taskSvc.NewRetrieve().
				Where(task.MatchKeys(tsk.Key)).
				Entry(&stored).
				Exec(ctx, tx); err != nil {
				return err
			}
			return statusSvc.NewWriter(tx).Set(ctx, &task.Status{
				Owner:   new(task.ResourceID(tsk.Key)),
				Name:    "kc4",
				Variant: status.VariantSuccess,
				Time:    telem.Now(),
				Details: task.StatusDetails{
					Running:    true,
					ConfigHash: stored.ConfigHash,
				},
			})
		})).To(Succeed())
		rec := MustSucceed(svc.RetrieveDrift(ctx, db, key))[key]
		// The Arc's own lists leave out a graph's node channels; the deployed lists
		// hold them.
		Expect([2][]channel.Key{rec.Reads, rec.Writes}).
			To(Equal([2][]channel.Key{{in.Key()}, {out.Key()}}))
	})
})
