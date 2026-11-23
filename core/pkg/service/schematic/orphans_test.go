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
	"encoding/json/v2"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	. "github.com/synnaxlabs/x/testutil"
)

// The vectors pin the save confirm of the TypeScript client to what a symbol save
// removes.
const orphanVectorsPath = "../../../testdata/orphans/vectors.json"

type orphanBindings struct {
	States   []schematic.StateBinding   `json:"states"`
	Commands []schematic.CommandBinding `json:"commands"`
}

type orphanOverride struct {
	Key    string   `json:"key"`
	Fields []string `json:"fields"`
}

type orphans struct {
	States    []string         `json:"states"`
	Overrides []orphanOverride `json:"overrides"`
	Commands  []string         `json:"commands"`
}

type orphanCase struct {
	Name string `json:"name"`
	// After holds the states and commands that the save sets, in their stored form. A
	// case that leaves one out keeps that of Before.
	After struct {
		States   []any `json:"states"`
		Commands []any `json:"commands"`
	} `json:"after"`
	Orphans orphans `json:"orphans"`
}

type orphanVectors struct {
	Before struct {
		States   []symbol.State   `json:"states"`
		Commands []symbol.Command `json:"commands"`
	} `json:"before"`
	Bindings orphanBindings `json:"bindings"`
	Cases    []orphanCase   `json:"cases"`
}

// removed returns the bindings and overrides that a save took from before to leave cfg.
func removed(before orphanBindings, cfg schematic.SymbolNodeConfig) orphans {
	res := orphans{
		States:    []string{},
		Overrides: []orphanOverride{},
		Commands:  []string{},
	}
	for _, b := range before.States {
		kept, ok := lo.Find(cfg.States, func(k schematic.StateBinding) bool {
			return k.Key == b.Key
		})
		if !ok {
			res.States = append(res.States, b.Key)
			continue
		}
		o := orphanOverride{Key: b.Key}
		for _, f := range []struct {
			name        string
			set, stayed bool
		}{
			{"bounds", b.Bounds != nil, kept.Bounds != nil},
			{"enumeration", b.Enumeration != nil, kept.Enumeration != nil},
			{"match", b.Match != nil, kept.Match != nil},
			{"default", b.Default != nil, kept.Default != nil},
		} {
			if f.set && !f.stayed {
				o.Fields = append(o.Fields, f.name)
			}
		}
		if len(o.Fields) > 0 {
			res.Overrides = append(res.Overrides, o)
		}
	}
	for _, b := range before.Commands {
		if !lo.ContainsBy(cfg.Commands, func(k schematic.CommandBinding) bool {
			return k.Key == b.Key
		}) {
			res.Commands = append(res.Commands, b.Key)
		}
	}
	return res
}

var _ = Describe("Orphan vectors", func() {
	var v orphanVectors
	Expect(json.Unmarshal(
		MustSucceed(os.ReadFile(orphanVectorsPath)),
		&v,
		json.RejectUnknownMembers(true),
	)).To(Succeed())
	BeforeEach(func(ctx SpecContext) {
		enums := lo.FilterMap(
			v.Bindings.States,
			func(b schematic.StateBinding, _ int) (enumeration.Enumeration, bool) {
				if b.Enumeration == nil {
					return enumeration.Enumeration{}, false
				}
				return enumeration.Enumeration{Key: *b.Enumeration, Name: b.Key}, true
			},
		)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return enumSvc.NewWriter(tx).CreateMany(ctx, enums)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return enumSvc.NewWriter(tx).Delete(
					ctx,
					lo.Map(enums, func(e enumeration.Enumeration, _ int) enumeration.Key {
						return e.Key
					})...,
				)
			})).To(Succeed())
		})
	})
	DescribeTable(
		"Should remove the orphans of a save",
		func(ctx SpecContext, c orphanCase) {
			sym := heater()
			sym.Data.States, sym.Data.Commands = v.Before.States, v.Before.Commands
			createSymbol(ctx, &sym)
			s := schematic.Schematic{
				Project: &proj.Key,
				Name:    "orphans",
				Nodes: []schematic.Node{symbolNode("n", schematic.SymbolNodeConfig{
					Spec:     customRef(sym),
					States:   v.Bindings.States,
					Commands: v.Bindings.Commands,
				})},
			}
			createSchematic(ctx, &s)
			var acts []symbol.Action
			for field, value := range map[string][]any{
				"states":   c.After.States,
				"commands": c.After.Commands,
			} {
				if value != nil {
					acts = append(acts, symbol.NewSetAction(symbol.SetPayload{
						Path:  []string{"data", field},
						Value: value,
					}))
				}
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.Symbol.NewWriter(tx).Apply(ctx, sym.Key, acts...)
			})).To(Succeed())
			var stored schematic.Schematic
			Expect(svc.NewRetrieve().
				Where(schematic.MatchKeys(s.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			cfg := as[schematic.SymbolNodeConfig](stored.Nodes[0].Config.Variant)
			Expect(removed(v.Bindings, cfg)).To(Equal(c.Orphans))
		},
		lo.Map(v.Cases, func(c orphanCase, _ int) TableEntry {
			return Entry(c.Name, c)
		}),
	)
})
