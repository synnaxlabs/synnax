// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"os"
	"testing"
)

var logger *zap.Logger

func openMemDB() cesium.DB {
	return MustSucceed(cesium.Open(
		"./testdata",
		cesium.MemBacked(),
		cesium.WithLogger(logger),
	))
}

func TestCaesium(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cesium Suite")
}

var _ = BeforeSuite(func() {
	logger = MustSucceed(zap.NewDevelopment())
	logger = zap.NewNop()
	zap.ReplaceGlobals(logger)
})

var _ = AfterSuite(func() {
	Expect(os.RemoveAll("./testdata/cesium")).To(Succeed())
	Expect(os.RemoveAll("./testdata/kv")).To(Succeed())
})
