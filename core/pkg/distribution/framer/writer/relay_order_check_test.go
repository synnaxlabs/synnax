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
	"context"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	xtime "github.com/synnaxlabs/x/time"
)

// pausedInlet holds the first sender that takes its inlet once armed, until release
// closes: the session goroutine is then between its Cesium write and its relay.
type pausedInlet struct {
	*confluence.Stream[relay.Response]
	armed   atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (p *pausedInlet) Inlet() chan<- relay.Response {
	if p.armed.CompareAndSwap(true, false) {
		close(p.entered)
		<-p.release
	}
	return p.Stream.Inlet()
}

// slowFS holds the first write to an index file once armed, until release closes.
type slowFS struct {
	fs.FS
	armed   *atomic.Bool
	entered chan struct{}
	release chan struct{}
}

func (s *slowFS) Sub(name string) (fs.FS, error) {
	sub, err := s.FS.Sub(name)
	if err != nil {
		return nil, err
	}
	return &slowFS{FS: sub, armed: s.armed, entered: s.entered, release: s.release}, nil
}

func (s *slowFS) Open(name string, flag int) (fs.File, error) {
	f, err := s.FS.Open(name, flag)
	if err != nil || !strings.HasSuffix(name, "index.domain") {
		return f, err
	}
	return &slowFile{File: f, fs: s}, nil
}

type slowFile struct {
	fs.File
	fs *slowFS
}

func (f *slowFile) hold() {
	if f.fs.armed.CompareAndSwap(true, false) {
		close(f.fs.entered)
		<-f.fs.release
	}
}

func (f *slowFile) Write(p []byte) (int, error) {
	f.hold()
	return f.File.Write(p)
}

func (f *slowFile) WriteAt(p []byte, off int64) (int, error) {
	f.hold()
	return f.File.WriteAt(p, off)
}

// The two relay order specs fail by design: the relay runs outside the claim lock, so
// a frame of a writer that lost control can relay after a frame of the new holder.
// That is the accepted limit of the control fix, kept here as a check.
var _ = Describe("Relay order check", func() {
	It("Should relay the frames of a handoff in write order when a persist is slow",
		func(ctx SpecContext) {
			disk := &slowFS{
				FS:      fs.NewMem(),
				armed:   &atomic.Bool{},
				entered: make(chan struct{}),
				release: make(chan struct{}),
			}
			db := DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(disk))))
			StartRun(db.Run)
			idx, data := GenerateChannelKey(), GenerateChannelKey()
			Expect(db.CreateChannel(
				ctx,
				cesium.Channel{
					Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
				},
				cesium.Channel{
					Key: data, Name: "cmd", Index: idx, DataType: telem.Int64T,
				},
			)).To(Succeed())
			writes := confluence.NewStream[relay.Response](100)
			svc := MustSucceed(writer.NewService(writer.ServiceConfig{
				TS:     db,
				Writes: writes,
				Clock:  xtime.Real,
			}))
			keys := channel.KeysFromUint32([]uint32{idx, data})
			fr := func(ts telem.TimeStamp, v int64) frame.Frame {
				return frame.NewMulti(keys, []telem.Series{
					telem.NewSeriesSecondsTSV(ts),
					telem.NewSeriesV(v),
				})
			}
			a := MustSucceed(svc.Open(context.WithoutCancel(ctx), writer.Config{
				ControlSubject:           xcontrol.Subject{Key: "a", Name: "a"},
				Keys:                     keys,
				Start:                    10 * telem.SecondTS,
				Authorities:              []xcontrol.Authority{100},
				EnableAutoCommit:         new(true),
				AutoIndexPersistInterval: writer.AlwaysIndexPersistOnAutoCommit,
				Sync:                     new(false),
			}))
			disk.armed.Store(true)
			Expect(a.Write(fr(10, 1))).To(BeTrue())
			Eventually(disk.entered).Should(BeClosed())
			b := MustSucceed(svc.Open(context.WithoutCancel(ctx), writer.Config{
				ControlSubject:   xcontrol.Subject{Key: "b", Name: "b"},
				Keys:             keys,
				Start:            10 * telem.SecondTS,
				Authorities:      []xcontrol.Authority{200},
				EnableAutoCommit: new(false),
				Sync:             new(true),
			}))
			Expect(b.Write(fr(11, 2))).To(BeTrue())
			close(disk.release)
			var values []int64
			for range 2 {
				var res relay.Response
				Eventually(writes.Outlet()).
					WithTimeout(2 * time.Second).
					Should(Receive(&res))
				values = append(
					values,
					res.Frame.Get(channel.Key(data)).Series[0].ValueAt[int64](0),
				)
			}
			Expect(b.Commit()).Error().ToNot(HaveOccurred())
			Expect(a.Close()).To(Succeed())
			Expect(b.Close()).To(Succeed())
			stored := MustSucceed(db.Read(ctx, telem.TimeRangeMax, data))
			Expect(stored.Get(data).Series[0].ValueAt[int64](-1)).To(Equal(int64(2)))
			// Storage ends on 2, so a streamer must end on 2 too.
			Expect(values).To(Equal([]int64{1, 2}))
		})

	It("Should relay no frame of a writer after a frame of the writer that took "+
		"control from it", func(ctx SpecContext) {
		db := DeferClose(MustSucceed(cesium.Open(ctx, "", cesium.WithFS(fs.NewMem()))))
		StartRun(db.Run)
		idx, data := GenerateChannelKey(), GenerateChannelKey()
		Expect(db.CreateChannel(
			ctx,
			cesium.Channel{
				Key: idx, Name: "time", IsIndex: true, DataType: telem.TimestampT,
			},
			cesium.Channel{Key: data, Name: "cmd", Index: idx, DataType: telem.Int64T},
		)).To(Succeed())
		inlet := &pausedInlet{
			Stream:  confluence.NewStream[relay.Response](100),
			entered: make(chan struct{}),
			release: make(chan struct{}),
		}
		svc := MustSucceed(writer.NewService(writer.ServiceConfig{
			TS:     db,
			Writes: inlet,
			Clock:  xtime.Real,
		}))
		keys := channel.KeysFromUint32([]uint32{idx, data})
		open := func(name string, auth xcontrol.Authority, sync bool) *writer.Writer {
			GinkgoHelper()
			return MustSucceed(svc.Open(context.WithoutCancel(ctx), writer.Config{
				ControlSubject:   xcontrol.Subject{Key: name, Name: name},
				Keys:             keys,
				Start:            10 * telem.SecondTS,
				Authorities:      []xcontrol.Authority{auth},
				EnableAutoCommit: new(true),
				Sync:             new(sync),
			}))
		}
		fr := func(ts telem.TimeStamp, v int64) frame.Frame {
			return frame.NewMulti(keys, []telem.Series{
				telem.NewSeriesSecondsTSV(ts),
				telem.NewSeriesV(v),
			})
		}
		a := open("a", 100, false)
		inlet.armed.Store(true)
		Expect(a.Write(fr(10, 1))).To(BeTrue())
		Eventually(inlet.entered).Should(BeClosed())
		b := open("b", 200, true)
		Expect(b.Write(fr(11, 2))).To(BeTrue())
		close(inlet.release)
		var values []int64
		for range 2 {
			var res relay.Response
			Eventually(inlet.Outlet()).
				WithTimeout(2 * time.Second).
				Should(Receive(&res))
			values = append(
				values,
				res.Frame.Get(channel.Key(data)).Series[0].ValueAt[int64](0),
			)
		}
		Expect(a.Close()).To(Succeed())
		Expect(b.Close()).To(Succeed())
		// Storage holds 1 then 2. A streamer must see the same last value.
		Expect(values).To(Equal([]int64{1, 2}))
	})

	It("Should keep the control of a virtual channel that a writer holds across a "+
		"delete claim", func(ctx SpecContext) {
		db := mustOpenDBOnFS(ctx, fs.NewMem())
		key := GenerateChannelKey()
		Expect(db.CreateChannel(ctx, virtualChannel(key, "cmd"))).To(Succeed())
		open := func(name string, auth xcontrol.Authority) *writer.Writer {
			GinkgoHelper()
			return MustSucceed(db.svc.Open(context.WithoutCancel(ctx), writer.Config{
				ControlSubject: xcontrol.Subject{Key: name, Name: name},
				Keys:           channel.Keys{channel.Key(key)},
				Start:          10 * telem.SecondTS,
				Authorities:    []xcontrol.Authority{auth},
				Sync:           new(true),
			}))
		}
		a := open("a", 200)
		// A delete whose transaction then fails releases its claim.
		release, claimErr := db.svc.ClaimDelete(channel.Keys{channel.Key(key)})
		if claimErr == nil {
			release()
		}
		b := open("b", 100)
		authorized := MustSucceed(b.Write(
			frame.NewUnary(channel.Key(key), telem.NewSeriesV[int64](7)),
		))
		Expect(a.Close()).To(Succeed())
		Expect(b.Close()).To(Succeed())
		Expect(authorized).To(BeFalse(), "b writes while a holds a higher authority")
		Expect(claimErr).To(MatchError(writer.ErrChannelInUse))
	})

	It(
		"Should refuse a lower writer on a virtual channel with no delete claim (baseline)",
		func(ctx SpecContext) {
			db := mustOpenDBOnFS(ctx, fs.NewMem())
			key := GenerateChannelKey()
			Expect(db.CreateChannel(ctx, virtualChannel(key, "cmd"))).To(Succeed())
			open := func(name string, auth xcontrol.Authority) *writer.Writer {
				GinkgoHelper()
				return MustSucceed(
					db.svc.Open(context.WithoutCancel(ctx), writer.Config{
						ControlSubject: xcontrol.Subject{Key: name, Name: name},
						Keys:           channel.Keys{channel.Key(key)},
						Start:          10 * telem.SecondTS,
						Authorities:    []xcontrol.Authority{auth},
						Sync:           new(true),
					}),
				)
			}
			a := open("a", 200)
			b := open("b", 100)
			authorized := MustSucceed(b.Write(
				frame.NewUnary(channel.Key(key), telem.NewSeriesV[int64](7)),
			))
			Expect(a.Close()).To(Succeed())
			Expect(b.Close()).To(Succeed())
			Expect(authorized).To(BeFalse())
		},
	)
})
