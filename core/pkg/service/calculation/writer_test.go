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
	"context"
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	dnode "github.com/synnaxlabs/synnax/pkg/distribution/node"
	disttestutil "github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	acttestutil "github.com/synnaxlabs/synnax/pkg/service/actions/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// expectDiagnostics expects err to hold the diagnostics of an expression that reads the
// missing channel, at the path of the expression.
func expectDiagnostics(err error) {
	GinkgoHelper()
	Expect(err).To(MatchError(ContainSubstring("missing")))
	_, ok := errors.AsType[*diagnostics.Diagnostics](err)
	Expect(ok).To(BeTrue(), "%v holds diagnostics", err)
	pathErr := MustBeOk(errors.AsType[validate.PathError](err))
	Expect(pathErr.Path).To(Equal([]string{"expression"}))
}

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
			AnnounceLinked: func(
				ctx context.Context,
				tx gorp.Tx,
				w channel.Writer,
				keys ...channel.Key,
			) error {
				return svc.AnnounceOutputs(ctx, tx, w, keys...)
			},
		}))
		DeferCleanup(func() { Expect(channels.Close()).To(Succeed()) })
		svc = MustSucceed(calculation.OpenService(ctx, calculation.ServiceConfig{
			DB:      node.DB,
			Channel: channels,
			Framer:  node.Framer,
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
	createdOf := func(
		rec *acttestutil.Recorder[channel.Key, channel.Action],
	) map[channel.Key]channel.Created {
		created := make(map[channel.Key]channel.Created)
		for _, batch := range rec.Snapshot() {
			for _, a := range batch.Actions {
				if a.Create != nil {
					created[batch.Key] = a.Create.Channel
				}
			}
		}
		return created
	}
	// announcedOf returns, by channel, each expression that the batches of rec set, in
	// order.
	announcedOf := func(
		rec *acttestutil.Recorder[channel.Key, channel.Action],
	) map[channel.Key][]string {
		GinkgoHelper()
		announced := make(map[channel.Key][]string)
		for _, batch := range rec.Snapshot() {
			for _, a := range batch.Actions {
				if a.Set == nil || !slices.Equal(a.Set.Path, []string{"expression"}) {
					continue
				}
				expression, ok := a.Set.Value.(string)
				Expect(ok).To(BeTrue())
				announced[batch.Key] = append(announced[batch.Key], expression)
			}
		}
		return announced
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

		It(
			"Should store the channels that the expression reads",
			func(ctx SpecContext) {
				a, b := createVirtual(ctx, "a"), createVirtual(ctx, "b")
				out := createVirtual(ctx, "sum")
				c := calculation.Calculation{
					Output:     out.Key(),
					Expression: "return b + a + b",
					Reads:      []channel.Key{out.Key()},
				}
				Expect(create(ctx, &c)).To(Succeed())
				want := []channel.Key{a.Key(), b.Key()}
				slices.Sort(want)
				Expect(c.Reads).To(Equal(want))
				Expect(retrieve(ctx, out.Key())[0].Reads).To(Equal(want))
			},
		)

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
		It("Should put no edit on the channel stream of the output", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			rec := &acttestutil.Recorder[channel.Key, channel.Action]{}
			DeferCleanup(channels.Actions().OnChange(rec.Record))
			window := calculation.Window{Duration: telem.Second}
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).WithDispatchKey("calculation-api").Apply(
					ctx,
					c.Key,
					calculation.NewSetAction(calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return 4",
					}),
					calculation.NewSetAction(calculation.SetPayload{
						Path: []string{"operations"},
						Value: []calculation.Operation{
							{Variant: calculation.AvgOperation{Window: window}},
						},
					}),
				)
				return err
			})).To(Succeed())
			Expect(retrieve(ctx, out.Key())[0].Expression).To(Equal("return 4"))
			Expect(rec.Snapshot()).To(BeEmpty())
		})

		It("Should set the expression of a calculation", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return 4",
					},
				))
				return err
			})).To(Succeed())
			Expect(retrieve(ctx, out.Key())[0].Expression).To(Equal("return 4"))
		})

		It("Should store the reads of a new expression", func(ctx SpecContext) {
			createVirtual(ctx, "first")
			second := createVirtual(ctx, "second")
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{
				Output:     out.Key(),
				Expression: "return first * 2",
			}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return second * 2",
					},
				))
				return err
			})).To(Succeed())
			Expect(retrieve(ctx, out.Key())[0].Reads).
				To(Equal([]channel.Key{second.Key()}))
		})

		It("Should keep a calculation whose new expression is empty", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(create(ctx, &c)).To(Succeed())
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{Path: []string{"expression"}, Value: ""},
				))
				return err
			})).To(MatchError(validate.ErrValidation))
			Expect(retrieve(ctx, out.Key())).To(ConsistOf(c))
		})
	})

	Describe("Announcements", func() {
		It(
			"Should announce the calculation of a stored output that a create links to "+
				"an index, and hold none in its create",
			func(
				ctx SpecContext,
			) {
				createVirtual(ctx, "wide")
				stored := createChannel(ctx, channel.Channel{
					Name:        "stored",
					DataType:    telem.Float32T,
					Virtual:     true,
					Leaseholder: dnode.KeyFree,
				})
				Expect(stored.LocalIndex).To(BeZero())
				c := calculation.Calculation{
					Output:     stored.Key(),
					Expression: "return wide * 2",
				}
				Expect(create(ctx, &c)).To(Succeed())
				rec := &acttestutil.Recorder[channel.Key, channel.Action]{}
				DeferCleanup(channels.Actions().OnChange(rec.Record))
				again := channel.Channel{
					Name:        "stored",
					DataType:    telem.Float32T,
					Virtual:     true,
					Leaseholder: dnode.KeyFree,
				}
				Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return channels.NewWriter(tx).Create(
						ctx,
						&again,
						channel.RetrieveIfNameExists(),
						channel.FreeIndex(),
					)
				})).To(Succeed())
				Expect(again.Key()).To(Equal(stored.Key()))
				Expect(again.LocalIndex).ToNot(BeZero())
				Expect(createdOf(rec)[stored.Key()].Expression).To(BeEmpty())
				Expect(announcedOf(rec)[stored.Key()]).
					To(Equal([]string{"return wide * 2"}))
			},
		)

		It(
			"Should announce the calculation of an existing output once, with the "+
				"dispatch key",
			func(ctx SpecContext) {
				createVirtual(ctx, "wide")
				output := createVirtual(ctx, "doubled")
				rec := &acttestutil.Recorder[channel.Key, channel.Action]{}
				DeferCleanup(channels.Actions().OnChange(rec.Record))
				c := calculation.Calculation{
					Output:     output.Key(),
					Expression: "return wide * 2",
				}
				Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
					return svc.NewWriter(tx).WithDispatchKey("dk").Create(ctx, &c)
				})).To(Succeed())
				Expect(announcedOf(rec)[output.Key()]).
					To(Equal([]string{"return wide * 2"}))
				Expect(rec.Snapshot()).To(ContainElement(SatisfyAll(
					HaveField("Key", output.Key()),
					HaveField("DispatchKey", "dk"),
				)))
			},
		)

		It("Should announce that a deleted calculation writes its output no more", func(
			ctx SpecContext,
		) {
			createVirtual(ctx, "wide")
			output := createVirtual(ctx, "doubled")
			c := calculation.Calculation{
				Output:     output.Key(),
				Expression: "return wide * 2",
			}
			Expect(create(ctx, &c)).To(Succeed())
			rec := &acttestutil.Recorder[channel.Key, channel.Action]{}
			DeferCleanup(channels.Actions().OnChange(rec.Record))
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, c.Key)
			})).To(Succeed())
			Expect(announcedOf(rec)).To(Equal(map[channel.Key][]string{
				output.Key(): {""},
			}))
		})

		It("Should announce nothing for an edit that keeps the output", func(
			ctx SpecContext,
		) {
			createVirtual(ctx, "wide")
			output := createVirtual(ctx, "doubled")
			c := calculation.Calculation{
				Output:     output.Key(),
				Expression: "return wide * 2",
			}
			Expect(create(ctx, &c)).To(Succeed())
			rec := &acttestutil.Recorder[channel.Key, channel.Action]{}
			DeferCleanup(channels.Actions().OnChange(rec.Record))
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := svc.NewWriter(tx).Apply(ctx, c.Key, calculation.NewSetAction(
					calculation.SetPayload{
						Path:  []string{"expression"},
						Value: "return wide * 3",
					},
				))
				return err
			})).To(Succeed())
			Expect(announcedOf(rec)).To(BeEmpty())
		})

		It("Should announce nothing for a channel that no calculation writes", func(
			ctx SpecContext,
		) {
			rec := &acttestutil.Recorder[channel.Key, channel.Action]{}
			DeferCleanup(channels.Actions().OnChange(rec.Record))
			plain := createVirtual(ctx, "plain")
			Expect(
				createdOf(rec)[plain.Key()],
			).To(Equal(channel.Created{Channel: plain}))
			Expect(announcedOf(rec)).To(BeEmpty())
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
				Expect(channels.NewRetrieve().
					Where(channel.MatchKeys(output.Index())).
					Entry(&index).
					Exec(ctx, node.DB)).To(Succeed())
				Expect(index.Name).To(Equal("doubled_time"))
				Expect(index.IsIndex).To(BeTrue())
				Expect(index.Virtual).To(BeTrue())
			},
		)

		It(
			"Should hold no calculation in the creates of the output and its index, and "+
				"announce it for the output",
			func(
				ctx SpecContext,
			) {
				createVirtual(ctx, "wide")
				rec := &acttestutil.Recorder[channel.Key, channel.Action]{}
				DeferCleanup(channels.Actions().OnChange(rec.Record))
				window := calculation.Window{Duration: telem.Second}
				c := calculation.Calculation{
					Expression: "return wide * 2",
					Operations: []calculation.Operation{
						{Variant: calculation.AvgOperation{Window: window}},
					},
				}
				Expect(createOutput(ctx, &c, "doubled")).To(Succeed())
				created := createdOf(rec)
				Expect(created).To(HaveLen(2))
				output := created[c.Output]
				Expect(output.Expression).To(BeEmpty())
				Expect(output.Operations).To(BeEmpty())
				Expect(
					announcedOf(rec)[c.Output],
				).To(Equal([]string{"return wide * 2"}))
				Expect(rec.Snapshot()).To(ContainElement(SatisfyAll(
					HaveField("Key", c.Output),
					HaveField(
						"Actions",
						ContainElement(HaveField("Set", HaveValue(Equal(
							channel.SetPayload{
								Path: []string{"operations"},
								Value: []channel.Operation{
									{
										Type:     channel.OperationTypeAvg,
										Duration: telem.Second,
									},
								},
							},
						)))),
					),
				)))
				index := created[output.Index()]
				Expect(index.IsIndex).To(BeTrue())
				Expect(index.Expression).To(BeEmpty())
				Expect(index.Operations).To(BeEmpty())
			},
		)

		It("Should fail an operation of no type before it makes an output", func(
			ctx SpecContext,
		) {
			createVirtual(ctx, "wide")
			c := calculation.Calculation{
				Expression: "return wide * 2",
				Operations: []calculation.Operation{{}},
			}
			Expect(
				createOutput(ctx, &c, "doubled"),
			).To(MatchError(validate.ErrValidation))
			Expect(channels.NewRetrieve().
				Where(channel.MatchNames("doubled")).
				Exists(ctx, node.DB)).To(BeFalse())
		})

		It("Should fail on a calculation that names its output", func(ctx SpecContext) {
			out := createVirtual(ctx, "doubled")
			c := calculation.Calculation{Output: out.Key(), Expression: "return 2"}
			Expect(
				createOutput(ctx, &c, "other"),
			).To(MatchError(validate.ErrValidation))
		})

		It("Should keep the diagnostics of a made output that does not compile", func(
			ctx SpecContext,
		) {
			c := calculation.Calculation{Expression: "return missing * 2"}
			expectDiagnostics(createOutput(ctx, &c, "doubled"))
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
				_, err := svc.NewWriter(tx).Apply(ctx, key, calculation.NewSetAction(
					calculation.SetPayload{Path: []string{path}, Value: value},
				))
				return err
			})
		}

		It("Should keep the diagnostics of a create that does not compile", func(
			ctx SpecContext,
		) {
			out := createVirtual(ctx, "doubled")
			expectDiagnostics(create(ctx, &calculation.Calculation{
				Output:     out.Key(),
				Expression: "return missing * 2",
			}))
		})

		It("Should keep the diagnostics of a new expression that cannot compile", func(
			ctx SpecContext,
		) {
			read, out := createVirtual(ctx, "read"), createVirtual(ctx, "doubled")
			c := calculation.Calculation{
				Output:     out.Key(),
				Expression: "return " + read.Name + " * 2",
			}
			Expect(create(ctx, &c)).To(Succeed())
			expectDiagnostics(apply(ctx, c.Key, "expression", "return missing * 2"))
		})

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
	Describe("Primary", func() {
		// stored creates a channel on its own index, and stores each stretch of stamps
		// on it through its own writer.
		stored := func(
			ctx SpecContext,
			name string,
			stretches ...[]telem.TimeStamp,
		) channel.Channel {
			GinkgoHelper()
			index := createChannel(ctx, channel.Channel{
				Name:     name + "_time",
				DataType: telem.TimestampT,
				IsIndex:  true,
			})
			data := createChannel(ctx, channel.Channel{
				Name:       name,
				DataType:   telem.Float32T,
				LocalIndex: index.LocalKey,
			})
			keys := channel.Keys{index.Key(), data.Key()}
			for _, stamps := range stretches {
				w := MustSucceed(node.Framer.OpenWriter(ctx, framer.WriterConfig{
					Start: stamps[0],
					Keys:  keys,
				}))
				Expect(MustSucceed(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeries(stamps),
					telem.NewSeries(make([]float32, len(stamps))),
				})))).To(BeTrue())
				_, authorized := MustSucceed2(w.Commit())
				Expect(authorized).To(BeTrue())
				Expect(w.Close()).To(Succeed())
			}
			return data
		}
		every := func(
			start telem.TimeStamp,
			interval telem.TimeSpan,
			n int,
		) []telem.TimeStamp {
			stamps := make([]telem.TimeStamp, n)
			for i := range stamps {
				stamps[i] = start.Add(telem.TimeSpan(i) * interval)
			}
			return stamps
		}
		createCalc := func(
			ctx SpecContext,
			c calculation.Calculation,
		) calculation.Calculation {
			GinkgoHelper()
			c.Output = createVirtual(ctx, "out").Key()
			Expect(create(ctx, &c)).To(Succeed())
			return retrieve(ctx, c.Output)[0]
		}
		apply := func(
			ctx SpecContext,
			c calculation.Calculation,
			expression string,
		) (calculation.Calculation, []calculation.Action) {
			GinkgoHelper()
			var computed []calculation.Action
			Expect(node.DB.WithTx(ctx, func(tx gorp.Tx) (err error) {
				computed, err = svc.NewWriter(tx).Apply(
					ctx,
					c.Key,
					calculation.NewSetAction(calculation.SetPayload{
						Path:  []string{"expression"},
						Value: expression,
					}),
				)
				return err
			})).To(Succeed())
			return retrieve(ctx, c.Output)[0], computed
		}

		It("Should pick the read with the shortest interval", func(ctx SpecContext) {
			stored(ctx, "slow", every(telem.SecondTS, 100*telem.Millisecond, 17))
			fast := stored(ctx, "fast", every(telem.SecondTS, 10*telem.Millisecond, 17))
			c := createCalc(
				ctx,
				calculation.Calculation{Expression: "return slow + fast"},
			)
			Expect(c.Primary).To(HaveValue(Equal(fast.Key())))
		})

		It("Should pick the earlier read on a tie", func(ctx SpecContext) {
			a := stored(ctx, "a", every(telem.SecondTS, telem.Second, 3))
			stored(ctx, "b", every(telem.SecondTS, telem.Second, 3))
			c := createCalc(ctx, calculation.Calculation{Expression: "return b + a"})
			Expect(c.Primary).To(HaveValue(Equal(a.Key())))
		})

		It("Should pick no virtual read or read with fewer than 2 samples", func(
			ctx SpecContext,
		) {
			createVirtual(ctx, "v")
			stored(ctx, "single", every(telem.SecondTS, telem.Millisecond, 1))
			slow := stored(ctx, "slow", every(telem.SecondTS, telem.Second, 2))
			c := createCalc(ctx, calculation.Calculation{
				Expression: "return v + single + slow",
			})
			Expect(c.Primary).To(HaveValue(Equal(slow.Key())))
		})

		It("Should pick the first read when no read is a candidate", func(
			ctx SpecContext,
		) {
			first := createVirtual(ctx, "first")
			createVirtual(ctx, "second")
			c := createCalc(ctx, calculation.Calculation{
				Expression: "return second + first",
			})
			Expect(c.Primary).To(HaveValue(Equal(first.Key())))
		})

		It("Should pick no primary for an expression that reads no channel", func(
			ctx SpecContext,
		) {
			c := createCalc(ctx, calculation.Calculation{Expression: "return 2"})
			Expect(c.Primary).To(BeNil())
		})

		It("Should time a read by its newest stretch alone", func(ctx SpecContext) {
			stored(
				ctx,
				"restarted",
				every(telem.SecondTS, telem.Millisecond, 17),
				every(2*telem.SecondTS, telem.Second, 2),
			)
			steady := stored(
				ctx,
				"steady",
				every(telem.SecondTS, 100*telem.Millisecond, 17),
			)
			c := createCalc(ctx, calculation.Calculation{
				Expression: "return restarted + steady",
			})
			Expect(c.Primary).To(HaveValue(Equal(steady.Key())))
		})

		It("Should keep a primary that the expression still reads", func(
			ctx SpecContext,
		) {
			stored(ctx, "fast", every(telem.SecondTS, telem.Millisecond, 17))
			slow := stored(ctx, "slow", every(telem.SecondTS, telem.Second, 17))
			c := createCalc(ctx, calculation.Calculation{
				Expression: "return fast + slow",
				Primary:    new(slow.Key()),
			})
			next, computed := apply(ctx, c, "return fast + slow + 1")
			Expect(next.Primary).To(HaveValue(Equal(slow.Key())))
			Expect(computed).To(BeEmpty())
		})

		It("Should pick again when the expression drops the primary", func(
			ctx SpecContext,
		) {
			fast := stored(ctx, "fast", every(telem.SecondTS, telem.Millisecond, 17))
			stored(ctx, "slow", every(telem.SecondTS, telem.Second, 17))
			mid := stored(ctx, "mid", every(telem.SecondTS, 10*telem.Millisecond, 17))
			c := createCalc(
				ctx,
				calculation.Calculation{Expression: "return fast + slow"},
			)
			Expect(c.Primary).To(HaveValue(Equal(fast.Key())))
			next, computed := apply(ctx, c, "return slow + mid")
			Expect(next.Primary).To(HaveValue(Equal(mid.Key())))
			Expect(computed).To(ContainElement(calculation.NewSetAction(
				calculation.SetPayload{Path: []string{"primary"}, Value: mid.Key()},
			)))
		})

		It("Should fail on a primary that the expression does not read", func(
			ctx SpecContext,
		) {
			other := createVirtual(ctx, "other")
			createVirtual(ctx, "read")
			c := calculation.Calculation{
				Output:     createVirtual(ctx, "out").Key(),
				Expression: "return read",
				Primary:    new(other.Key()),
			}
			Expect(create(ctx, &c)).To(And(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(fmt.Sprintf(
					"channel %s is not a channel that the expression reads",
					other.Key(),
				))),
			))
		})
	})
})
