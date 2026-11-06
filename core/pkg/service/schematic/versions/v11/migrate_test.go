// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("MigrateSchematic", func() {
	stored := func(raw msgpack.EncodedJSON) v10.ElementConfig {
		GinkgoHelper()
		return MustSucceed(v10.DecodeWithDefaults(raw))
	}

	It("Should leave the stroke width of a lifted edge absent", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"e1": stored(msgpack.EncodedJSON{"variant": "pipe"}),
			},
		}))
		pipe := out.Configs["e1"].Variant.(v11.PipeElementConfig)
		Expect(pipe.StrokeWidth).To(BeNil())
	})

	It("Should keep the values a v10 edge carries", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"e1": stored(msgpack.EncodedJSON{
					"variant":  "hydraulic",
					"segments": []any{map[string]any{"direction": "x", "length": 40.0}},
				}),
			},
		}))
		edge := out.Configs["e1"].Variant.(v11.HydraulicElementConfig)
		Expect(edge.Segments).To(HaveLen(1))
		Expect(edge.Segments[0].Length).To(BeEquivalentTo(40))
	})

	It("Should keep a node config", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"n1": stored(msgpack.EncodedJSON{"variant": "setpoint", "units": "mV"}),
			},
		}))
		setpoint := out.Configs["n1"].Variant.(v11.SetpointElementConfig)
		Expect(setpoint.Units).To(Equal("mV"))
		Expect(setpoint.InlineSize).To(BeEquivalentTo(100))
	})

	It("Should keep the schematic's identity", func(ctx SpecContext) {
		key := uuid.New()
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Key:      key,
			Name:     "Feed system",
			Snapshot: true,
		}))
		Expect(out.Key).To(Equal(key))
		Expect(out.Name).To(Equal("Feed system"))
		Expect(out.Snapshot).To(BeTrue())
	})
})

var _ = Describe("Migration", func() {
	It("Should lift stored schematics to v11", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		old := v10.Schematic{
			Key:  uuid.New(),
			Name: "Feed system",
			Configs: map[string]v10.ElementConfig{
				"e1": MustSucceed(v10.DecodeWithDefaults(msgpack.EncodedJSON{
					"variant": "pipe",
				})),
			},
		}
		MustSucceed(
			gorp.OpenTable(ctx, gorp.TableConfig[v10.Key, v10.Schematic]{DB: db}),
		)
		Expect(gorp.NewCreate[v10.Key, v10.Schematic]().
			Entry(&old).Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Schematic",
			Migrations: []gorp.Migration{v11.Migration},
		})).To(Succeed())
		var got v11.Schematic
		Expect(gorp.NewRetrieve[v11.Key, v11.Schematic]().
			Where(gorp.MatchKeys[v11.Key, v11.Schematic](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal("Feed system"))
		pipe := got.Configs["e1"].Variant.(v11.PipeElementConfig)
		Expect(pipe.StrokeWidth).To(BeNil())
	})

	It("Should not reuse a v10 migration key", func() {
		Expect(v11.Migration.Key()).ToNot(Equal(v10.Migration.Key()))
	})
})
