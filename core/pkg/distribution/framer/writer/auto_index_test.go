// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Writer AutoIndex", func() {
	var host testutil.Node
	BeforeEach(func(ctx SpecContext) { host = testutil.NewNode(ctx) })

	createIndex := func(ctx SpecContext, name string) channel.Channel {
		GinkgoHelper()
		return MustSucceed(host.CreateChannels(ctx, []channel.Channel{
			{Name: name, IsIndex: true, DataType: telem.TimestampT},
		}))[0]
	}

	// createData creates a float64 channel on idx.
	createData := func(
		ctx SpecContext,
		name string,
		idx channel.Channel,
	) channel.Channel {
		GinkgoHelper()
		return MustSucceed(host.CreateChannels(ctx, []channel.Channel{
			{Name: name, DataType: telem.Float64T, LocalIndex: idx.LocalKey},
		}))[0]
	}

	// openWriter opens a sync writer, so that Write reports a refusal.
	openWriter := func(ctx SpecContext, cfg writer.Config) (*writer.Writer, error) {
		cfg.Sync = new(true)
		return host.Framer.OpenWriter(ctx, cfg)
	}

	// readStamps returns the stamps of the first series of idx in tr.
	readStamps := func(
		ctx SpecContext,
		tr telem.TimeRange,
		idx channel.Channel,
	) []telem.TimeStamp {
		GinkgoHelper()
		fr := MustSucceed(host.Storage.TS.Read(ctx, tr, uint32(idx.Key())))
		return fr.SeriesAt(0).Unmarshal[telem.TimeStamp]()
	}

	Describe("Data-only writers", func() {
		It(
			"Should stamp the missing index with monotonic timestamps",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auto_idx_1")
				data := createData(ctx, "auto_data_1", idx)
				before := telem.Now()
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      channel.Keys{data.Key()},
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float64](1, 2, 3),
				))).To(BeTrue())
				MustCommit(w)
				after := telem.Now()

				ts := readStamps(ctx, telem.TimeRangeMax, idx)
				Expect(ts).To(HaveLen(3))
				Expect(ts[0]).To(BeNumerically(">=", before))
				Expect(ts[2]).To(BeNumerically("<=", after+telem.TimeStamp(3)))
				Expect(ts[1]).To(Equal(ts[0] + 1))
				Expect(ts[2]).To(Equal(ts[0] + 2))
			},
		)

		It(
			"Should remain strictly monotonic across multiple writes",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auto_idx_2")
				data := createData(ctx, "auto_data_2", idx)
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      channel.Keys{data.Key()},
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float64](1, 2, 3),
				))).To(BeTrue())
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float64](4, 5, 6),
				))).To(BeTrue())
				MustCommit(w)

				ts := readStamps(ctx, telem.TimeRangeMax, idx)
				Expect(ts).To(HaveLen(6))
				for i := 1; i < len(ts); i++ {
					Expect(ts[i]).To(BeNumerically(">", ts[i-1]))
				}
			},
		)
	})

	Describe("Mixed user-provided and auto-generated timestamps", func() {
		It(
			"Should pass through caller-supplied index series untouched",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "mix_idx_1")
				data := createData(ctx, "mix_data_1", idx)
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      channel.Keys{data.Key()},
					AutoIndex: new(true),
					Start:     100 * telem.SecondTS,
				}))
				Expect(w.Write(frame.NewMulti(
					channel.Keys{idx.Key(), data.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(100, 101, 102),
						telem.NewSeriesV[float64](1, 2, 3),
					},
				))).To(BeTrue())
				MustCommit(w)

				ts := readStamps(
					ctx,
					(100 * telem.SecondTS).Range(103*telem.SecondTS),
					idx,
				)
				Expect(ts).To(Equal([]telem.TimeStamp{
					100 * telem.SecondTS,
					101 * telem.SecondTS,
					102 * telem.SecondTS,
				}))
			},
		)

		It(
			"Should reject an auto-stamp that would regress the index past a caller-provided future timestamp",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "mix_idx_2")
				data := createData(ctx, "mix_data_2", idx)
				future := telem.Now() + telem.HourTS
				w := MustSucceed(openWriter(ctx, writer.Config{
					Keys:      channel.Keys{data.Key()},
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					channel.Keys{idx.Key(), data.Key()},
					[]telem.Series{
						telem.NewSeriesV(future),
						telem.NewSeriesV[float64](1),
					},
				))).To(BeTrue())
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float64](2),
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			},
		)

		It(
			"Does not advance the auto-stamp clock when the caller writes explicit timestamps",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "mix_idx_3")
				data := createData(ctx, "mix_data_3", idx)
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      channel.Keys{data.Key()},
					AutoIndex: new(true),
				}))
				explicit := telem.Now() + 10*telem.MillisecondTS
				Expect(w.Write(frame.NewMulti(
					channel.Keys{idx.Key(), data.Key()},
					[]telem.Series{
						telem.NewSeriesV(explicit),
						telem.NewSeriesV[float64](1),
					},
				))).To(BeTrue())
				Eventually(telem.Now).Should(BeNumerically(">", explicit))
				afterExplicit := telem.Now()
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float64](2),
				))).To(BeTrue())
				MustCommit(w)

				ts := readStamps(ctx, telem.TimeRangeMax, idx)
				Expect(ts).To(HaveLen(2))
				Expect(ts[0]).To(Equal(explicit))
				Expect(ts[1]).To(BeNumerically(">=", afterExplicit))
			},
		)
	})

	Describe("Virtual channels", func() {
		It(
			"Should auto-stamp the indexed channel while leaving a virtual channel in the same frame untouched",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "virt_idx")
				data := createData(ctx, "virt_data", idx)
				virtual := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
					{Name: "virt_chan", DataType: telem.Int64T, Virtual: true},
				}))[0]
				keys := channel.Keys{data.Key(), virtual.Key()}
				before := telem.Now()
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      keys,
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[float64](1, 2, 3),
					telem.NewSeriesV[int64](10, 20, 30),
				}))).To(BeTrue())
				MustCommit(w)

				ts := readStamps(ctx, telem.TimeRangeMax, idx)
				Expect(ts).To(HaveLen(3))
				Expect(ts[0]).To(BeNumerically(">=", before))
			},
		)
	})

	Describe("Multiple data channels per index", func() {
		It(
			"Should stamp once for an index shared across data channels",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "shared_idx")
				keys := channel.Keys{
					createData(ctx, "shared_data_1", idx).Key(),
					createData(ctx, "shared_data_2", idx).Key(),
				}
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      keys,
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[float64](1, 2, 3),
					telem.NewSeriesV[float64](4, 5, 6),
				}))).To(BeTrue())
				MustCommit(w)

				Expect(readStamps(ctx, telem.TimeRangeMax, idx)).To(HaveLen(3))
			},
		)
	})

	Describe("Multiple indexes in one writer", func() {
		It(
			"Should size each implicit index to its own referencing data channel",
			func(ctx SpecContext) {
				idx1 := createIndex(ctx, "multi_idx_a")
				data1 := createData(ctx, "multi_data_a", idx1)
				idx2 := createIndex(ctx, "multi_idx_b")
				data2 := createData(ctx, "multi_data_b", idx2)
				keys := channel.Keys{data1.Key(), data2.Key()}
				before := telem.Now()
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      keys,
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[float64](1, 2, 3),
					telem.NewSeriesV[float64](10, 20, 30, 40),
				}))).To(BeTrue())
				MustCommit(w)

				ts1 := readStamps(ctx, telem.TimeRangeMax, idx1)
				Expect(ts1).To(HaveLen(3))
				Expect(ts1[0]).To(BeNumerically(">=", before))
				Expect(ts1[1]).To(Equal(ts1[0] + 1))
				Expect(ts1[2]).To(Equal(ts1[0] + 2))

				Expect(readStamps(ctx, telem.TimeRangeMax, idx2)).To(HaveLen(4))
			},
		)

		It(
			"Should generate co-aligned timestamps across distinct indexes in the same write",
			func(ctx SpecContext) {
				idx1 := createIndex(ctx, "coalign_idx_a")
				data1 := createData(ctx, "coalign_data_a", idx1)
				idx2 := createIndex(ctx, "coalign_idx_b")
				data2 := createData(ctx, "coalign_data_b", idx2)
				keys := channel.Keys{data1.Key(), data2.Key()}
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      keys,
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesV[float64](1, 2, 3),
					telem.NewSeriesV[float64](4, 5, 6),
				}))).To(BeTrue())
				MustCommit(w)

				Expect(readStamps(ctx, telem.TimeRangeMax, idx1)).To(Equal(
					readStamps(ctx, telem.TimeRangeMax, idx2),
				))
			},
		)

		It(
			"Should auto-stamp one index while passing through a user-provided value for another",
			func(ctx SpecContext) {
				idx1 := createIndex(ctx, "mixed_multi_idx_a")
				data1 := createData(ctx, "mixed_multi_data_a", idx1)
				idx2 := createIndex(ctx, "mixed_multi_idx_b")
				data2 := createData(ctx, "mixed_multi_data_b", idx2)
				keys := channel.Keys{idx1.Key(), data1.Key(), data2.Key()}
				before := telem.Now()
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      keys,
					Start:     100 * telem.SecondTS,
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(100, 101),
					telem.NewSeriesV[float64](1, 2),
					telem.NewSeriesV[float64](3, 4),
				}))).To(BeTrue())
				MustCommit(w)

				Expect(readStamps(
					ctx,
					(100 * telem.SecondTS).Range(102*telem.SecondTS),
					idx1,
				)).To(Equal([]telem.TimeStamp{
					100 * telem.SecondTS,
					101 * telem.SecondTS,
				}))

				ts2 := readStamps(ctx, telem.TimeRangeMax, idx2)
				Expect(ts2).To(HaveLen(2))
				Expect(ts2[0]).To(BeNumerically(">=", before))
				Expect(ts2[1]).To(Equal(ts2[0] + 1))
			},
		)
	})

	Describe("Start", func() {
		It(
			"Stamps from the time the writer opens when Start is zero",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "default_start_idx")
				data := createData(ctx, "default_start_data", idx)
				before := telem.Now()
				w := MustOpen(openWriter(ctx, writer.Config{
					Keys:      channel.Keys{data.Key()},
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float64](1, 2, 3),
				))).To(BeTrue())
				MustCommit(w)

				ts := readStamps(ctx, telem.TimeRangeMax, idx)
				Expect(ts).To(HaveLen(3))
				Expect(ts[0]).To(BeNumerically(">=", before))
			},
		)

		It(
			"Accepts an explicit index series with timestamps before the open time",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "default_start_reject_idx")
				data := createData(ctx, "default_start_reject_data", idx)
				w := MustSucceed(openWriter(ctx, writer.Config{
					Keys:      channel.Keys{data.Key()},
					AutoIndex: new(true),
				}))
				Expect(w.Write(frame.NewMulti(
					channel.Keys{idx.Key(), data.Key()},
					[]telem.Series{
						telem.NewSeriesSecondsTSV(1),
						telem.NewSeriesV[float64](1),
					},
				))).To(BeTrue())
				MustCommit(w)
				Expect(w.Close()).To(Succeed())
				fr := MustSucceed(
					host.Storage.TS.Read(ctx, telem.TimeRangeMax, uint32(idx.Key())),
				)
				Expect(fr.SeriesAt(0)).To(telem.MatchSeriesData(
					telem.NewSeriesSecondsTSV(1),
				))
			},
		)

		It("Preserves an explicitly provided Start", func(ctx SpecContext) {
			idx := createIndex(ctx, "default_start_preserve_idx")
			data := createData(ctx, "default_start_preserve_data", idx)
			w := MustSucceed(openWriter(ctx, writer.Config{
				Keys:      channel.Keys{data.Key()},
				AutoIndex: new(true),
				Start:     50 * telem.SecondTS,
			}))
			Expect(w.Write(frame.NewMulti(
				channel.Keys{idx.Key(), data.Key()},
				[]telem.Series{
					telem.NewSeriesSecondsTSV(100, 101),
					telem.NewSeriesV[float64](1, 2),
				},
			))).To(BeTrue())
			MustCommit(w)
			Expect(w.Close()).To(Succeed())

			Expect(readStamps(
				ctx,
				(50 * telem.SecondTS).Range(102*telem.SecondTS),
				idx,
			)).To(Equal([]telem.TimeStamp{
				100 * telem.SecondTS,
				101 * telem.SecondTS,
			}))
		})
	})

	Describe("Disabled", func() {
		It(
			"Should reject a frame missing the index when AutoIndex is false",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "off_idx")
				data := createData(ctx, "off_data", idx)
				w := MustSucceed(openWriter(ctx, writer.Config{
					Keys:  channel.Keys{idx.Key(), data.Key()},
					Start: 1 * telem.SecondTS,
				}))
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float64](1, 2),
				))).Error().To(MatchError(validate.ErrValidation))
				Expect(w.Close()).To(MatchError(validate.ErrValidation))
			},
		)
	})

	// writeIndex writes ts to idx through w and returns whether w had control.
	writeIndex := func(w *writer.Writer, idx channel.Channel, ts telem.TimeStamp) bool {
		GinkgoHelper()
		return MustSucceed(w.Write(frame.NewUnary(
			idx.Key(),
			telem.NewSeriesSecondsTSV(ts),
		)))
	}

	Describe("SetAuthority propagation", func() {
		It(
			"Raises an implicit index when its referencing data channel is raised",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auth_idx_1")
				data := createData(ctx, "auth_data_1", idx)
				wA := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{data.Key()},
					Authorities:    []control.Authority{50},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "wA"},
				}))
				wB := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{idx.Key()},
					Authorities:    []control.Authority{100},
					ControlSubject: control.Subject{Key: "wB"},
				}))
				Expect(writeIndex(wB, idx, 10)).To(BeTrue())
				Expect(wA.SetAuthority(writer.Config{
					Keys:        channel.Keys{data.Key()},
					Authorities: []control.Authority{200},
				})).To(Succeed())
				Expect(writeIndex(wB, idx, 11)).To(BeFalse())
			},
		)

		It(
			"Tracks the max across data channels sharing an implicit index",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auth_idx_2")
				data1 := createData(ctx, "auth_data_2a", idx)
				data2 := createData(ctx, "auth_data_2b", idx)
				wA := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{data1.Key(), data2.Key()},
					Authorities:    []control.Authority{50, 50},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "wA_max"},
				}))
				wB := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{idx.Key()},
					Authorities:    []control.Authority{150},
					ControlSubject: control.Subject{Key: "wB_max"},
				}))
				Expect(writeIndex(wB, idx, 10)).To(BeTrue())
				Expect(wA.SetAuthority(writer.Config{
					Keys:        channel.Keys{data1.Key()},
					Authorities: []control.Authority{100},
				})).To(Succeed())
				Expect(writeIndex(wB, idx, 11)).To(BeTrue())
				Expect(wA.SetAuthority(writer.Config{
					Keys:        channel.Keys{data2.Key()},
					Authorities: []control.Authority{200},
				})).To(Succeed())
				Expect(writeIndex(wB, idx, 12)).To(BeFalse())
			},
		)

		It(
			"Broadcasts a single authority across multiple data channels and propagates to the implicit index",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auth_bcast_idx")
				keys := channel.Keys{
					createData(ctx, "auth_bcast_data1", idx).Key(),
					createData(ctx, "auth_bcast_data2", idx).Key(),
				}
				wA := MustOpen(openWriter(ctx, writer.Config{
					Keys:           keys,
					Authorities:    []control.Authority{50, 50},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "wA_bcast"},
				}))
				wB := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{idx.Key()},
					Authorities:    []control.Authority{100},
					ControlSubject: control.Subject{Key: "wB_bcast"},
				}))
				Expect(writeIndex(wB, idx, 10)).To(BeTrue())
				Expect(wA.SetAuthority(writer.Config{
					Keys:        keys,
					Authorities: []control.Authority{200},
				})).To(Succeed())
				Expect(writeIndex(wB, idx, 11)).To(BeFalse())
			},
		)

		It(
			"Lowers the implicit index when both referencing data channels are lowered",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auth_lower_idx")
				keys := channel.Keys{
					createData(ctx, "auth_lower_data1", idx).Key(),
					createData(ctx, "auth_lower_data2", idx).Key(),
				}
				wA := MustOpen(openWriter(ctx, writer.Config{
					Keys:           keys,
					Authorities:    []control.Authority{200, 200},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "wA_lower"},
				}))
				wB := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{idx.Key()},
					Authorities:    []control.Authority{150},
					ControlSubject: control.Subject{Key: "wB_lower"},
				}))
				Expect(writeIndex(wB, idx, 10)).To(BeFalse())
				Expect(wA.SetAuthority(writer.Config{
					Keys:        keys,
					Authorities: []control.Authority{50, 50},
				})).To(Succeed())
				Expect(writeIndex(wB, idx, 11)).To(BeTrue())
			},
		)

		It(
			"Propagates from a broadcast SetAuthority through subsequent per-channel calls",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auth_chain_idx")
				data1 := createData(ctx, "auth_chain_data1", idx)
				data2 := createData(ctx, "auth_chain_data2", idx)
				wA := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{data1.Key(), data2.Key()},
					Authorities:    []control.Authority{50},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "wA_chain"},
				}))
				wB := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{idx.Key()},
					Authorities:    []control.Authority{150},
					ControlSubject: control.Subject{Key: "wB_chain"},
				}))
				Expect(writeIndex(wB, idx, 10)).To(BeTrue())
				Expect(wA.SetAuthority(writer.Config{
					Authorities: []control.Authority{100},
				})).To(Succeed())
				Expect(wA.SetAuthority(writer.Config{
					Keys:        channel.Keys{data1.Key()},
					Authorities: []control.Authority{200},
				})).To(Succeed())
				Expect(writeIndex(wB, idx, 11)).To(BeFalse())
			},
		)

		It(
			"Leaves the index alone when explicitly named in SetAuthority",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "auth_idx_3")
				data := createData(ctx, "auth_data_3", idx)
				wA := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{data.Key()},
					Authorities:    []control.Authority{50},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "wA_explicit"},
				}))
				wB := MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{idx.Key()},
					Authorities:    []control.Authority{100},
					ControlSubject: control.Subject{Key: "wB_explicit"},
				}))
				Expect(writeIndex(wB, idx, 10)).To(BeTrue())
				Expect(wA.SetAuthority(writer.Config{
					Keys:        channel.Keys{data.Key(), idx.Key()},
					Authorities: []control.Authority{250, 30},
				})).To(Succeed())
				Expect(writeIndex(wB, idx, 11)).To(BeTrue())
			},
		)
	})

	// openIndexWriter opens a writer to idx alone, at auth.
	openIndexWriter := func(
		ctx SpecContext,
		idx channel.Channel,
		auth control.Authority,
		subject string,
	) *writer.Writer {
		GinkgoHelper()
		return MustOpen(openWriter(ctx, writer.Config{
			Keys:           channel.Keys{idx.Key()},
			Authorities:    []control.Authority{auth},
			ControlSubject: control.Subject{Key: subject},
		}))
	}

	Describe("Open-time authority inheritance", func() {
		It(
			"Inherits the broadcast authority on an implicit index",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "open_idx_1")
				data := createData(ctx, "open_data_1", idx)
				MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{data.Key()},
					Authorities:    []control.Authority{150},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "open_wA"},
				}))
				wLow := openIndexWriter(ctx, idx, 100, "open_wLow")
				Expect(writeIndex(wLow, idx, 10)).To(BeFalse())
				wHigh := openIndexWriter(ctx, idx, 200, "open_wHigh")
				Expect(writeIndex(wHigh, idx, 11)).To(BeTrue())
			},
		)

		It(
			"Takes the max across data channels sharing an index",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "open_idx_2")
				data1 := createData(ctx, "open_data_2a", idx)
				data2 := createData(ctx, "open_data_2b", idx)
				MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{data1.Key(), data2.Key()},
					Authorities:    []control.Authority{50, 200},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "open_max_wA"},
				}))
				wBetween := openIndexWriter(ctx, idx, 100, "open_max_wBetween")
				Expect(writeIndex(wBetween, idx, 10)).To(BeFalse())
				wAbove := openIndexWriter(ctx, idx, 220, "open_max_wAbove")
				Expect(writeIndex(wAbove, idx, 11)).To(BeTrue())
			},
		)

		It(
			"Resolves authority per-index when distinct indexes are implicit",
			func(ctx SpecContext) {
				idx1 := createIndex(ctx, "open_perindex_idx_a")
				data1 := createData(ctx, "open_perindex_data_a", idx1)
				idx2 := createIndex(ctx, "open_perindex_idx_b")
				data2 := createData(ctx, "open_perindex_data_b", idx2)
				MustOpen(openWriter(ctx, writer.Config{
					Keys:           channel.Keys{data1.Key(), data2.Key()},
					Authorities:    []control.Authority{80, 180},
					AutoIndex:      new(true),
					Start:          1 * telem.SecondTS,
					ControlSubject: control.Subject{Key: "open_perindex_wA"},
				}))
				wIdx1 := openIndexWriter(ctx, idx1, 100, "open_perindex_wIdx1")
				Expect(writeIndex(wIdx1, idx1, 10)).To(BeTrue())
				wIdx2 := openIndexWriter(ctx, idx2, 100, "open_perindex_wIdx2")
				Expect(writeIndex(wIdx2, idx2, 10)).To(BeFalse())
			},
		)
	})

	Describe("Time Range Stamping", func() {
		It(
			"Should stamp auto-indexed frames with the generated timestamps",
			func(ctx SpecContext) {
				idx := createIndex(ctx, "Hopper")
				data := MustSucceed(host.CreateChannels(ctx, []channel.Channel{
					{
						Name:       "Lovelace",
						DataType:   telem.Float32T,
						LocalIndex: idx.LocalKey,
					},
				}))[0]
				keys := channel.Keys{idx.Key(), data.Key()}
				w := MustSucceed(openWriter(ctx, writer.Config{
					Keys:      keys,
					Start:     telem.Now(),
					AutoIndex: new(true),
				}))
				streamer := MustSucceed(host.Framer.NewStreamer(framer.StreamerConfig{
					Keys:        keys,
					SendOpenAck: new(true),
				}))
				_, out := confluence.Attach(streamer, 2)
				sCtx, cancel := signal.WithCancel(ctx)
				streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
				closer := signal.NewHardShutdown(sCtx, cancel)
				var res framer.StreamerResponse
				Eventually(out.Outlet()).Should(Receive(&res))
				Expect(w.Write(frame.NewUnary(
					data.Key(),
					telem.NewSeriesV[float32](1, 2, 3),
				))).To(BeTrue())
				Eventually(out.Outlet()).Should(Receive(&res))
				idxSeries := res.Frame.Get(idx.Key()).Series[0]
				expected := telem.TimeRange{
					Start: idxSeries.ValueAt[telem.TimeStamp](0),
					End:   idxSeries.ValueAt[telem.TimeStamp](-1) + 1,
				}
				Expect(expected.Start).ToNot(Equal(telem.TimeStamp(0)))
				Expect(idxSeries.TimeRange).To(Equal(expected))
				Expect(
					res.Frame.Get(data.Key()).Series[0].TimeRange,
				).To(Equal(expected))
				Expect(closer.Close()).To(Succeed())
				Expect(w.Close()).To(Succeed())
			},
		)
	})
})
