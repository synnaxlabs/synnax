// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	"context"
	"io"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func openDBOnFS(ctx context.Context, fs fs.FS) *cesium.DB {
	return MustSucceed(cesium.Open(ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithInstrumentation(PanicLogger()),
	))
}

func mustOpenDBOnFS(ctx context.Context, fs fs.FS) *cesium.DB {
	return DeferClose(openDBOnFS(ctx, fs))
}

func channelKeyToPath(key cesium.ChannelKey) string { return strconv.Itoa(int(key)) }

// openStreamer opens a streamer on db with the given config and starts it in an
// isolated signal context, returning the streamer, its response outlet, and a closer
// that shuts the streamer down.
func openStreamer(db *cesium.DB, cfg cesium.StreamerConfig) (
	cesium.Streamer[cesium.StreamerResponse],
	confluence.Outlet[cesium.StreamerResponse],
	io.Closer,
) {
	streamer := MustSucceed(db.NewStreamer(cfg))
	responses := confluence.NewStream[cesium.StreamerResponse](2)
	streamer.OutTo(responses)
	sCtx, cancel := signal.Isolated()
	streamer.Flow(sCtx, confluence.CloseOutputInletsOnExit())
	return streamer, responses, signal.NewHardShutdown(sCtx, cancel)
}

// virtualChannel returns a virtual channel with the given key and name.
func virtualChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:      key,
		Name:     name,
		DataType: telem.Int64T,
		Virtual:  true,
	}
}

// virtualIndexChannel returns a virtual index channel with the given key and name.
func virtualIndexChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:      key,
		Name:     name,
		DataType: telem.TimeStampT,
		IsIndex:  true,
		Virtual:  true,
	}
}

// virtualDataChannel returns a virtual channel indexed by index.
func virtualDataChannel(key, index cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:      key,
		Name:     name,
		DataType: telem.Int64T,
		Index:    index,
		Virtual:  true,
	}
}

func TestCesium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cesium Suite")
}
