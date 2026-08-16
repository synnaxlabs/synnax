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
	"strconv"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/alamos/testutil"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func openDBOnFS(ctx context.Context, fs fs.FS, opts ...cesium.Option) *cesium.DB {
	return MustSucceed(cesium.Open(ctx,
		"",
		append([]cesium.Option{
			cesium.WithFS(fs),
			cesium.WithInstrumentation(PanicLogger()),
		}, opts...)...,
	))
}

func mustOpenDBOnFS(ctx context.Context, fs fs.FS, opts ...cesium.Option) *cesium.DB {
	return DeferClose(openDBOnFS(ctx, fs, opts...))
}

func channelKeyToPath(key cesium.ChannelKey) string { return strconv.Itoa(int(key)) }

// virtualChannel returns a virtual channel with the given key and name.
func virtualChannel(key cesium.ChannelKey, name string) cesium.Channel {
	return cesium.Channel{
		Key:      key,
		Name:     name,
		DataType: telem.Int64T,
		Virtual:  true,
	}
}

func TestCesium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cesium Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()
