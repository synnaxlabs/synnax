// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer", func() {
	var (
		node     disttestutil.Node
		channels *channel.Service
		svc      *calculation.Service
	)
	BeforeEach(func(ctx SpecContext) {
		node = disttestutil.OpenNode(ctx)
		DeferCleanup(func() { Expect(node.Close()).To(Succeed()) })
		statuses := MustSucceed(
			status.OpenService(ctx, status.ServiceConfig{DB: node.DB}),
		)
		DeferCleanup(func() { Expect(statuses.Close()).To(Succeed()) })
		channels = MustSucceed(channel.OpenService(ctx, channel.ServiceConfig{
			Channel: node.Channel,
			TS:      node.Storage.TS,
			DB:      node.DB,
			HostKey: node.Identity.HostKey,
			Status:  statuses,
		}))
		DeferCleanup(func() { Expect(channels.Close()).To(Succeed()) })
		svc = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      node.DB,
			Channel: channels,
		}))
		DeferCleanup(func() { Expect(svc.Close()).To(Succeed()) })
	})

	createChannel := func(ctx SpecContext, ch channel.Channel) channel.Channel {
		GinkgoHelper()
		Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return channels.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	createVirtual := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		return createChannel(ctx, channel.Channel{
			Name:     name,
			DataType: telem.Float32T,
			Virtual:  true,
		})
	}
	create := func(ctx SpecContext, c *calculation.Calculation) error {
		return node.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, c)
		})
	}
	retrieve := func(ctx SpecContext, output channel.Key) []calculation.Calculation {
		GinkgoHelper()
		var out []calculation.Calculation
		Expect(svc.NewRetrieve().
			Where(calculation.MatchOutputs(output)).
			Entries(&out).
			Exec(ctx, node.DB)).To(Succeed())
		return out
	}

	Describe("Create", func() {
		It("Should create a calculation on a virtual channel", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(c))
		})

		It("Should fail on an output that stores its samples", func(ctx SpecContext) {
			index := createChannel(ctx, channel.Channel{
				Name:     "time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			stored := createChannel(ctx, channel.Channel{
				Name:       "pressure",
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			})
			c := calculation.Calculation{Output: stored.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(
					"channel pressure stores its samples, so no calculation writes it",
				)),
			))
		})

		It("Should fail on an output that another calculation writes", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			first := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &first)).To(Succeed())
			second := calculation.Calculation{Output: out.Key(), Expression: "return 3"}
			Expect(create(ctx, &second)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(
					ContainSubstring("another record already writes channel doubled"),
				),
			))
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(first))
		})

		It("Should fail on an output that does not exist", func(ctx SpecContext) {
			c := calculation.Calculation{Output: 1 << 20, Expression: "return 2"}
			Expect(create(ctx, &c)).To(MatchError(query.ErrNotFound))
		})

		It("Should fail on an empty expression", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key()}
			Expect(create(ctx, &c)).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, out.Key())).To(BeEmpty())
		})
	})

	Describe("Apply", func() {
		It("Should set the expression of a calculation", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return 4",
					},
				))
			})).To(Succeed())
			Expect(retrieve(ctx, out.Key())[0].Expression).To(Equal("return 4"))
		})

		It("Should keep a calculation whose new expression is empty", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{Path: []string{"expression"}, Value: ""},
				))
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(c))
		})
	})

	Describe("CreateOutput", func() {
		createOutput := func(ctx SpecContext, c *calculation.Calculation, name string) error {
			return node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).CreateOutput(ctx, c, name, nil)
				return err
			})
		}

		It(
			"Should make a virtual output typed by the expression, with a free index",
			func(
				ctx SpecContext,
			) {
				createChannel(ctx, channel.Channel{
					Name:     "wide",
					DataType: telem.Int64T,
					Virtual:  true,
				})
				c := calculation.Calculation{Expression: "return wide * 2"}
				Expect(createOutput(ctx, &c, "doubled")).To(Succeed())
				Expect(retrieve(ctx, c.Output)).To(ConsistOf(c))
				var output, index channel.Channel
				Expect(channels.NewRetrieve().
					Where(channel.MatchKeys(c.Output)).
					Entry(&output).
					Exec(ctx, node.DB)).To(Succeed())
				Expect(output.Name).To(Equal("doubled"))
				Expect(output.Virtual).To(BeTrue())
				Expect(output.DataType).To(Equal(telem.Int64T))
				Expect(output.Expression).To(BeEmpty())
				Expect(channels.NewRetrieve().
					Where(channel.MatchKeys(output.Index())).
					Entry(&index).
					Exec(ctx, node.DB)).To(Succeed())
				Expect(index.Name).To(Equal("doubled_time"))
				Expect(index.IsIndex).To(BeTrue())
				Expect(index.Virtual).To(BeTrue())
			},
		)

		It("Should fail on a calculation that names its output", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(
				createOutput(ctx, &c, "other"),
			).To(MatchError(validate.ErrValidation))
		})

		It("Should make no output for an expression that does not compile", func(
			ctx SpecContext,
		) {
			c := calculation.Calculation{Expression: "return missing * 2"}
			Expect(
				createOutput(ctx, &c, "doubled"),
			).To(MatchError(ContainSubstring("missing")))
			Expect(channels.NewRetrieve().
				Where(channel.MatchNames("doubled")).
				Exists(ctx, node.DB)).To(BeFalse())
		})
	})

	Describe("Compile", func() {
		apply := func(ctx SpecContext, key calculation.Key, path string, value any) error {
			return node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Apply(ctx, key, calculation.NewSetAction(
					calculation.SetPayload{Path: []string{path}, Value: value},
				))
			})
		}

		It("Should fail on an expression that does not compile", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{
				Output:     out.Key(),
				Expression: "return missing * 2",
			}
			Expect(create(ctx, &c)).To(MatchError(ContainSubstring("missing")))
			Expect(retrieve(ctx, out.Key())).To(BeEmpty())
		})

		It("Should fail on a primary that the expression does not read", func(
			ctx SpecContext,
		) {
			read, other := createVirtual(ctx, "read"), createVirtual(ctx, "other")
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{
				Output:     out.Key(),
				Expression: "return read * 2",
				Primary:    new(other.Key()),
			}
			Expect(create(ctx, &c)).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("not a channel that the expression reads")),
			))
			c.Primary = new(read.Key())
			Expect(create(ctx, &c)).To(Succeed())
		})

		It("Should give the output the data type of the expression", func(
			ctx SpecContext,
		) {
			createChannel(ctx, channel.Channel{
				Name:     "wide",
				DataType: telem.Int64T,
				Virtual:  true,
			})
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{
				Output:     out.Key(),
				Expression: "return wide * 2",
			}
			Expect(create(ctx, &c)).To(Succeed())
			var stored channel.Channel
			Expect(channels.NewRetrieve().
				Where(channel.MatchKeys(out.Key())).
				Entry(&stored).
				Exec(ctx, node.DB)).To(Succeed())
			Expect(stored.DataType).To(Equal(telem.Int64T))
		})

		It("Should fail on an edit after which calculations read each other", func(
			ctx SpecContext,
		) {
			createVirtual(ctx, "base")
			first := calculation.Calculation{
				Output:     createVirtual(ctx, "first").Key(),
				Expression: "return base * 2",
			}
			Expect(create(ctx, &first)).To(Succeed())
			second := calculation.Calculation{
				Output:     createVirtual(ctx, "second").Key(),
				Expression: "return first + 1",
			}
			Expect(create(ctx, &second)).To(Succeed())
			Expect(
				apply(ctx, first.Key, "expression", "return second * 2"),
			).To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("first -> second -> first")),
			))
			Expect(
				retrieve(ctx, first.Output)[0].Expression,
			).To(Equal("return base * 2"))
		})
	})

	Describe("ApplyChannel", func() {
		applyChannel := func(
			ctx SpecContext,
			key channel.Key,
			acts ...channel.Action,
		) ([]channel.Action, error) {
			var computed []channel.Action
			err := node.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
				computed, err = svc.NewWriter(tx).ApplyChannel(ctx, key, acts...)
				return err
			})
			return computed, err
		}
		set := func(path string, value any) channel.Action {
			return channel.NewSetAction(
				channel.SetPayload{Path: []string{path}, Value: value},
			)
		}

		It("Should edit the expression and operations of the calculation", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(applyChannel(
				ctx,
				out.Key(),
				set("expression", "return 3"),
				set("operations", []channel.Operation{
					{Type: channel.OperationTypeMax, Duration: telem.Second},
				}),
			)).Error().To(Succeed())
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(calculation.Calculation{
				Key:        c.Key,
				Output:     out.Key(),
				Expression: "return 3",
				Operations: []calculation.Operation{{Variant: calculation.MaxOperation{
					Window: calculation.Window{Duration: telem.Second},
				}}},
			}))
		})

		It("Should rename the channel and keep its calculation", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(applyChannel(
				ctx,
				out.Key(),
				channel.NewRenameAction(channel.RenamePayload{Name: "tripled"}),
			)).Error().To(Succeed())
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(c))
		})

		It("Should return a set of the data type that a new expression changes", func(
			ctx SpecContext,
		) {
			createChannel(ctx, channel.Channel{
				Name:     "wide",
				DataType: telem.Int64T,
				Virtual:  true,
			})
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2.0"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(applyChannel(ctx, out.Key(), set("expression", "return wide * 2"))).
				To(ConsistOf(set("data_type", telem.Int64T)))
		})

		It(
			"Should refuse an edit of the expression of a channel with no calculation",
			func(
				ctx SpecContext,
			) {
				out := createVirtual(ctx, "plain")
				Expect(applyChannel(ctx, out.Key(), set("expression", "return 3"))).
					Error().To(MatchError(validate.ErrValidation))
			},
		)
	})

	Describe("Delete", func() {
		It("Should keep the output of a deleted calculation", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, c.Key)
			})).To(Succeed())
			Expect(retrieve(ctx, out.Key())).To(BeEmpty())
			Expect(channels.NewRetrieve().
				Where(channel.MatchKeys(out.Key())).
				Exists(ctx, node.DB)).To(BeTrue())
		})

		It("Should delete the calculation of a deleted output", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return channels.NewWriter(tx).Delete(ctx, out.Key(), false)
			})).To(Succeed())
			Expect(svc.NewRetrieve().
				Where(calculation.MatchKeys(c.Key)).
				Exists(ctx, node.DB)).To(BeFalse())
		})
	})
})
