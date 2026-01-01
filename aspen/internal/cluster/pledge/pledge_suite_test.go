// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pledge_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
	"testing"
)

var (
	ctx context.Context
	ins alamos.Instrumentation
)

func TestMembership(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Membership Suite")
}

var _ = BeforeSuite(func() {
	ins = Instrumentation("pledge", InstrumentationConfig{Log: config.False()})
	ctx = context.Background()
})
