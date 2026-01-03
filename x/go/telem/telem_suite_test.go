// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
)

var ctx = context.Background()

var _ = BeforeSuite(func() {
	zap.ReplaceGlobals(MustSucceed(zap.NewDevelopment()))
})

func TestTelem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Telem Suite")
}
