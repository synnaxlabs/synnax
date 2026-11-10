// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/patch"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

type configsUpdate = patch.TypedMap[
	string,
	schematic.ElementConfig,
	schematic.ElementConfigUpdate,
]

var _ = Describe("Update", func() {
	var s schematic.Schematic
	BeforeEach(func() {
		s = schematic.Schematic{
			Name: "s",
			Configs: map[string]schematic.ElementConfig{
				"t1": tankCfg("Tank", ""),
				"e1": pipeCfg(""),
			},
		}
	})
	update := func(configs configsUpdate) (schematic.Schematic, error) {
		return schematic.Reduce(s, schematic.NewUpdateAction(
			schematic.UpdatePayload{Value: schematic.Update{Configs: &configs}},
		))
	}

	It("Should merge an update into the config under its key", func() {
		label := patch.Merge[schematic.LabelConfig](
			schematic.LabelConfigUpdate{Label: new("Ox")},
		)
		next := MustSucceed(update(configsUpdate{
			Update: map[string]schematic.ElementConfigUpdate{
				"t1": {Variant: schematic.TankElementConfigUpdate{Label: &label}},
			},
		}))
		Expect(next.Configs["t1"]).To(Equal(tankCfg("Ox", "")))
		Expect(next.Configs["e1"]).To(Equal(pipeCfg("")))
		Expect(s.Configs["t1"]).To(Equal(tankCfg("Tank", "")))
	})

	It("Should name the key when a new variant needs a field", func() {
		next, err := update(configsUpdate{
			Update: map[string]schematic.ElementConfigUpdate{
				"t1": {Variant: schematic.PipeElementConfigUpdate{}},
			},
		})
		Expect(err).To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("t1: segments is required")),
		))
		Expect(next).To(Equal(s))
	})

	It("Should insert and remove configs by key", func() {
		next := MustSucceed(update(configsUpdate{
			Insert: map[string]schematic.ElementConfig{"t2": tankCfg("Fuel", "")},
			Remove: []string{"e1"},
		}))
		Expect(next.Configs).To(Equal(map[string]schematic.ElementConfig{
			"t1": tankCfg("Tank", ""),
			"t2": tankCfg("Fuel", ""),
		}))
	})

	It("Should leave the schematic alone when a config is missing", func() {
		next, err := update(configsUpdate{Remove: []string{"n9"}})
		Expect(err).To(And(
			MatchError(patch.ErrStale),
			MatchError(ContainSubstring("n9 not found")),
		))
		Expect(next).To(Equal(s))
	})
})
