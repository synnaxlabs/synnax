// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	. "github.com/synnaxlabs/x/testutil"
)

func builtin(name string) symbol.Builtin {
	b, ok := lo.Find(symbol.Builtins(), func(b symbol.Builtin) bool {
		return b.Name == name
	})
	Expect(ok).To(BeTrue(), name)
	return b
}

type shipped struct {
	enumeration          enumeration.Key
	deactivate, activate string
}

func enumerationOf(st symbol.State) *enumeration.Key {
	enum, ok := st.Variant.(symbol.EnumerationState)
	Expect(ok).To(BeTrue())
	return enum.Enumeration
}

func commandNames(b symbol.Builtin) []string {
	return lo.Map(b.Commands, func(c symbol.Command, _ int) string { return c.Name })
}

var _ = Describe("Builtins", func() {
	It("Should name each built-in once", func() {
		names := lo.Map(symbol.Builtins(), func(b symbol.Builtin, _ int) string {
			return b.Name
		})
		Expect(names).To(HaveLen(85))
		Expect(lo.Uniq(names)).To(HaveLen(len(names)))
	})

	It("Should keep every rule of the symbol model", func(ctx SpecContext) {
		tx := DeferClose(db.OpenTx())
		for _, b := range symbol.Builtins() {
			sym := symbol.Symbol{Name: b.Name, Data: symbol.Spec{
				Drawing:  svg(`<svg/>`),
				Variant:  "static",
				Scale:    1,
				States:   b.States,
				Commands: b.Commands,
			}}
			Expect(svc.NewWriter(tx).Create(ctx, &sym)).To(Succeed(), b.Name)
		}
	})

	It("Should give each state a built-in enumeration", func() {
		keys := lo.Map(
			enumeration.Builtins(),
			func(e enumeration.Enumeration, _ int) enumeration.Key { return e.Key },
		)
		for _, b := range symbol.Builtins() {
			for _, st := range b.States {
				enum := enumerationOf(st)
				Expect(enum).ToNot(BeNil(), b.Name)
				Expect(keys).To(ContainElement(*enum), b.Name)
			}
		}
	})

	DescribeTable(
		"Should name the commands of each family",
		func(name string, want shipped) {
			b := builtin(name)
			Expect(b.States).To(HaveLen(1))
			Expect(enumerationOf(b.States[0])).To(HaveValue(Equal(want.enumeration)))
			Expect(commandNames(b)).To(Equal([]string{want.deactivate, want.activate}))
			for i, c := range b.Commands {
				Expect(c.State).To(HaveValue(Equal("active")))
				Expect(c.Value.Variant).
					To(Equal(symbol.FixedCommandValue{Value: float64(i)}))
			}
		},
		Entry("valve", "valve", shipped{enumeration.ValveKey, "Close", "Open"}),
		Entry(
			"local valve",
			"manual_valve",
			shipped{enumeration.ValveKey, "Close", "Open"},
		),
		Entry(
			"normally open valve",
			"normally_open_solenoid_valve",
			shipped{enumeration.NormallyOpenValveKey, "Open", "Close"},
		),
		Entry(
			"pump",
			"turbo_compressor",
			shipped{enumeration.PumpKey, "Stop", "Start"},
		),
		Entry(
			"agitator",
			"rotary_mixer",
			shipped{enumeration.PumpKey, "Stop", "Start"},
		),
		Entry(
			"switch",
			"thruster",
			shipped{enumeration.SwitchKey, "Turn off", "Turn on"},
		),
	)

	It("Should give a static picture no state or command", func() {
		b := builtin("t_junction")
		Expect(b.States).To(BeEmpty())
		Expect(b.Commands).To(BeEmpty())
	})

	It("Should give the light a state and no command", func() {
		b := builtin("light")
		Expect(b.States).To(HaveLen(1))
		Expect(enumerationOf(b.States[0])).To(HaveValue(Equal(enumeration.LightKey)))
		Expect(b.Commands).To(BeEmpty())
	})
})
