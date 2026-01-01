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
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/testutil"
	xfs "github.com/synnaxlabs/x/io/fs"
	. "github.com/synnaxlabs/x/testutil"
)

var (
	ctx         = context.Background()
	fileSystems = testutil.FileSystems
)

func openDBOnFS(fs xfs.FS) *cesium.DB {
	return MustSucceed(cesium.Open(ctx,
		"",
		cesium.WithFS(fs),
		cesium.WithInstrumentation(PanicLogger()),
	))
}

func channelKeyToPath(key cesium.ChannelKey) string {
	return strconv.Itoa(int(key))
}

func TestCesium(t *testing.T) {
	runtime.GOMAXPROCS(4)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cesium Suite")
}

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
})
