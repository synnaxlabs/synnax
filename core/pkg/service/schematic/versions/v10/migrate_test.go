// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v10_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v9 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v9"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// stored decodes a v9 config the way the v9 Core persists it.
func stored(raw msgpack.EncodedJSON) v9.ElementConfig {
	GinkgoHelper()
	return MustSucceed(v9.DecodeWithDefaults(raw))
}

var _ = Describe("MigrateSchematic", func() {
	// lifted runs a single v9 config through the v10 migration and returns the lifted
	// variant.
	lifted := func(ctx SpecContext, raw msgpack.EncodedJSON) v10.ElementConfigVariant {
		GinkgoHelper()
		out := MustSucceed(v10.MigrateSchematic(ctx, v9.Schematic{
			Configs: map[string]v9.ElementConfig{"n1": stored(raw)},
		}))
		return out.Configs["n1"].Variant
	}

	It("Should fill the defaults of the fields v10 adds", func(ctx SpecContext) {
		setpoint := lifted(ctx, msgpack.EncodedJSON{
			"variant": "setpoint",
		}).(v10.SetpointElementConfig)
		Expect(setpoint.InlineSize).To(BeEquivalentTo(100))
		Expect(setpoint.StalenessTimeout).To(BeEquivalentTo(5))
		Expect(setpoint.StateChannel).To(BeNil())
		input := lifted(ctx, msgpack.EncodedJSON{
			"variant": "input",
		}).(v10.InputElementConfig)
		Expect(input.InlineSize).To(BeEquivalentTo(150))
		Expect(input.StalenessTimeout).To(BeEquivalentTo(5))
		sel := lifted(ctx, msgpack.EncodedJSON{
			"variant": "select",
		}).(v10.SelectElementConfig)
		Expect(sel.StalenessTimeout).To(BeEquivalentTo(5))
	})

	It("Should keep every value the v9 config carries, zero included", func(
		ctx SpecContext,
	) {
		value := lifted(ctx, msgpack.EncodedJSON{
			"variant":   "value",
			"precision": 0.0,
			"units":     "kPa",
		}).(v10.ValueElementConfig)
		Expect(value.Precision).To(BeEquivalentTo(0))
		Expect(value.Units).To(Equal("kPa"))
		sel := lifted(ctx, msgpack.EncodedJSON{
			"variant":     "select",
			"inline_size": 240.0,
		}).(v10.SelectElementConfig)
		Expect(sel.InlineSize).To(BeEquivalentTo(240))
	})

	It("Should keep the units a v9 symbol showed by default", func(ctx SpecContext) {
		value := lifted(ctx, msgpack.EncodedJSON{
			"variant": "value",
		}).(v10.ValueElementConfig)
		Expect(value.Units).To(Equal("psi"))
	})

	It("Should keep the schematic's identity", func(ctx SpecContext) {
		key := uuid.New()
		out := MustSucceed(v10.MigrateSchematic(ctx, v9.Schematic{
			Key:      key,
			Name:     "Feed system",
			Snapshot: true,
		}))
		Expect(out.Key).To(Equal(key))
		Expect(out.Name).To(Equal("Feed system"))
		Expect(out.Snapshot).To(BeTrue())
	})
})

var _ = Describe("MigrateElementConfig", func() {
	It("Should copy the v9 values and leave the fields v10 adds unset", func(
		ctx SpecContext,
	) {
		cmd := channel.Key(7)
		out := MustSucceed(v10.MigrateElementConfig(ctx, v9.ElementConfig{
			Variant: v9.SetpointElementConfig{
				CommandChannel: &cmd,
				Units:          "kPa",
				Disabled:       true,
				OnClickDelay:   250,
			},
		}))
		setpoint := out.Variant.(v10.SetpointElementConfig)
		Expect(setpoint.CommandChannel).To(HaveValue(Equal(cmd)))
		Expect(setpoint.Units).To(Equal("kPa"))
		Expect(setpoint.Disabled).To(BeTrue())
		Expect(setpoint.OnClickDelay).To(BeEquivalentTo(250))
		Expect(setpoint.StateChannel).To(BeNil())
		Expect(setpoint.StalenessTimeout).To(BeZero())
		Expect(setpoint.InlineSize).To(BeZero())
	})

	It("Should keep a nil variant nil", func(ctx SpecContext) {
		out := MustSucceed(v10.MigrateElementConfig(ctx, v9.ElementConfig{}))
		Expect(out.Variant).To(BeNil())
	})
})

var _ = Describe("Migration", func() {
	It("Should lift stored schematics to v10", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		old := v9.Schematic{
			Key:  uuid.New(),
			Name: "Feed system",
			Configs: map[string]v9.ElementConfig{
				"n1": stored(msgpack.EncodedJSON{"variant": "setpoint", "units": "mV"}),
			},
		}
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v9.Key, v9.Schematic]{DB: db}))
		Expect(gorp.NewCreate[v9.Key, v9.Schematic]().
			Entry(&old).Exec(ctx, db)).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Schematic",
			Migrations: []gorp.Migration{v10.Migration},
		})).To(Succeed())
		var got v10.Schematic
		Expect(gorp.NewRetrieve[v10.Key, v10.Schematic]().
			Where(gorp.MatchKeys[v10.Key, v10.Schematic](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal("Feed system"))
		setpoint := got.Configs["n1"].Variant.(v10.SetpointElementConfig)
		Expect(setpoint.Units).To(Equal("mV"))
		Expect(setpoint.InlineSize).To(BeEquivalentTo(100))
	})

	It("Should not reuse a v9 migration key", func() {
		Expect(v10.Migration.Key()).ToNot(
			BeElementOf(v9.Migration.Key(), v9.ScaleMigration.Key()),
		)
	})
})
