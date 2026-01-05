// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos_test

import (
	"testing"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

var devIns alamos.Instrumentation

func TestAlamos(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Alamos Suite")
}

var _ = BeforeSuite(func() {
	devIns = Instrumentation("alamos-test", InstrumentationConfig{
		Trace: config.True(),
	})
})
