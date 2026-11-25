// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/calculation"
	"github.com/synnaxlabs/synnax/pkg/service/calculation/compiler"
	calctestutil "github.com/synnaxlabs/synnax/pkg/service/calculation/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/telem/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe(
	"Bare paths in a calculation (breaker)",
	Ordered,
	ContinueOnFailure,
	func() {
		var (
			pdb     *gorp.DB
			psvc    *channel.Service
			creator calctestutil.Creator
			groups  *group.Service
			base    channel.Channel
		)
		const dir = "calc_dir"
		BeforeAll(func(ctx SpecContext) {
			ShouldNotLeakGoroutines()
			node := testutil.NewNode(ctx)
			pdb = node.DB
			groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: pdb}))
			MustOpen(label.OpenService(ctx, label.ServiceConfig{DB: pdb}))
			statusSvc := MustOpen(
				status.OpenService(ctx, status.ServiceConfig{DB: pdb}),
			)
			psvc = MustOpen(channel.OpenService(ctx, channel.ServiceConfig{
				Channel: node.Channel,
				TS:      node.Storage.TS,
				DB:      pdb,
				HostKey: node.Identity.HostKey,
				Status:  statusSvc,
			}))
			creator = calctestutil.Creator{
				DB:      pdb,
				Channel: psvc,
				Calculation: MustOpen(calculation.OpenService(
					ctx,
					calculation.ServiceConfig{DB: pdb, Channel: psvc},
				)),
			}
			g := group.Group{Name: dir, Scope: resource.TypeChannel}
			Expect(pdb.WithTx(ctx, func(tx gorp.Tx) error {
				return groups.NewWriter(tx).Create(ctx, &g)
			})).To(Succeed())
			base = channel.Channel{
				Name:     "path_base",
				DataType: telem.Int64T,
				Virtual:  true,
				Group:    &g.Key,
			}
			Expect(pdb.WithTx(ctx, func(tx gorp.Tx) error {
				return psvc.NewWriter(tx).Create(ctx, &base)
			})).To(Succeed())
		})

		DescribeTable("C4 computes each sample of a calculation that reads a bare path",
			func(ctx SpecContext, name, ref string) {
				out := MustSucceed(creator.Create(
					ctx,
					calctestutil.Calculation{
						Name:       name,
						DataType:   telem.Int64T,
						Expression: "return " + ref + " * 2",
					},
				))
				mod := MustSucceed(compiler.Compile(ctx, pdb, compiler.Config{
					ChannelService: psvc,
					Channel:        out,
					Calculation:    MustSucceed(creator.CalculationOf(ctx, out.Key())),
				}))
				c := MustOpen(calculator.Open(ctx, calculator.Config{Module: mod}))
				fr := frame.NewUnary(base.Key(), telem.NewSeriesV[int64](10, 20, 30))
				of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
				Expect(changed).To(BeTrue())
				Expect(of.Get(out.Key()).Series[0]).
					To(MatchSeriesDataV[int64](20, 40, 60))
			},
			Entry("control: a bare name", "by_name", "path_base"),
			Entry("a bare path", "by_path", dir+".path_base"),
		)

		It("C5 compiles a calculation that reads a path that a module member shares",
			func(ctx SpecContext) {
				g := group.Group{Name: "math", Scope: resource.TypeChannel}
				Expect(pdb.WithTx(ctx, func(tx gorp.Tx) error {
					return groups.NewWriter(tx).Create(ctx, &g)
				})).To(Succeed())
				inf := channel.Channel{
					Name:     "inf",
					DataType: telem.Float64T,
					Virtual:  true,
					Group:    &g.Key,
				}
				Expect(pdb.WithTx(ctx, func(tx gorp.Tx) error {
					return psvc.NewWriter(tx).Create(ctx, &inf)
				})).To(Succeed())
				out := MustSucceed(creator.Create(
					ctx,
					calctestutil.Calculation{
						Name:       "by_math_path",
						DataType:   telem.Float64T,
						Expression: "return math.inf * 2.0",
					},
				))
				MustSucceed(compiler.Compile(ctx, pdb, compiler.Config{
					ChannelService: psvc,
					Channel:        out,
					Calculation:    MustSucceed(creator.CalculationOf(ctx, out.Key())),
				}))
			})
	},
)
