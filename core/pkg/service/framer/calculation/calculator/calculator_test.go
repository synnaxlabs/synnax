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
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	channelanalyzer "github.com/synnaxlabs/synnax/pkg/service/channel/calculation/analyzer"
	"github.com/synnaxlabs/synnax/pkg/service/channel/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculator", Ordered, func() {
	var (
		arcSvc *arc.Service
		dist   mock.Node
	)
	BeforeAll(func(ctx SpecContext) {
		distB := DeferClose(mock.NewCluster())
		dist = DeferClose(distB.Provision(ctx))
		labelSvc := MustOpen(label.OpenService(ctx, label.ServiceConfig{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
			Search:   dist.Search,
		}))
		statusSvc := MustOpen(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Group:    dist.Group,
			Signals:  dist.Signals,
			Ontology: dist.Ontology,
			Label:    labelSvc,
			Search:   dist.Search,
		}))
		rackService := MustOpen(rack.OpenService(ctx, rack.ServiceConfig{
			DB:           dist.DB,
			Ontology:     dist.Ontology,
			Group:        dist.Group,
			HostProvider: mock.StaticHostKeyProvider(1),
			Status:       statusSvc,
			Search:       dist.Search,
		}))
		taskSvc := MustOpen(task.OpenService(ctx, task.ServiceConfig{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Rack:     rackService,
			Status:   statusSvc,
			Search:   dist.Search,
		}))
		arcSvc = MustOpen(arc.OpenService(ctx, arc.ServiceConfig{
			Channel:  dist.Channel,
			Ontology: dist.Ontology,
			DB:       dist.DB,
			Signals:  dist.Signals,
			Task:     taskSvc,
			Search:   dist.Search,
		}))
	})

	open := func(
		ctx context.Context,
		indexes, bases *[]channel.Channel,
		calc *channel.Channel,
	) *calculator.Calculator {
		if indexes != nil {
			Expect(dist.Channel.CreateMany(ctx, indexes)).To(Succeed())
		}
		if bases != nil {
			for i, channel := range *bases {
				if channel.Virtual {
					continue
				}
				toGet := i
				if len(*indexes) == 1 {
					toGet = 0
				}
				channel.LocalIndex = (*indexes)[toGet].LocalKey
				(*bases)[i] = channel
			}
			Expect(dist.Channel.CreateMany(ctx, bases)).To(Succeed())
		}
		Expect(dist.Channel.Create(ctx, calc)).To(Succeed())
		mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
			ChannelService: dist.Channel,
			Channel:        *calc,
			SymbolResolver: arcSvc.NewSymbolResolver(nil),
		}))
		return MustSucceed(calculator.Open(ctx, calculator.Config{Module: mod}))
	}

	Describe("Alignment", func() {
		Specify("Single alignment propagation", func(ctx SpecContext) {
			base := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", base[0].Name),
			}
			c := open(ctx, nil, &base, &calc)
			d := telem.NewSeriesV[int64](10, 20, 30)
			d.Alignment = telem.NewAlignment(100, 50)
			fr := frame.NewUnary(base[0].Key(), d)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](20, 40, 60))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(100, 50)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Multiple alignments accumulation", func(ctx SpecContext) {
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, nil, &bases, &calc)
			d1 := telem.NewSeriesV[int64](1, 2)
			d1.Alignment = telem.NewAlignment(10, 5)
			d2 := telem.NewSeriesV[int64](3, 4)
			d2.Alignment = telem.NewAlignment(20, 3)
			fr := frame.NewMulti(
				[]channel.Key{bases[0].Key(), bases[1].Key()},
				[]telem.Series{d1, d2},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](4, 6))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(30, 8)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Alignment persistence across calls", func(ctx SpecContext) {
			base := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + 5", base[0].Name),
			}
			c := open(ctx, nil, &base, &calc)
			d1 := telem.NewSeriesV[int64](1)
			d1.Alignment = telem.NewAlignment(15, 2)
			fr1 := frame.NewUnary(base[0].Key(), d1)
			of, changed := MustSucceed2(c.Next(ctx, fr1, frame.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](6))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(15, 2)))
			d2 := telem.NewSeriesV[int64](2)
			d2.Alignment = telem.NewAlignment(25, 7)
			fr2 := frame.NewUnary(base[0].Key(), d2)
			of, changed = MustSucceed2(c.Next(ctx, fr2, frame.Frame{}))
			Expect(changed).To(BeTrue())
			od = of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](7))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(25, 7)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Mixed alignment sources", func(ctx SpecContext) {
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + %s + %s", bases[0].Name, bases[1].Name, bases[2].Name),
			}
			c := open(ctx, nil, &bases, &calc)
			d1 := telem.NewSeriesV[int64](1)
			d1.Alignment = telem.NewAlignment(10, 3)
			d2 := telem.NewSeriesV[int64](2)
			d3 := telem.NewSeriesV[int64](3)
			d3.Alignment = telem.NewAlignment(5, 1)
			fr := frame.NewMulti(
				[]channel.Key{bases[0].Key(), bases[1].Key(), bases[2].Key()},
				[]telem.Series{d1, d2, d3},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](6))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(15, 4)))
			Expect(c.Close()).To(Succeed())
		})
	})

	Describe("Channel Configurations", func() {
		Specify("Two virtual channels", func(ctx SpecContext) {
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float32T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float32T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s - %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, nil, &bases, &calc)
			fr := frame.NewMulti(
				[]channel.Key{bases[0].Key(), bases[1].Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](10.5, 20.5, 30.5),
					telem.NewSeriesV[float32](0.5, 1.5, 2.5),
				},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](10.0, 19.0, 28.0))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Three virtual channels", func(ctx SpecContext) {
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int32T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int32T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int32T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * %s + %s", bases[0].Name, bases[1].Name, bases[2].Name),
			}
			c := open(ctx, nil, &bases, &calc)
			fr := frame.NewMulti(
				[]channel.Key{bases[0].Key(), bases[1].Key(), bases[2].Key()},
				[]telem.Series{
					telem.NewSeriesV[int32](2, 3),
					telem.NewSeriesV[int32](4, 5),
					telem.NewSeriesV[int32](1, 2),
				},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int32](9, 17))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Single persisted channel", func(ctx SpecContext) {
			indexes := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Float64T,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s / 2", bases[0].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(1, 2, 3)
			idxData.Alignment = telem.NewAlignment(10, 5)
			valData := telem.NewSeriesV(100.0, 200.0, 300.0)
			valData.Alignment = telem.NewAlignment(10, 5)
			fr := frame.NewMulti(
				[]channel.Key{indexes[0].Key(), bases[0].Key()},
				[]telem.Series{idxData, valData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV(50.0, 100.0, 150.0))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(10, 5)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Two persisted channels shared index", func(ctx SpecContext) {
			indexes := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(10, 20, 30)
			idxData.Alignment = telem.NewAlignment(5, 2)
			tempData := telem.NewSeriesV[int64](15, 25, 35)
			tempData.Alignment = telem.NewAlignment(5, 2)
			pressureData := telem.NewSeriesV[int64](5, 10, 15)
			pressureData.Alignment = telem.NewAlignment(5, 2)
			fr := frame.NewMulti(
				[]channel.Key{indexes[0].Key(), bases[0].Key(), bases[1].Key()},
				[]telem.Series{idxData, tempData, pressureData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](20, 35, 50))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				10*telem.SecondTS, 20*telem.SecondTS, 30*telem.SecondTS,
			))
			// Alignment is summed: (5,2) + (5,2) = (10,4)
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(10, 4)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Two persisted channels unique indexes", func(ctx SpecContext) {
			indexes := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float32T,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float32T,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)
			idx1Data := telem.NewSeriesSecondsTSV(1, 2)
			idx1Data.Alignment = telem.NewAlignment(3, 1)
			voltageData := telem.NewSeriesV[float32](2.0, 4.0)
			voltageData.Alignment = telem.NewAlignment(3, 1)
			idx2Data := telem.NewSeriesSecondsTSV(10, 20)
			idx2Data.Alignment = telem.NewAlignment(7, 3)
			currentData := telem.NewSeriesV[float32](3.0, 5.0)
			currentData.Alignment = telem.NewAlignment(7, 3)
			fr := frame.NewMulti(
				[]channel.Key{indexes[0].Key(), bases[0].Key(), indexes[1].Key(), bases[1].Key()},
				[]telem.Series{idx1Data, voltageData, idx2Data, currentData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](6.0, 20.0))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(10, 4)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Mixed virtual and persisted", func(ctx SpecContext) {
			indexes := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s - %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(5, 10)
			idxData.Alignment = telem.NewAlignment(8, 4)
			persistedData := telem.NewSeriesV[int64](100, 200)
			persistedData.Alignment = telem.NewAlignment(8, 4)
			virtualData := telem.NewSeriesV[int64](30, 50)
			virtualData.Alignment = telem.NewAlignment(12, 2)
			fr := frame.NewMulti(
				[]channel.Key{indexes[0].Key(), bases[0].Key(), bases[1].Key()},
				[]telem.Series{idxData, persistedData, virtualData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](70, 150))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(20, 6)))
			Expect(c.Close()).To(Succeed())
		})
	})

	Describe("Data Types", func() {
		Specify("Float32", func(ctx SpecContext) {
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float32T,
					Virtual:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float32T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s / %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, nil, &bases, &calc)
			fr := frame.NewMulti(
				[]channel.Key{bases[0].Key(), bases[1].Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](10.0, 20.0, 30.0),
					telem.NewSeriesV[float32](2.0, 4.0, 5.0),
				},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](5.0, 5.0, 6.0))
			Expect(c.Close()).To(Succeed())
		})
	})

	Describe("Accumulation", func() {
		Specify("Index after data", func(ctx SpecContext) {
			indexes := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Int64T,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 3", bases[0].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)
			dataOnly := telem.NewSeriesV[int64](10, 20, 30)
			dataOnly.Alignment = telem.NewAlignment(5, 2)
			fr1 := frame.NewUnary(bases[0].Key(), dataOnly)
			of, changed := MustSucceed2(c.Next(ctx, fr1, frame.Frame{}))
			Expect(changed).To(BeFalse())
			idxData := telem.NewSeriesSecondsTSV(1, 2, 3)
			idxData.Alignment = telem.NewAlignment(5, 2)
			fr2 := frame.NewUnary(indexes[0].Key(), idxData)
			of, changed = MustSucceed2(c.Next(ctx, fr2, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](30, 60, 90))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(5, 2)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Data after index", func(ctx SpecContext) {
			indexes := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Int64T,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", bases[0].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(1, 2, 3)
			idxData.Alignment = telem.NewAlignment(3, 1)
			fr1 := frame.NewUnary(indexes[0].Key(), idxData)
			of, changed := MustSucceed2(c.Next(ctx, fr1, frame.Frame{}))
			Expect(changed).To(BeFalse())
			dataOnly := telem.NewSeriesV[int64](15, 25, 35)
			dataOnly.Alignment = telem.NewAlignment(3, 1)
			fr2 := frame.NewUnary(bases[0].Key(), dataOnly)
			of, changed = MustSucceed2(c.Next(ctx, fr2, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](30, 50, 70))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(3, 1)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Sequential channel arrivals", func(ctx SpecContext) {
			indexes := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float64T,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float64T,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)
			idx := telem.NewSeriesSecondsTSV(1, 2, 3)
			idx.Alignment = telem.NewAlignment(5, 1)
			fr1 := frame.NewUnary(indexes[0].Key(), idx)
			_, changed := MustSucceed2(c.Next(ctx, fr1, frame.Frame{}))
			Expect(changed).To(BeFalse())
			ch1Data := telem.NewSeriesV(10.0, 20.0, 30.0)
			ch1Data.Alignment = telem.NewAlignment(5, 1)
			fr2 := frame.NewUnary(bases[0].Key(), ch1Data)
			of := frame.Frame{}
			_, changed = MustSucceed2(c.Next(ctx, fr2, of))
			Expect(changed).To(BeFalse())
			ch2Data := telem.NewSeriesV(1.0, 2.0, 3.0)
			ch2Data.Alignment = telem.NewAlignment(5, 1)
			fr3 := frame.NewUnary(bases[1].Key(), ch2Data)
			of = frame.Frame{}
			of, changed = MustSucceed2(c.Next(ctx, fr3, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV(11.0, 22.0, 33.0))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			// Alignment is summed: (5,1) + (5,1) = (10,2)
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(10, 2)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Different indexes from different writers", func(ctx SpecContext) {
			indexes := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}
			bases := []channel.Channel{
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float64T,
				},
				{
					Name:     channel.NewRandomName(),
					DataType: telem.Float64T,
				},
			}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + %s", bases[0].Name, bases[1].Name),
			}
			c := open(ctx, &indexes, &bases, &calc)

			// Writer 1 sends idx1 + ch1 — not enough inputs to compute yet
			idx1 := telem.NewSeriesSecondsTSV(1, 2, 3)
			idx1.Alignment = telem.NewAlignment(5, 0)
			ch1Data := telem.NewSeriesV(10.0, 20.0, 30.0)
			ch1Data.Alignment = telem.NewAlignment(5, 0)
			fr1 := frame.NewMulti(
				[]channel.Key{indexes[0].Key(), bases[0].Key()},
				[]telem.Series{idx1, ch1Data},
			)
			_, changed := MustSucceed2(c.Next(ctx, fr1, frame.Frame{}))
			Expect(changed).To(BeFalse())

			// Writer 2 sends idx2 + ch2 — now both inputs available, should compute
			idx2 := telem.NewSeriesSecondsTSV(10, 20, 30)
			idx2.Alignment = telem.NewAlignment(8, 0)
			ch2Data := telem.NewSeriesV(1.0, 2.0, 3.0)
			ch2Data.Alignment = telem.NewAlignment(8, 0)
			fr2 := frame.NewMulti(
				[]channel.Key{indexes[1].Key(), bases[1].Key()},
				[]telem.Series{idx2, ch2Data},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr2, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV(11.0, 22.0, 33.0))

			Expect(c.Close()).To(Succeed())
		})
	})

	It("Operations", func(ctx SpecContext) {
		idx := []channel.Channel{{
			Name:     channel.NewRandomName(),
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}}
		base := []channel.Channel{{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
		}}
		calc := channel.Channel{
			Name:       channel.NewRandomName(),
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: fmt.Sprintf("return %s", base[0].Name),
			Operations: []channel.Operation{
				{
					Type:     "avg",
					Duration: 6 * telem.Second,
				},
			},
		}
		c := open(ctx, &idx, &base, &calc)
		d := telem.NewSeriesV[int64](10, 20, 30)
		i := telem.NewSeriesSecondsTSV(1, 2, 3)
		d.Alignment = telem.NewAlignment(1, 0)
		i.Alignment = d.Alignment
		fr := frame.NewMulti(
			[]channel.Key{idx[0].Key(), base[0].Key()},
			[]telem.Series{i, d},
		)
		o, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(o.Len()).To(BeEquivalentTo(1))
		Expect(o.Get(calc.Index()).Series[0]).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(3)))
		Expect(o.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](20))

		d = telem.NewSeriesV[int64](40, 50, 60)
		i = telem.NewSeriesSecondsTSV(4, 5, 6)
		d.Alignment = telem.NewAlignment(1, 3)
		i.Alignment = d.Alignment
		fr = frame.NewMulti(
			[]channel.Key{idx[0].Key(), base[0].Key()},
			[]telem.Series{i, d},
		)
		o, changed = MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(o.Len()).To(BeEquivalentTo(1))
		Expect(o.Get(calc.Index()).Series[0]).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(6)))
		Expect(o.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](35))
	})

	It("Should compute derivative operation with type promotion", func(ctx SpecContext) {
		idx := []channel.Channel{{
			Name:     channel.NewRandomName(),
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}}
		base := []channel.Channel{{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
		}}
		calc := channel.Channel{
			Name:       channel.NewRandomName(),
			DataType:   telem.Float64T,
			Virtual:    true,
			Expression: fmt.Sprintf("return %s", base[0].Name),
			Operations: []channel.Operation{
				{Type: "derivative"},
			},
		}
		c := open(ctx, &idx, &base, &calc)
		d := telem.NewSeriesV[int64](10, 20, 40)
		i := telem.NewSeriesSecondsTSV(1, 2, 4)
		d.Alignment = telem.NewAlignment(1, 0)
		i.Alignment = d.Alignment
		fr := frame.NewMulti(
			[]channel.Key{idx[0].Key(), base[0].Key()},
			[]telem.Series{i, d},
		)
		o, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(o.Len()).To(BeEquivalentTo(3))
		result := telem.UnmarshalSeries[float64](o.Get(calc.Key()).Series[0])
		Expect(result).To(HaveLen(3))
		Expect(result[0]).To(BeNumerically("~", 0.0, 0.01))
		Expect(result[1]).To(BeNumerically("~", 10.0, 0.01))
		Expect(result[2]).To(BeNumerically("~", 10.0, 0.01))

		d = telem.NewSeriesV[int64](70)
		i = telem.NewSeriesSecondsTSV(7)
		d.Alignment = telem.NewAlignment(1, 3)
		i.Alignment = d.Alignment
		fr = frame.NewMulti(
			[]channel.Key{idx[0].Key(), base[0].Key()},
			[]telem.Series{i, d},
		)
		o, changed = MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		result = telem.UnmarshalSeries[float64](o.Get(calc.Key()).Series[0])
		Expect(result).To(HaveLen(1))
		Expect(result[0]).To(BeNumerically("~", 10.0, 0.01))

		Expect(c.Close()).To(Succeed())
	})

	It("Should correctly chain multiple operations", func(ctx SpecContext) {
		idx := []channel.Channel{{
			Name:     channel.NewRandomName(),
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}}
		base := []channel.Channel{{
			Name:     channel.NewRandomName(),
			DataType: telem.Int64T,
		}}
		calc := channel.Channel{
			Name:       channel.NewRandomName(),
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: fmt.Sprintf("return %s", base[0].Name),
			Operations: []channel.Operation{
				{
					Type:     "avg",
					Duration: 3 * telem.Second,
				},
				{
					Type:     "avg",
					Duration: 6 * telem.Second,
				},
			},
		}
		c := open(ctx, &idx, &base, &calc)

		d := telem.NewSeriesV[int64](10, 20, 30)
		i := telem.NewSeriesSecondsTSV(1, 2, 3)
		d.Alignment = telem.NewAlignment(1, 0)
		i.Alignment = d.Alignment
		fr := frame.NewMulti(
			[]channel.Key{idx[0].Key(), base[0].Key()},
			[]telem.Series{i, d},
		)
		o, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
		Expect(changed).To(BeTrue())
		// The first avg (3s window) should produce output, and that output
		// should flow into the second avg (6s window). If the operations
		// are not correctly chained, the second avg never receives data
		// and the calculator produces no output.
		Expect(o.Get(calc.Key()).Series).ToNot(BeEmpty())

		Expect(c.Close()).To(Succeed())
	})

	Describe("Group", func() {

		It("Should aggregate ReadFrom keys from all calculators", func(ctx SpecContext) {
			idx := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.TimeStampT, IsIndex: true}}
			b1 := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.Int64T}}
			b2 := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}}
			c1 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + 1", b1[0].Name),
			}
			c2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", b2[0].Name),
			}
			calc1 := open(ctx, &idx, &b1, &c1)
			calc2 := open(ctx, nil, &b2, &c2)
			g := calculator.Group{calc1, calc2}
			keys := g.ReadFrom()
			Expect(keys).To(HaveLen(3))
			Expect(keys).To(ContainElements(idx[0].Key(), b1[0].Key(), b2[0].Key()))
		})

		It("Should execute all calculators and aggregate results", func(ctx SpecContext) {
			idx := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.TimeStampT, IsIndex: true}}
			b1 := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.Int64T}}
			b2 := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}}
			c1 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + 1", b1[0].Name),
			}
			c2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", b2[0].Name),
			}
			calc1 := open(ctx, &idx, &b1, &c1)
			calc2 := open(ctx, nil, &b2, &c2)
			g := calculator.Group{calc1, calc2}
			d1 := telem.NewSeriesV[int64](10, 20)
			d2 := telem.NewSeriesV[int64](5, 10)
			i := telem.NewSeriesSecondsTSV(1, 2)
			fr := frame.NewMulti([]channel.Key{idx[0].Key(), b1[0].Key(), b2[0].Key()}, []telem.Series{i, d1, d2})
			output, changed, statuses := g.Next(ctx, fr)
			Expect(statuses).To(BeEmpty())
			Expect(changed).To(BeTrue())
			Expect(output.Len()).To(BeNumerically(">", 0))
			Expect(output.Get(c1.Key()).Series).ToNot(BeEmpty())
			Expect(output.Get(c2.Key()).Series).ToNot(BeEmpty())
			Expect(output.Get(c1.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](11, 21))
			Expect(output.Get(c2.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](10, 20))
		})

		It("Should close all calculators", func(ctx SpecContext) {
			idx := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.TimeStampT, IsIndex: true}}
			b1 := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.Int64T}}
			b2 := []channel.Channel{{Name: channel.NewRandomName(), DataType: telem.Int64T, Virtual: true}}
			c1 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s", b1[0].Name),
			}
			c2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s", b2[0].Name),
			}
			calc1 := open(ctx, &idx, &b1, &c1)
			calc2 := open(ctx, &idx, &b2, &c2)
			g := calculator.Group{calc1, calc2}
			Expect(g.Close()).To(Succeed())
		})

		It("Should execute nested calculators", func(ctx SpecContext) {
			b1 := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			c1 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s + 1", b1[0].Name),
			}
			c2 := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: fmt.Sprintf("return %s * 2", c1.Name),
			}
			calc1 := open(ctx, nil, &b1, &c1)
			calc2 := open(ctx, nil, nil, &c2)
			g := calculator.Group{calc1, calc2}
			for i := range 10 {
				d1 := telem.NewSeriesV[int64](10, 20)
				d1.Alignment = telem.NewAlignment(0, uint32(i*2))
				fr := frame.NewMulti([]channel.Key{b1[0].Key()}, []telem.Series{d1})
				output, changed, statuses := g.Next(ctx, fr)
				Expect(statuses).To(BeEmpty())
				Expect(changed).To(BeTrue())
				Expect(output.Len()).To(BeNumerically(">", 0))
				Expect(output.Get(c1.Key()).Series).ToNot(BeEmpty())
				Expect(output.Get(c2.Key()).Series).ToNot(BeEmpty())
				Expect(output.Get(c1.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](11, 21))
				Expect(output.Get(c2.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](22, 42))
			}
		})
	})

	Describe("Leading Literal Type Coercion", func() {
		openWithInferredType := func(
			ctx context.Context,
			bases *[]channel.Channel,
			calc *channel.Channel,
		) *calculator.Calculator {
			Expect(dist.Channel.CreateMany(ctx, bases)).To(Succeed())
			res := MustSucceed(channelanalyzer.New(arcSvc.NewSymbolResolver(nil)).
				Analyze(ctx, *calc))
			calc.DataType = res.ChanDataType
			Expect(dist.Channel.Create(ctx, calc)).To(Succeed())
			mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
				ChannelService: dist.Channel,
				Channel:        *calc,
				SymbolResolver: arcSvc.NewSymbolResolver(nil),
			}))
			return MustSucceed(calculator.Open(ctx, calculator.Config{Module: mod}))
		}

		Specify("Float literal * f32 channel should infer f32 and produce correct results", func(ctx SpecContext) {
			base := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				Virtual:    true,
				Expression: fmt.Sprintf("return 2.0 * %s", base[0].Name),
			}
			c := openWithInferredType(ctx, &base, &calc)
			Expect(calc.DataType).To(Equal(telem.Float32T))
			fr := frame.NewUnary(
				base[0].Key(),
				telem.NewSeriesV[float32](10.0, 20.0, 30.0),
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(
				telem.MatchSeriesDataV[float32](20.0, 40.0, 60.0),
			)
			Expect(c.Close()).To(Succeed())
		})

		Specify("Integer literal - f32 channel should infer f32 and produce correct results", func(ctx SpecContext) {
			base := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				Virtual:    true,
				Expression: fmt.Sprintf("return 1000 - %s", base[0].Name),
			}
			c := openWithInferredType(ctx, &base, &calc)
			Expect(calc.DataType).To(Equal(telem.Float32T))
			fr := frame.NewUnary(
				base[0].Key(),
				telem.NewSeriesV[float32](10.0, 20.0),
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(
				telem.MatchSeriesDataV[float32](990.0, 980.0),
			)
			Expect(c.Close()).To(Succeed())
		})

		Specify("Float literal / f32 channel should infer f32 and produce correct results", func(ctx SpecContext) {
			base := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				Virtual:    true,
				Expression: fmt.Sprintf("return 1000.0 / %s", base[0].Name),
			}
			c := openWithInferredType(ctx, &base, &calc)
			Expect(calc.DataType).To(Equal(telem.Float32T))
			fr := frame.NewUnary(
				base[0].Key(),
				telem.NewSeriesV[float32](10.0, 20.0),
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(
				telem.MatchSeriesDataV[float32](100.0, 50.0),
			)
			Expect(c.Close()).To(Succeed())
		})

		Specify("Stale f64 output type from old inference with f32 channel and leading literal", func(ctx SpecContext) {
			base := []channel.Channel{{
				Name:     channel.NewRandomName(),
				DataType: telem.Float32T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       channel.NewRandomName(),
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: fmt.Sprintf("return 2.0 * %s", base[0].Name),
			}
			Expect(dist.Channel.CreateMany(ctx, &base)).To(Succeed())
			Expect(dist.Channel.Create(ctx, &calc)).To(Succeed())
			mod := MustSucceed(compiler.Compile(ctx, compiler.Config{
				ChannelService: dist.Channel,
				Channel:        calc,
				SymbolResolver: arcSvc.NewSymbolResolver(nil),
			}))
			c := MustSucceed(calculator.Open(ctx, calculator.Config{Module: mod}))
			fr := frame.NewUnary(
				base[0].Key(),
				telem.NewSeriesV[float32](10.0, 20.0, 30.0),
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, frame.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(
				telem.MatchSeriesDataV[float32](20.0, 40.0, 60.0),
			)
			Expect(c.Close()).To(Succeed())
		})
	})
})
