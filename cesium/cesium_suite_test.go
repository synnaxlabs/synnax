// Copyright 2023 Synnax Labs, Inc.
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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/testutil"
	"path"
	"strconv"
	"testing"
)

var (
	ctx         = context.Background()
	rootPath    = "cesium-testdata"
	fileSystems map[string]func() xfs.FS
	cleanUp     func() error
)

func openDBOnFS(fs xfs.FS) *cesium.DB {
	return testutil.MustSucceed(cesium.Open(
		rootPath,
		cesium.WithFS(fs),
	))
}

func pathInDBFromKey(key cesium.ChannelKey) string {
	return path.Join(rootPath, strconv.Itoa(int(key)))
}

func TestCesium(t *testing.T) {
	BeforeSuite(func() {
		fileSystems, cleanUp = testutil.FileSystems()
	})

	AfterSuite(func() {
		Expect(cleanUp()).To(Succeed())
	})

	RegisterFailHandler(Fail)
	RunSpecs(t, "Cesium Suite")
}
