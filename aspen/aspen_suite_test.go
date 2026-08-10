// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	. "github.com/synnaxlabs/x/testutil"
)

const (
	// ephemeralAddress binds to a port the operating system chooses, so parallel
	// suites never contend for the same one.
	ephemeralAddress aspen.Address = "localhost:0"
	// unreachableAddress never accepts a connection. Port 1 needs privileges no test
	// process has.
	unreachableAddress aspen.Address = "localhost:1"
)

func TestAspen(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Aspen Suite")
}

var _ = ShouldNotLeakGoroutinesPerSpec()
