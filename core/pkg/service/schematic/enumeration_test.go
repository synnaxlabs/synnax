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
	"github.com/synnaxlabs/gorp"
	. "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/x/defaults"
)

var _ = Describe("Enumeration references", func() {
	var (
		used, other enumeration.Enumeration
		diagram     schematic.Schematic
	)
	createEnumeration := func(ctx SpecContext, name string) enumeration.Enumeration {
		GinkgoHelper()
		created := []enumeration.Enumeration{{Name: name}}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return enumSvc.NewWriter(tx).CreateMany(ctx, created)
		})).To(Succeed())
		return created[0]
	}
	node := func(key string, cfg schematic.NodeConfigVariant) schematic.Node {
		return schematic.Node{
			Key:    key,
			Config: defaults.Fill(schematic.NodeConfig{Variant: cfg}),
		}
	}
	enumerationsOf := func(s schematic.Schematic) []*enumeration.Key {
		GinkgoHelper()
		keys := make([]*enumeration.Key, 0, len(s.Nodes))
		for _, n := range s.Nodes {
			switch cfg := n.Config.Variant.(type) {
			case schematic.SelectNodeConfig:
				keys = append(keys, cfg.Enumeration)
			case schematic.StateIndicatorNodeConfig:
				keys = append(keys, cfg.Enumeration)
			}
		}
		return keys
	}
	BeforeEach(func(ctx SpecContext) {
		used = createEnumeration(ctx, "Breaker")
		other = createEnumeration(ctx, "Valve")
		diagram = schematic.Schematic{
			Name:    "enumerations",
			Project: &proj.Key,
			Nodes: []schematic.Node{
				node("select", schematic.SelectNodeConfig{Enumeration: &used.Key}),
				node("indicator", schematic.StateIndicatorNodeConfig{
					Enumeration: &used.Key,
				}),
			},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &diagram)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := svc.NewWriter(tx).Delete(ctx, diagram.Key); err != nil {
					return err
				}
				return enumSvc.NewWriter(tx).Delete(ctx, used.Key, other.Key)
			})).To(Succeed())
		})
	})

	It("Should list a schematic whose select or indicator uses the enumeration", func(
		ctx SpecContext,
	) {
		Expect(enumSvc.Dependents(ctx, db, used.Key)).To(ConsistOf(
			schematic.ResourceID(diagram.Key),
		))
		Expect(enumSvc.Dependents(ctx, db, other.Key)).To(BeEmpty())
	})

	It("Should refuse the delete of an enumeration that a select uses", func(
		ctx SpecContext,
	) {
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return enumSvc.NewWriter(tx).Delete(ctx, used.Key)
		})).To(MatchError(gorp.RestrictError{
			Target:    gorp.NewTarget("enumeration", used.Key),
			Holder:    gorp.NewTarget("schematic", diagram.Key),
			Reference: "nodes.config.enumeration",
		}))
	})

	It("Should move each select and indicator to the replacing enumeration", func(
		ctx SpecContext,
	) {
		rec := &Recorder[schematic.Key, schematic.Action]{}
		DeferCleanup(svc.OnAction(rec.Record))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return enumSvc.NewWriter(tx).Replace(ctx, used.Key, other.Key)
		})).To(Succeed())
		var res schematic.Schematic
		Expect(svc.NewRetrieve().
			Where(schematic.MatchKeys(diagram.Key)).
			Entry(&res).
			Exec(ctx, db)).To(Succeed())
		want := []*enumeration.Key{&other.Key, &other.Key}
		Expect(enumerationsOf(res)).To(Equal(want))
		seen := rec.Snapshot()
		Expect(seen).To(HaveLen(1))
		Expect(seen[0].Key).To(Equal(diagram.Key))
		Expect(seen[0].Actions).To(HaveLen(1))
		Expect(seen[0].Actions[0].Create).ToNot(BeNil())
		Expect(seen[0].Actions[0].Create.Schematic).To(Equal(res))
	})
})
