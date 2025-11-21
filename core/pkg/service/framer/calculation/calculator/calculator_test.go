// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/calculator"
	"github.com/synnaxlabs/synnax/pkg/service/framer/calculation/compiler"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Calculator", Ordered, func() {
	var (
		arcSvc *arc.Service
		dist   mock.Node
	)
	BeforeAll(func() {
		distB := mock.NewCluster()
		dist = distB.Provision(ctx)
		labelSvc := MustSucceed(label.OpenService(ctx, label.Config{
			DB:       dist.DB,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		statusSvc := MustSucceed(status.OpenService(ctx, status.ServiceConfig{
			DB:       dist.DB,
			Label:    labelSvc,
			Ontology: dist.Ontology,
			Group:    dist.Group,
			Signals:  dist.Signals,
		}))
		arcSvc = MustSucceed(arc.OpenService(ctx, arc.ServiceConfig{
			Channel:  dist.Channel,
			Ontology: dist.Ontology,
			DB:       dist.DB,
			Framer:   dist.Framer,
			Status:   statusSvc,
			Signals:  dist.Signals,
		}))
	})

	AfterAll(func() {
		Expect(dist.Close()).To(Succeed())
	})

	open := func(
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
			SymbolResolver: arcSvc.SymbolResolver(),
		}))
		return MustSucceed(calculator.Open(ctx, calculator.Config{Module: mod}))
	}

	Describe("Alignment", func() {
		Specify("Single alignment propagation", func() {
			base := []channel.Channel{{
				Name:     "base",
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base * 2",
			}
			c := open(nil, &base, &calc)
			d := telem.NewSeriesV[int64](10, 20, 30)
			d.Alignment = telem.NewAlignment(100, 50)
			fr := core.UnaryFrame(base[0].Key(), d)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](20, 40, 60))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(100, 50)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Multiple alignments accumulation", func() {
			bases := []channel.Channel{
				{
					Name:     "base_1",
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     "base_2",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base_1 + base_2",
			}
			c := open(nil, &bases, &calc)
			d1 := telem.NewSeriesV[int64](1, 2)
			d1.Alignment = telem.NewAlignment(10, 5)
			d2 := telem.NewSeriesV[int64](3, 4)
			d2.Alignment = telem.NewAlignment(20, 3)
			fr := core.MultiFrame(
				[]channel.Key{bases[0].Key(), bases[1].Key()},
				[]telem.Series{d1, d2},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](4, 6))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(30, 8)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Alignment persistence across calls", func() {
			base := []channel.Channel{{
				Name:     "base",
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base + 5",
			}
			c := open(nil, &base, &calc)
			d1 := telem.NewSeriesV[int64](1)
			d1.Alignment = telem.NewAlignment(15, 2)
			fr1 := core.UnaryFrame(base[0].Key(), d1)
			of, changed := MustSucceed2(c.Next(ctx, fr1, core.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](6))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(15, 2)))
			d2 := telem.NewSeriesV[int64](2)
			d2.Alignment = telem.NewAlignment(25, 7)
			fr2 := core.UnaryFrame(base[0].Key(), d2)
			of, changed = MustSucceed2(c.Next(ctx, fr2, core.Frame{}))
			Expect(changed).To(BeTrue())
			od = of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](7))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(25, 7)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Mixed alignment sources", func() {
			bases := []channel.Channel{
				{
					Name:     "base_1",
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     "base_2",
					DataType: telem.Int64T,
					Virtual:  true,
				},
				{
					Name:     "base_3",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base_1 + base_2 + base_3",
			}
			c := open(nil, &bases, &calc)
			d1 := telem.NewSeriesV[int64](1)
			d1.Alignment = telem.NewAlignment(10, 3)
			d2 := telem.NewSeriesV[int64](2)
			d3 := telem.NewSeriesV[int64](3)
			d3.Alignment = telem.NewAlignment(5, 1)
			fr := core.MultiFrame(
				[]channel.Key{bases[0].Key(), bases[1].Key(), bases[2].Key()},
				[]telem.Series{d1, d2, d3},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			od := of.Get(calc.Key()).Series[0]
			Expect(od).To(telem.MatchSeriesDataV[int64](6))
			Expect(od.Alignment).To(Equal(telem.NewAlignment(15, 4)))
			Expect(c.Close()).To(Succeed())
		})
	})

	Describe("Channel Configurations", func() {
		Specify("Two virtual channels", func() {
			bases := []channel.Channel{
				{
					Name:     "sensor_a",
					DataType: telem.Float32T,
					Virtual:  true,
				},
				{
					Name:     "sensor_b",
					DataType: telem.Float32T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return sensor_a - sensor_b",
			}
			c := open(nil, &bases, &calc)
			fr := core.MultiFrame(
				[]channel.Key{bases[0].Key(), bases[1].Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](10.5, 20.5, 30.5),
					telem.NewSeriesV[float32](0.5, 1.5, 2.5),
				},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](10.0, 19.0, 28.0))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Three virtual channels", func() {
			bases := []channel.Channel{
				{
					Name:     "x",
					DataType: telem.Int32T,
					Virtual:  true,
				},
				{
					Name:     "y",
					DataType: telem.Int32T,
					Virtual:  true,
				},
				{
					Name:     "z",
					DataType: telem.Int32T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int32T,
				Virtual:    true,
				Expression: "return x * y + z",
			}
			c := open(nil, &bases, &calc)
			fr := core.MultiFrame(
				[]channel.Key{bases[0].Key(), bases[1].Key(), bases[2].Key()},
				[]telem.Series{
					telem.NewSeriesV[int32](2, 3),
					telem.NewSeriesV[int32](4, 5),
					telem.NewSeriesV[int32](1, 2),
				},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int32](9, 17))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Single persisted channel", func() {
			indexes := []channel.Channel{{
				Name:     "idx",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{{
				Name:     "data",
				DataType: telem.Float64T,
			}}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Float64T,
				Virtual:    true,
				Expression: "return data / 2",
			}
			c := open(&indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(1, 2, 3)
			idxData.Alignment = telem.NewAlignment(10, 5)
			valData := telem.NewSeriesV(100.0, 200.0, 300.0)
			valData.Alignment = telem.NewAlignment(10, 5)
			fr := core.MultiFrame(
				[]channel.Key{indexes[0].Key(), bases[0].Key()},
				[]telem.Series{idxData, valData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV(50.0, 100.0, 150.0))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(10, 5)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Two persisted channels shared index", func() {
			indexes := []channel.Channel{{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{
				{
					Name:     "temp",
					DataType: telem.Int64T,
				},
				{
					Name:     "pressure",
					DataType: telem.Int64T,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return temp + pressure",
			}
			c := open(&indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(10, 20, 30)
			idxData.Alignment = telem.NewAlignment(5, 2)
			tempData := telem.NewSeriesV[int64](15, 25, 35)
			tempData.Alignment = telem.NewAlignment(5, 2)
			pressureData := telem.NewSeriesV[int64](5, 10, 15)
			pressureData.Alignment = telem.NewAlignment(5, 2)
			fr := core.MultiFrame(
				[]channel.Key{indexes[0].Key(), bases[0].Key(), bases[1].Key()},
				[]telem.Series{idxData, tempData, pressureData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](20, 35, 50))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				10*telem.SecondTS, 20*telem.SecondTS, 30*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(5, 2)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Two persisted channels unique indexes", func() {
			indexes := []channel.Channel{
				{
					Name:     "time_1",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
				{
					Name:     "time_2",
					DataType: telem.TimeStampT,
					IsIndex:  true,
				},
			}
			bases := []channel.Channel{
				{
					Name:     "voltage",
					DataType: telem.Float32T,
				},
				{
					Name:     "current",
					DataType: telem.Float32T,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return voltage * current",
			}
			c := open(&indexes, &bases, &calc)
			idx1Data := telem.NewSeriesSecondsTSV(1, 2)
			idx1Data.Alignment = telem.NewAlignment(3, 1)
			voltageData := telem.NewSeriesV[float32](2.0, 4.0)
			voltageData.Alignment = telem.NewAlignment(3, 1)
			idx2Data := telem.NewSeriesSecondsTSV(10, 20)
			idx2Data.Alignment = telem.NewAlignment(7, 3)
			currentData := telem.NewSeriesV[float32](3.0, 5.0)
			currentData.Alignment = telem.NewAlignment(7, 3)
			fr := core.MultiFrame(
				[]channel.Key{indexes[0].Key(), bases[0].Key(), indexes[1].Key(), bases[1].Key()},
				[]telem.Series{idx1Data, voltageData, idx2Data, currentData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](6.0, 20.0))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(10, 4)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Mixed virtual and persisted", func() {
			indexes := []channel.Channel{{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{
				{
					Name:     "persisted_ch",
					DataType: telem.Int64T,
				},
				{
					Name:     "virtual_ch",
					DataType: telem.Int64T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return persisted_ch - virtual_ch",
			}
			c := open(&indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(5, 10)
			idxData.Alignment = telem.NewAlignment(8, 4)
			persistedData := telem.NewSeriesV[int64](100, 200)
			persistedData.Alignment = telem.NewAlignment(8, 4)
			virtualData := telem.NewSeriesV[int64](30, 50)
			virtualData.Alignment = telem.NewAlignment(12, 2)
			fr := core.MultiFrame(
				[]channel.Key{indexes[0].Key(), bases[0].Key(), bases[1].Key()},
				[]telem.Series{idxData, persistedData, virtualData},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](70, 150))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(20, 6)))
			Expect(c.Close()).To(Succeed())
		})
	})

	Describe("Data Types", func() {
		Specify("Float32", func() {
			bases := []channel.Channel{
				{
					Name:     "a",
					DataType: telem.Float32T,
					Virtual:  true,
				},
				{
					Name:     "b",
					DataType: telem.Float32T,
					Virtual:  true,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Float32T,
				Virtual:    true,
				Expression: "return a / b",
			}
			c := open(nil, &bases, &calc)
			fr := core.MultiFrame(
				[]channel.Key{bases[0].Key(), bases[1].Key()},
				[]telem.Series{
					telem.NewSeriesV[float32](10.0, 20.0, 30.0),
					telem.NewSeriesV[float32](2.0, 4.0, 5.0),
				},
			)
			of, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[float32](5.0, 5.0, 6.0))
			Expect(c.Close()).To(Succeed())
		})
	})

	Describe("Accumulation", func() {
		Specify("Index after data", func() {
			indexes := []channel.Channel{{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{{
				Name:     "sensor",
				DataType: telem.Int64T,
			}}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return sensor * 3",
			}
			c := open(&indexes, &bases, &calc)
			dataOnly := telem.NewSeriesV[int64](10, 20, 30)
			dataOnly.Alignment = telem.NewAlignment(5, 2)
			fr1 := core.UnaryFrame(bases[0].Key(), dataOnly)
			of, changed := MustSucceed2(c.Next(ctx, fr1, core.Frame{}))
			Expect(changed).To(BeFalse())
			idxData := telem.NewSeriesSecondsTSV(1, 2, 3)
			idxData.Alignment = telem.NewAlignment(5, 2)
			fr2 := core.UnaryFrame(indexes[0].Key(), idxData)
			of, changed = MustSucceed2(c.Next(ctx, fr2, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](30, 60, 90))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(5, 2)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Data after index", func() {
			indexes := []channel.Channel{{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{{
				Name:     "sensor",
				DataType: telem.Int64T,
			}}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return sensor * 2",
			}
			c := open(&indexes, &bases, &calc)
			idxData := telem.NewSeriesSecondsTSV(1, 2, 3)
			idxData.Alignment = telem.NewAlignment(3, 1)
			fr1 := core.UnaryFrame(indexes[0].Key(), idxData)
			of, changed := MustSucceed2(c.Next(ctx, fr1, core.Frame{}))
			Expect(changed).To(BeFalse())
			dataOnly := telem.NewSeriesV[int64](15, 25, 35)
			dataOnly.Alignment = telem.NewAlignment(3, 1)
			fr2 := core.UnaryFrame(bases[0].Key(), dataOnly)
			of, changed = MustSucceed2(c.Next(ctx, fr2, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](30, 50, 70))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(3, 1)))
			Expect(c.Close()).To(Succeed())
		})

		Specify("Interleaved", func() {
			indexes := []channel.Channel{{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{{
				Name:     "sensor",
				DataType: telem.Int64T,
			}}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return sensor + 10",
			}
			c := open(&indexes, &bases, &calc)

			// First write: Data Series 1. Should not Calculate
			data1 := telem.NewSeriesV[int64](5, 15)
			data1.Alignment = telem.NewAlignment(2, 1)
			fr1 := core.UnaryFrame(bases[0].Key(), data1)
			_, changed := MustSucceed2(c.Next(ctx, fr1, core.Frame{}))
			Expect(changed).To(BeFalse())

			// Second Write: Data Series 2. Should Not Calculate
			idx1 := telem.NewSeriesSecondsTSV(1, 2)
			idx1.Alignment = telem.NewAlignment(2, 1)
			fr2 := core.UnaryFrame(indexes[0].Key(), idx1)
			of := core.Frame{}
			of, changed = MustSucceed2(c.Next(ctx, fr2, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](15, 25))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(2, 1)))

			data2 := telem.NewSeriesV[int64](25)
			data2.Alignment = telem.NewAlignment(3, 2)
			fr3 := core.UnaryFrame(bases[0].Key(), data2)
			of = core.Frame{}
			_, changed = MustSucceed2(c.Next(ctx, fr3, of))
			Expect(changed).To(BeFalse())

			idx2 := telem.NewSeriesSecondsTSV(3)
			idx2.Alignment = telem.NewAlignment(3, 2)
			fr4 := core.UnaryFrame(indexes[0].Key(), idx2)
			of = core.Frame{}
			of, changed = MustSucceed2(c.Next(ctx, fr4, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](35))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(3 * telem.SecondTS))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(3, 2)))

			Expect(c.Close()).To(Succeed())
		})

		Specify("Sequential channel arrivals", func() {
			indexes := []channel.Channel{{
				Name:     "time",
				DataType: telem.TimeStampT,
				IsIndex:  true,
			}}
			bases := []channel.Channel{
				{
					Name:     "ch1",
					DataType: telem.Float64T,
				},
				{
					Name:     "ch2",
					DataType: telem.Float64T,
				},
			}
			calc := channel.Channel{
				Name:       "calc",
				DataType:   telem.Float64T,
				Virtual:    true,
				Expression: "return ch1 + ch2",
			}
			c := open(&indexes, &bases, &calc)
			idx := telem.NewSeriesSecondsTSV(1, 2, 3)
			idx.Alignment = telem.NewAlignment(5, 1)
			fr1 := core.UnaryFrame(indexes[0].Key(), idx)
			_, changed := MustSucceed2(c.Next(ctx, fr1, core.Frame{}))
			Expect(changed).To(BeFalse())
			ch1Data := telem.NewSeriesV(10.0, 20.0, 30.0)
			ch1Data.Alignment = telem.NewAlignment(5, 1)
			fr2 := core.UnaryFrame(bases[0].Key(), ch1Data)
			of := core.Frame{}
			_, changed = MustSucceed2(c.Next(ctx, fr2, of))
			Expect(changed).To(BeFalse())
			ch2Data := telem.NewSeriesV(1.0, 2.0, 3.0)
			ch2Data.Alignment = telem.NewAlignment(5, 1)
			fr3 := core.UnaryFrame(bases[1].Key(), ch2Data)
			of = core.Frame{}
			of, changed = MustSucceed2(c.Next(ctx, fr3, of))
			Expect(changed).To(BeTrue())
			Expect(of.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV(11.0, 22.0, 33.0))
			Expect(of.Get(calc.Index()).Series[0]).To(telem.MatchSeriesDataV(
				1*telem.SecondTS, 2*telem.SecondTS, 3*telem.SecondTS,
			))
			Expect(of.Get(calc.Index()).Series[0].Alignment).To(Equal(telem.NewAlignment(5, 1)))
			Expect(c.Close()).To(Succeed())
		})
	})

	It("Operations", func() {
		idx := []channel.Channel{{
			Name:     "time",
			DataType: telem.TimeStampT,
			IsIndex:  true,
		}}
		base := []channel.Channel{{
			Name:     "base",
			DataType: telem.Int64T,
		}}
		calc := channel.Channel{
			Name:       "calc",
			DataType:   telem.Int64T,
			Virtual:    true,
			Expression: "return base",
			Operations: []channel.Operation{
				{
					Type:     "avg",
					Duration: 6 * telem.Second,
				},
			},
		}
		c := open(&idx, &base, &calc)
		d := telem.NewSeriesV[int64](10, 20, 30)
		i := telem.NewSeriesSecondsTSV(1, 2, 3)
		d.Alignment = telem.NewAlignment(1, 0)
		i.Alignment = d.Alignment
		fr := core.MultiFrame(
			[]channel.Key{idx[0].Key(), base[0].Key()},
			[]telem.Series{i, d},
		)
		o, changed := MustSucceed2(c.Next(ctx, fr, core.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(o.Len()).To(BeEquivalentTo(1))
		Expect(o.Get(calc.Index()).Series[0]).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(3)))
		Expect(o.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](20))

		d = telem.NewSeriesV[int64](40, 50, 60)
		i = telem.NewSeriesSecondsTSV(4, 5, 6)
		d.Alignment = telem.NewAlignment(1, 3)
		i.Alignment = d.Alignment
		fr = core.MultiFrame(
			[]channel.Key{idx[0].Key(), base[0].Key()},
			[]telem.Series{i, d},
		)
		o, changed = MustSucceed2(c.Next(ctx, fr, core.Frame{}))
		Expect(changed).To(BeTrue())
		Expect(o.Len()).To(BeEquivalentTo(1))
		Expect(o.Get(calc.Index()).Series[0]).To(telem.MatchSeriesData(telem.NewSeriesSecondsTSV(6)))
		Expect(o.Get(calc.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](35))
	})

	Describe("Group", func() {

		It("Should aggregate ReadFrom keys from all calculators", func() {
			idx := []channel.Channel{{Name: "time", DataType: telem.TimeStampT, IsIndex: true}}
			b1 := []channel.Channel{{Name: "base1", DataType: telem.Int64T}}
			b2 := []channel.Channel{{Name: "base2", DataType: telem.Int64T, Virtual: true}}
			c1 := channel.Channel{
				Name:       "calc1",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base1 + 1",
			}
			c2 := channel.Channel{
				Name:       "calc2",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base2 * 2",
			}
			calc1 := open(&idx, &b1, &c1)
			calc2 := open(nil, &b2, &c2)
			g := calculator.Group{calc1, calc2}
			keys := g.ReadFrom()
			Expect(keys).To(HaveLen(3))
			Expect(keys).To(ContainElements(idx[0].Key(), b1[0].Key(), b2[0].Key()))
		})

		It("Should execute all calculators and aggregate results", func() {
			idx := []channel.Channel{{Name: "time", DataType: telem.TimeStampT, IsIndex: true}}
			b1 := []channel.Channel{{Name: "base1", DataType: telem.Int64T}}
			b2 := []channel.Channel{{Name: "base2", DataType: telem.Int64T, Virtual: true}}
			c1 := channel.Channel{
				Name:       "calc1",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base1 + 1",
			}
			c2 := channel.Channel{
				Name:       "calc2",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base2 * 2",
			}
			calc1 := open(&idx, &b1, &c1)
			calc2 := open(nil, &b2, &c2)
			g := calculator.Group{calc1, calc2}
			d1 := telem.NewSeriesV[int64](10, 20)
			d2 := telem.NewSeriesV[int64](5, 10)
			i := telem.NewSeriesSecondsTSV(1, 2)
			fr := core.MultiFrame([]channel.Key{idx[0].Key(), b1[0].Key(), b2[0].Key()}, []telem.Series{i, d1, d2})
			output, changed, statuses := g.Next(ctx, fr)
			Expect(statuses).To(BeEmpty())
			Expect(changed).To(BeTrue())
			Expect(output.Len()).To(BeNumerically(">", 0))
			Expect(output.Get(c1.Key()).Series).ToNot(BeEmpty())
			Expect(output.Get(c2.Key()).Series).ToNot(BeEmpty())
			Expect(output.Get(c1.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](11, 21))
			Expect(output.Get(c2.Key()).Series[0]).To(telem.MatchSeriesDataV[int64](10, 20))
		})

		It("Should close all calculators", func() {
			idx := []channel.Channel{{Name: "time", DataType: telem.TimeStampT, IsIndex: true}}
			b1 := []channel.Channel{{Name: "base1", DataType: telem.Int64T}}
			b2 := []channel.Channel{{Name: "base2", DataType: telem.Int64T, Virtual: true}}
			c1 := channel.Channel{
				Name:       "calc1",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base1",
			}
			c2 := channel.Channel{
				Name:       "calc2",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base2",
			}
			calc1 := open(&idx, &b1, &c1)
			calc2 := open(&idx, &b2, &c2)
			g := calculator.Group{calc1, calc2}
			Expect(g.Close()).To(Succeed())
		})

		It("Should execute nested calculators", func() {
			b1 := []channel.Channel{{
				Name:     "base_a_1",
				DataType: telem.Int64T,
				Virtual:  true,
			}}
			c1 := channel.Channel{
				Name:       "calc_a_1",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return base_a_1 + 1",
			}
			c2 := channel.Channel{
				Name:       "calc_a_2",
				DataType:   telem.Int64T,
				Virtual:    true,
				Expression: "return calc_a_1 * 2",
			}
			calc1 := open(nil, &b1, &c1)
			calc2 := open(nil, nil, &c2)
			g := calculator.Group{calc1, calc2}
			for i := 0; i < 10; i++ {
				d1 := telem.NewSeriesV[int64](10, 20)
				d1.Alignment = telem.NewAlignment(0, uint32(i*2))
				fr := core.MultiFrame([]channel.Key{b1[0].Key()}, []telem.Series{d1})
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
})
