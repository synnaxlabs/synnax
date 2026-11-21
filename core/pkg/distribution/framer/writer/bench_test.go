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
	"fmt"
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	xtestutil "github.com/synnaxlabs/x/testutil"
)

// openVirtual opens a node with n virtual channels on the leaseholder.
func openVirtual(
	b *testing.B,
	leaseholder node.Key,
	n int,
) (testutil.Node, channel.Keys) {
	b.Helper()
	gomega.RegisterTestingT(b)
	dist := testutil.OpenNode(b.Context())
	b.Cleanup(func() {
		if err := dist.Close(); err != nil {
			b.Error(err)
		}
	})
	channels := make([]channel.Channel, n)
	for i := range channels {
		channels[i] = channel.Channel{
			Name:        fmt.Sprintf("virtual_%d", i),
			DataType:    telem.Int64T,
			Leaseholder: leaseholder,
			Virtual:     true,
		}
	}
	channels = xtestutil.MustSucceed(dist.CreateChannels(b.Context(), channels))
	return dist, channel.KeysFromChannels(channels)
}

var paths = []struct {
	name        string
	leaseholder node.Key
}{
	{name: "host"},
	{name: "free", leaseholder: node.KeyFree},
}

func BenchmarkWrite(b *testing.B) {
	for _, path := range paths {
		for _, channels := range []int{1, 100} {
			for _, samples := range []int{1, 1000} {
				name := fmt.Sprintf(
					"%s/channels=%d/samples=%d",
					path.name,
					channels,
					samples,
				)
				b.Run(name, func(b *testing.B) {
					dist, keys := openVirtual(b, path.leaseholder, channels)
					w := xtestutil.MustSucceed(dist.Framer.OpenWriter(
						b.Context(),
						writer.Config{Keys: keys, Start: 10 * telem.SecondTS},
					))
					series := make([]telem.Series, channels)
					for i := range series {
						series[i] = telem.NewSeries(make([]int64, samples))
					}
					b.ReportAllocs()
					b.ResetTimer()
					for range b.N {
						// The writer stamps the series in place, so each write gets its
						// own slice.
						fr := frame.NewMulti(
							keys,
							append([]telem.Series(nil), series...),
						)
						if authorized, err := w.Write(fr); err != nil || !authorized {
							b.Fatalf(
								"write failed: authorized=%v, err=%v",
								authorized,
								err,
							)
						}
					}
					if err := w.Close(); err != nil {
						b.Fatal(err)
					}
				})
			}
		}
	}
}

func BenchmarkOpen(b *testing.B) {
	for _, path := range paths {
		for _, channels := range []int{1, 100} {
			b.Run(
				fmt.Sprintf("%s/channels=%d", path.name, channels),
				func(b *testing.B) {
					dist, keys := openVirtual(b, path.leaseholder, channels)
					cfg := writer.Config{Keys: keys, Start: 10 * telem.SecondTS}
					b.ReportAllocs()
					b.ResetTimer()
					for range b.N {
						w, err := dist.Framer.OpenWriter(b.Context(), cfg)
						if err != nil {
							b.Fatal(err)
						}
						if err = w.Close(); err != nil {
							b.Fatal(err)
						}
					}
				},
			)
		}
	}
}

// openStored opens a node with an index channel and n data channels on it, all stored,
// and returns the keys of the index and then of the data channels.
func openStored(b *testing.B, n int) (testutil.Node, channel.Keys) {
	b.Helper()
	gomega.RegisterTestingT(b)
	dist := testutil.OpenNode(b.Context())
	b.Cleanup(func() {
		if err := dist.Close(); err != nil {
			b.Error(err)
		}
	})
	idx := xtestutil.MustSucceed(dist.CreateChannels(b.Context(), []channel.Channel{{
		Name:     "time",
		DataType: telem.TimestampT,
		IsIndex:  true,
	}}))[0]
	data := make([]channel.Channel, n)
	for i := range data {
		data[i] = channel.Channel{
			Name:       fmt.Sprintf("data_%d", i),
			DataType:   telem.Int64T,
			LocalIndex: idx.LocalKey,
		}
	}
	data = xtestutil.MustSucceed(dist.CreateChannels(b.Context(), data))
	return dist, append(channel.Keys{idx.Key()}, channel.KeysFromChannels(data)...)
}

// storedFrame returns a frame of one sample at ts for the index and each data channel
// of keys.
func storedFrame(keys channel.Keys, ts telem.TimeStamp) frame.Frame {
	series := make([]telem.Series, len(keys))
	series[0] = telem.NewSeries([]telem.TimeStamp{ts})
	for i := 1; i < len(series); i++ {
		series[i] = telem.NewSeries([]int64{int64(i)})
	}
	return frame.NewMulti(keys, series)
}

// BenchmarkWriteStored writes stored channels through the skip predicate. With
// contended set, a writer with less authority writes the same channels once before each
// write of the holder, and Cesium skips its every write. The steps are in turn, so the
// allocations of an op do not move with scheduling.
func BenchmarkWriteStored(b *testing.B) {
	for _, contended := range []bool{false, true} {
		for _, channels := range []int{1, 100} {
			name := fmt.Sprintf("channels=%d", channels)
			if contended {
				name += "/contended"
			}
			b.Run(name, func(b *testing.B) {
				dist, keys := openStored(b, channels)
				open := func(subject string, auth control.Authority) *writer.Writer {
					return xtestutil.MustSucceed(dist.Framer.OpenWriter(
						b.Context(),
						writer.Config{
							ControlSubject: control.Subject{Key: subject},
							Keys:           keys,
							Start:          10 * telem.SecondTS,
							Authorities:    []control.Authority{auth},
						},
					))
				}
				w := open("holder", control.AuthorityAbsolute)
				var (
					lw    *writer.Writer
					lower frame.Frame
				)
				if contended {
					lw = open("contender", 1)
					lower = storedFrame(keys, 10*telem.SecondTS)
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := range b.N {
					if lw != nil {
						if _, err := lw.Write(lower); err != nil {
							b.Fatal(err)
						}
					}
					fr := storedFrame(keys, telem.TimeStamp(i+10)*telem.SecondTS)
					if authorized, err := w.Write(fr); err != nil || !authorized {
						b.Fatalf("write failed: authorized=%v, err=%v", authorized, err)
					}
				}
				b.StopTimer()
				if lw != nil {
					if err := lw.Close(); err != nil {
						b.Fatal(err)
					}
				}
				if err := w.Close(); err != nil {
					b.Fatal(err)
				}
			})
		}
	}
}
