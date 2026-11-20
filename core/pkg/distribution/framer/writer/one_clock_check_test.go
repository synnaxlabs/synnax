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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/control"
	. "github.com/synnaxlabs/x/io/fs/testutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("One clock check", func() {
	var (
		db        *testDB
		idx, data cesium.ChannelKey
		keys      []cesium.ChannelKey
	)
	BeforeEach(func(ctx SpecContext) {
		db = mustOpenDBOnFS(ctx, FileSystems["memFS"]())
		idx, data = GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key:      idx,
				Name:     "d17_idx",
				IsIndex:  true,
				DataType: telem.TimestampT,
			},
			cesium.Channel{
				Key:      data,
				Name:     "d17_data",
				Index:    idx,
				DataType: telem.Int64T,
			},
		)).To(Succeed())
		keys = []cesium.ChannelKey{idx, data}
	})
	fr := func(stamps []telem.TimeStamp, values ...int64) cesium.Frame {
		return telem.MultiFrame(keys, []telem.Series{
			telem.NewSeries(stamps),
			telem.NewSeries(values),
		})
	}
	secs := func(s ...int64) []telem.TimeStamp {
		out := make([]telem.TimeStamp, len(s))
		for i, v := range s {
			out[i] = telem.TimeStamp(v) * telem.SecondTS
		}
		return out
	}
	open := func(ctx SpecContext, cfg storageConfig) *storageWriter {
		GinkgoHelper()
		w := MustSucceed(db.OpenWriter(ctx, cfg))
		DeferCleanup(func() { _ = w.Close() })
		return w
	}
	openFramer := func(ctx SpecContext, cfg writer.Config) *writer.Writer {
		GinkgoHelper()
		w := MustSucceed(db.svc.Open(ctx, cfg))
		DeferCleanup(func() { _ = w.Close() })
		return w
	}

	It("Should return the end of a stream-only commit", func(ctx SpecContext) {
		w := openFramer(ctx, writer.Config{
			Keys:             channel.KeysFromUint32(keys),
			Start:            10 * telem.SecondTS,
			Mode:             writer.ModeStreamOnly,
			EnableAutoCommit: new(false),
			Sync:             new(true),
		})
		Expect(w.Write(frame.NewFromStorage(fr(secs(10, 11, 12, 13), 1, 2, 3, 4)))).
			To(BeTrue())
		<-db.writes.Outlet()
		end, authorized, err := w.Commit()
		Expect(err).ToNot(HaveOccurred())
		Expect(authorized).To(BeTrue())
		Expect(end).To(Equal(13*telem.SecondTS + 1))
	})

	It(
		"Should not give a stream-only write the alignment of a persisted live write",
		func(ctx SpecContext) {
			p := open(ctx, storageConfig{Channels: keys, Start: 10 * telem.SecondTS})
			persisted := MustSucceed(p.Write(ctx, fr(secs(10, 11, 12), 1, 2, 3)))
			MustSucceed(p.Commit(ctx))
			Expect(p.Close()).To(Succeed())
			s := open(ctx, storageConfig{
				Channels: keys,
				Start:    20 * telem.SecondTS,
				Mode:     writer.ModeStreamOnly,
			})
			streamed := MustSucceed(s.Write(ctx, fr(secs(20, 21), 4, 5)))
			a1 := persisted.Get(data).Series[0].Alignment
			a2 := streamed.Get(data).Series[0].Alignment
			Expect(a2.DomainIndex()).ToNot(
				Equal(a1.DomainIndex()),
				"persisted %v and streamed %v share an alignment domain",
				a1,
				a2,
			)
		},
	)

	It(
		"Should report a stream-only end of domain over an index that another writer "+
			"controls as not authorized",
		func(ctx SpecContext) {
			open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "holder"},
				Channels:       keys,
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{200},
			})
			s := openFramer(ctx, writer.Config{
				ControlSubject:   control.Subject{Key: "streamer"},
				Keys:             channel.KeysFromUint32(keys),
				Start:            10 * telem.SecondTS,
				Mode:             writer.ModeStreamOnly,
				Authorities:      []control.Authority{100},
				EnableAutoCommit: new(false),
				Sync:             new(true),
			})
			Expect(s.EndDomain(channel.Keys{channel.Key(idx)})).To(BeFalse())
		},
	)

	It(
		"Should stream nothing of a group whose index a stream-only writer does not "+
			"control",
		func(ctx SpecContext) {
			open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "holder"},
				Channels:       keys,
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{100},
			})
			s := open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "data_only"},
				Channels:       keys,
				Start:          10 * telem.SecondTS,
				Mode:           writer.ModeStreamOnly,
				Authorities:    []control.Authority{50, 200},
			})
			written, err := s.Write(ctx, fr(secs(10, 11), 1, 2))
			Expect(err).To(MatchError(control.ErrUnauthorized))
			Expect(written.Get(data).Len()).To(
				BeZero(),
				"streamed %v of a group whose index the writer lacks",
				written.Get(data).Series,
			)
		},
	)

	DescribeTable(
		"Should refuse an invalid frame of stored channels in stream-only mode",
		func(ctx SpecContext, bad func() cesium.Frame, msg string) {
			s := open(ctx, storageConfig{
				Channels: keys,
				Start:    10 * telem.SecondTS,
				Mode:     writer.ModeStreamOnly,
			})
			_, err := s.Write(ctx, bad())
			Expect(err).To(MatchError(validate.ErrValidation))
			Expect(err).To(MatchError(ContainSubstring(msg)))
		},
		Entry(
			"wrong data type",
			func() cesium.Frame {
				return telem.MultiFrame(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(10, 11),
					telem.NewSeriesV[float64](1, 2),
				})
			},
			"invalid data type, expected int64, got float64",
		),
		Entry(
			"series of different lengths",
			func() cesium.Frame {
				return telem.MultiFrame(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(10, 11, 12),
					telem.NewSeriesV[int64](1, 2),
				})
			},
			"frame must have the same length for all series",
		),
		Entry(
			"no index series",
			func() cesium.Frame {
				return telem.UnaryFrame(data, telem.NewSeriesV[int64](1, 2))
			},
			"received no data for index channel",
		),
	)

	It(
		"Should refuse a stream-only writer whose start lies in stored data",
		func(ctx SpecContext) {
			p := open(ctx, storageConfig{Channels: keys, Start: 10 * telem.SecondTS})
			MustSucceed(p.Write(ctx, fr(secs(10, 11, 12), 1, 2, 3)))
			MustSucceed(p.Commit(ctx))
			Expect(p.Close()).To(Succeed())
			s, err := db.OpenWriter(ctx, storageConfig{
				Channels: keys,
				Start:    11 * telem.SecondTS,
				Mode:     writer.ModeStreamOnly,
			})
			if err == nil {
				_ = s.Close()
			}
			Expect(err).To(MatchError(ContainSubstring(
				"cannot open writer because there is already data in the writer's " +
					"time range",
			)))
		},
	)

	It(
		"Should hand off control on a set authority after a write error",
		func(ctx SpecContext) {
			waiting := open(ctx, storageConfig{
				ControlSubject: control.Subject{Key: "waiting"},
				Channels:       keys,
				Start:          10 * telem.SecondTS,
				Authorities:    []control.Authority{100},
			})
			sw := MustSucceed(db.svc.NewStream(ctx, writer.Config{
				ControlSubject:   control.Subject{Key: "failing"},
				Keys:             channel.KeysFromUint32(keys),
				Start:            10 * telem.SecondTS,
				Authorities:      []control.Authority{200},
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			req, res := confluence.Attach(sw, 10)
			sCtx, cancel := signal.WithCancel(ctx)
			DeferCleanup(cancel)
			sw.Flow(sCtx, confluence.CloseOutputInletsOnExit())
			DeferCleanup(req.Close)
			req.Inlet() <- writer.Request{
				Command: writer.CommandWrite,
				Frame:   frame.NewFromStorage(fr(secs(12, 11), 1, 2)),
			}
			r := <-res.Outlet()
			Expect(r.Err).To(MatchError(ContainSubstring("index out of order")))
			req.Inlet() <- writer.Request{
				Command: writer.CommandSetAuthority,
				Config:  writer.Config{Authorities: []control.Authority{0}},
			}
			r = <-res.Outlet()
			Expect(r.Command).To(Equal(writer.CommandSetAuthority))
			Expect(r.Err).ToNot(HaveOccurred())
			Expect(r.Authorized).To(BeTrue())
			written := MustSucceed(waiting.Write(ctx, fr(secs(20, 21), 1, 2)))
			Expect(written.Get(data).Len()).To(Equal(int64(2)))
		},
	)
})
