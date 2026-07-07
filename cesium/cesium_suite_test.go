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
	"runtime"
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func openDBOnFS(ctx context.Context, fs xfs.FS) *cesium.DB {
	return MustSucceed(cesium.Open(ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithInstrumentation(PanicLogger()),
	))
}

func channelKeyToPath(key cesium.ChannelKey) string {
	return strconv.Itoa(int(key))
}

// transientChannel returns a transient virtual channel with the given key and name.
func transientChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:       key,
		Name:      name,
		DataType:  telem.Int64T,
		Virtual:   true,
		Transient: true,
	}
}

// virtualIndexChannel returns a transient virtual index channel with the given key
// and name.
func virtualIndexChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:       key,
		Name:      name,
		DataType:  telem.TimeStampT,
		IsIndex:   true,
		Virtual:   true,
		Transient: true,
	}
}

// virtualDataChannel returns a transient virtual channel indexed by index.
func virtualDataChannel(key, index cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:       key,
		Name:      name,
		DataType:  telem.Int64T,
		Index:     index,
		Virtual:   true,
		Transient: true,
	}
}

func TestCesium(t *testing.T) {
	runtime.GOMAXPROCS(4)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cesium Suite")
}

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
})
