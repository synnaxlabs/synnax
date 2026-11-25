// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	channeltestutil "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/legacy/device"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// createApplyHolders creates a channel that a task and a line plot use, and returns the
// channel and the task.
func createApplyHolders(
	b *testing.B,
	layer *service.Layer,
	db *gorp.DB,
) (channel.Key, task.Key) {
	b.Helper()
	ctx := b.Context()
	ch := channel.Channel{
		Name:     channeltestutil.UniqueChannelName(),
		DataType: telem.Float32T,
		Virtual:  true,
	}
	proj := project.Project{Name: "bench"}
	dev := device.Device{
		Key:      channeltestutil.UniqueChannelName(),
		Rack:     layer.Rack.EmbeddedKey,
		Name:     "plc",
		Make:     "modbus",
		Model:    "modbus",
		Location: "127.0.0.1:502",
	}
	var tsk task.Task
	if err := db.WithTx(ctx, func(tx gorp.Tx) error {
		if err := layer.Project.NewWriter(tx).Create(ctx, &proj); err != nil {
			return err
		}
		if err := layer.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
			return err
		}
		if err := layer.Device.NewWriter(tx).Create(ctx, &dev); err != nil {
			return err
		}
		tsk = task.Task{
			Name: "modbus_read",
			Type: "modbus_read",
			Config: msgpack.EncodedJSON{
				"device": dev.Key,
				"channels": []any{map[string]any{
					"type":    "coil",
					"key":     "coil-1",
					"channel": ch.Key(),
				}},
			},
		}
		if err := layer.Task.NewWriter(tx).Create(ctx, &tsk); err != nil {
			return err
		}
		plot := defaults.Fill(lineplot.LinePlot{Name: "plot", Project: &proj.Key})
		plot.Channels.Y1 = []channel.Key{ch.Key()}
		return layer.LinePlot.NewWriter(tx).Create(ctx, &plot)
	}); err != nil {
		b.Fatalf("failed to create the holders: %v", err)
	}
	return ch.Key(), tsk.Key
}

// BenchmarkApply measures the delete of a channel that a task and a line plot use, with
// a pick that deletes the task, against today's two steps: delete the task, then the
// channel. Each run uses a transaction that it then discards. The pick target is at
// most 1.10 x today.
// openApplyBench opens a node and a service layer that close when b ends.
func openApplyBench(b *testing.B) (*service.Layer, *gorp.DB) {
	b.Helper()
	node := testutil.OpenNode(b.Context())
	layer, err := svctestutil.OpenLayer(b.Context(), node)
	if err != nil {
		b.Fatalf("failed to open the layer: %v", err)
	}
	b.Cleanup(func() {
		if err := errors.Join(layer.Close(), node.Close()); err != nil {
			b.Errorf("failed to close: %v", err)
		}
	})
	return layer, node.DB
}

// runApply runs op in a transaction b.N times, closing each transaction unwritten.
func runApply(b *testing.B, db *gorp.DB, op func(tx gorp.Tx) error) {
	b.Helper()
	b.ReportAllocs()
	for range b.N {
		tx := db.OpenTx()
		if err := errors.Join(op(tx), tx.Close()); err != nil {
			b.Fatalf("failed to delete: %v", err)
		}
	}
}

func BenchmarkApply(b *testing.B) {
	gomega.RegisterTestingT(b)
	ctx := b.Context()
	layer, db := openApplyBench(b)
	ch, tsk := createApplyHolders(b, layer, db)
	change := []resource.ID{channel.ResourceID(ch)}
	picks := map[string]impact.Choice{
		task.ResourceID(tsk).String(): impact.ChoiceDelete,
	}
	b.Run("today", func(b *testing.B) {
		runApply(b, db, func(tx gorp.Tx) error {
			if err := layer.Task.NewWriter(tx).Delete(ctx, tsk, false); err != nil {
				return err
			}
			return channeltestutil.Delete(ctx, layer.Channel, tx, ch)
		})
	})
	b.Run("picks", func(b *testing.B) {
		runApply(b, db, func(tx gorp.Tx) error {
			_, err := layer.Impact.Apply(
				ctx,
				tx,
				change,
				impact.ApplyOptions{Picks: picks},
			)
			return err
		})
	})
	visible := func(resource.ID) bool { return true }
	b.Run("plan", func(b *testing.B) {
		runApply(b, db, func(tx gorp.Tx) error {
			_, err := layer.Impact.Plan(
				ctx,
				tx,
				change,
				impact.PlanOptions{Visible: visible},
			)
			return err
		})
	})
	planned, err := layer.Impact.Plan(ctx, db, change, impact.PlanOptions{})
	if err != nil {
		b.Fatalf("failed to plan: %v", err)
	}
	b.Run("token", func(b *testing.B) {
		runApply(b, db, func(tx gorp.Tx) error {
			_, err := layer.Impact.Apply(ctx, tx, change, impact.ApplyOptions{
				Token:   planned.Token,
				Picks:   picks,
				Visible: visible,
			})
			return err
		})
	})
}
