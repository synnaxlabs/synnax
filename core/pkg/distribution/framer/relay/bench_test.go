// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay_test

import (
	"testing"

	"github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	xtestutil "github.com/synnaxlabs/x/testutil"
)

const benchWindow = 32

// openFreeStreamer runs a relay and returns its free-write inlet and the outlet of a
// streamer on one free channel, past its open ack.
func openFreeStreamer(
	b *testing.B,
) (confluence.Inlet[relay.Response], confluence.Outlet[relay.Response]) {
	b.Helper()
	gomega.RegisterTestingT(b)
	store := storagetestutil.OpenLayer(b.Context())
	freeWrites := confluence.NewStream[relay.Response](benchWindow)
	r := xtestutil.MustSucceed(relay.Open(relay.Config{
		TS:                 store.TS,
		FreeWrites:         freeWrites,
		StreamerBufferSize: 2 * benchWindow,
	}))
	stopRelay := xtestutil.GoRun(b.Context(), r.Run)
	streamer := xtestutil.MustSucceed(r.NewStreamer(relay.StreamerConfig{
		Keys:        channel.Keys{channel.NewKey(node.KeyFree, 1)},
		SendOpenAck: new(true),
	}))
	sCtx, cancel := signal.WithCancel(b.Context())
	req, res := confluence.Attach(streamer, 2*benchWindow)
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	<-res.Outlet()
	b.Cleanup(func() {
		req.Close()
		cancel()
		confluence.Drain(res)
		if err := stopRelay(); err != nil {
			b.Error(err)
		}
		if err := store.Close(); err != nil {
			b.Error(err)
		}
	})
	return freeWrites, res
}

func BenchmarkFreeWrite(b *testing.B) {
	write := relay.Response{
		Frame: frame.NewUnary(
			channel.NewKey(node.KeyFree, 1),
			telem.NewSeriesV[int64](1),
		),
	}
	b.Run("PingPong", func(b *testing.B) {
		in, out := openFreeStreamer(b)
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			in.Inlet() <- write
			<-out.Outlet()
		}
	})
	b.Run("Window32", func(b *testing.B) {
		in, out := openFreeStreamer(b)
		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			for range benchWindow {
				in.Inlet() <- write
			}
			for range benchWindow {
				<-out.Outlet()
			}
		}
	})
}
