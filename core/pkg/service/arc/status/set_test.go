// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/runtime/node"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	arcstatus "github.com/synnaxlabs/synnax/pkg/service/arc/status"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Symbols", func() {
	findByName := func(name string) *symbol.Symbol {
		for _, s := range arcstatus.NewSymbols() {
			if s.Name == name {
				return s
			}
			if s.Kind == symbol.KindModule {
				for _, c := range s.Children() {
					if c.Name == name {
						return c
					}
				}
			}
		}
		return nil
	}

	It("Should expose set_status as a deprecated bare global", func() {
		sym := findByName("set_status")
		Expect(sym).ToNot(BeNil())
		Expect(sym.Kind).To(Equal(symbol.KindFunction))
		Expect(sym.Deprecated).ToNot(BeNil())
	})

	It("Should expose set as a member of the status module", func() {
		sym := findByName("set")
		Expect(sym).ToNot(BeNil())
		Expect(sym.Kind).To(Equal(symbol.KindFunction))
	})

	It("Should declare four config parameters on set_status", func() {
		sym := findByName("set_status")
		Expect(sym.Type.Kind).To(Equal(types.KindFunction))
		Expect(sym.Type.Config).To(HaveLen(4))
		Expect(sym.Type.Config[0].Name).To(Equal("status_key"))
		Expect(sym.Type.Config[1].Name).To(Equal("variant"))
		Expect(sym.Type.Config[2].Name).To(Equal("message"))
		Expect(sym.Type.Config[3].Name).To(Equal("name"))
	})

	It("Should declare a single u8 input on set_status", func() {
		sym := findByName("set_status")
		Expect(sym.Type.Inputs).To(HaveLen(1))
		Expect(sym.Type.Inputs[0].Name).To(Equal(ir.DefaultOutputParam))
		Expect(sym.Type.Inputs[0].Type).To(Equal(types.U8()))
	})
})

var _ = Describe("Module", func() {
	var mod *arcstatus.Module

	BeforeEach(func(ctx SpecContext) {
		mod = arcstatus.NewModule(statSvc)
	})

	Describe("Create", func() {
		It("Should return not found for an unrecognized node type", func(ctx SpecContext) {
			cfg := node.Config{Node: ir.Node{Type: "wrong_type"}}
			Expect(mod.Create(ctx, cfg)).Error().To(MatchError(query.ErrNotFound))
		})

		It("Should create a node with valid config", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_status",
					Config: types.Params{
						{Name: "status_key", Value: "test_alarm"},
						{Name: "variant", Value: "success"},
						{Name: "message", Value: "All systems nominal"},
					},
				},
			}
			n := MustSucceed(mod.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})

		It("Should return an error when status_key is missing from config", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_status",
					Config: types.Params{
						{Name: "variant", Value: "success"},
						{Name: "message", Value: "msg"},
					},
				},
			}
			Expect(mod.Create(ctx, cfg)).Error().To(HaveOccurred())
		})

		It("Should return an error when variant is missing from config", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_status",
					Config: types.Params{
						{Name: "status_key", Value: "test_alarm"},
						{Name: "message", Value: "msg"},
					},
				},
			}
			Expect(mod.Create(ctx, cfg)).Error().To(HaveOccurred())
		})

		It("Should return an error when message is missing from config", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_status",
					Config: types.Params{
						{Name: "status_key", Value: "test_alarm"},
						{Name: "variant", Value: "success"},
					},
				},
			}
			Expect(mod.Create(ctx, cfg)).Error().To(HaveOccurred())
		})

		It("Should create a node with the qualified member type", func(ctx SpecContext) {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set",
					Config: types.Params{
						{Name: "status_key", Value: "test_alarm"},
						{Name: "variant", Value: "success"},
						{Name: "message", Value: "All systems nominal"},
					},
				},
			}
			n := MustSucceed(mod.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})

		It("Should populate the status from an existing entry in the service", func(ctx SpecContext) {
			existing := status.Status[any]{
				Key:     "pre_existing",
				Variant: xstatus.VariantError,
				Message: "old message",
				Time:    telem.Now(),
			}
			Expect(statSvc.NewWriter(nil).Set(ctx, &existing)).To(Succeed())
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_status",
					Config: types.Params{
						{Name: "status_key", Value: "pre_existing"},
						{Name: "variant", Value: "success"},
						{Name: "message", Value: "new message"},
					},
				},
			}
			n := MustSucceed(mod.Create(ctx, cfg))
			Expect(n).ToNot(BeNil())
		})
	})

	Describe("Node Behavior", func() {
		var n node.Node

		createNode := func(ctx context.Context, key, variant, message string) node.Node {
			cfg := node.Config{
				Node: ir.Node{
					Type: "set_status",
					Config: types.Params{
						{Name: "status_key", Value: key},
						{Name: "variant", Value: variant},
						{Name: "message", Value: message},
					},
				},
			}
			return MustSucceed(mod.Create(ctx, cfg))
		}

		Describe("IsOutputTruthy", func() {
			BeforeEach(func(ctx SpecContext) {
				n = createNode(ctx, "truthy_test", "info", "msg")
			})

			It("Should return false for any output index", func(ctx SpecContext) {
				Expect(n.IsOutputTruthy(0)).To(BeFalse())
				Expect(n.IsOutputTruthy(1)).To(BeFalse())
				Expect(n.IsOutputTruthy(-1)).To(BeFalse())
			})
		})

		Describe("Reset", func() {
			It("Should not panic", func(ctx SpecContext) {
				n = createNode(ctx, "reset_test", "info", "msg")
				Expect(func() { n.Reset() }).ToNot(Panic())
			})
		})

		Describe("Next", func() {
			It("Should set the status in the service", func(ctx SpecContext) {
				n = createNode(ctx, "next_test", "success", "All good")
				nodeCtx := node.Context{Context: ctx}
				n.Next(nodeCtx)
				var retrieved status.Status[any]
				Expect(statSvc.NewRetrieve().
					Where(status.MatchKeys[any]("next_test")).
					Entry(&retrieved).
					Exec(ctx, nil)).To(Succeed())
				Expect(retrieved.Key).To(Equal("next_test"))
				Expect(retrieved.Variant).To(Equal(xstatus.VariantSuccess))
				Expect(retrieved.Message).To(Equal("All good"))
				Expect(retrieved.Time).ToNot(BeZero())
			})

			It("Should update an existing status with a new timestamp", func(ctx SpecContext) {
				n = createNode(ctx, "timestamp_test", "warning", "Check this")
				nodeCtx := node.Context{Context: ctx}
				n.Next(nodeCtx)
				var first status.Status[any]
				Expect(statSvc.NewRetrieve().
					Where(status.MatchKeys[any]("timestamp_test")).
					Entry(&first).
					Exec(ctx, nil)).To(Succeed())

				n.Next(nodeCtx)
				var second status.Status[any]
				Expect(statSvc.NewRetrieve().
					Where(status.MatchKeys[any]("timestamp_test")).
					Entry(&second).
					Exec(ctx, nil)).To(Succeed())
				Expect(second.Time).To(BeNumerically(">=", first.Time))
			})

			It("Should preserve the variant set at creation time", func(ctx SpecContext) {
				n = createNode(ctx, "variant_test", "error", "Something broke")
				nodeCtx := node.Context{Context: ctx}
				n.Next(nodeCtx)
				var retrieved status.Status[any]
				Expect(statSvc.NewRetrieve().
					Where(status.MatchKeys[any]("variant_test")).
					Entry(&retrieved).
					Exec(ctx, nil)).To(Succeed())
				Expect(retrieved.Variant).To(Equal(xstatus.VariantError))
			})
		})
	})
})
