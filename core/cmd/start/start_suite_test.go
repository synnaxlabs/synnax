// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package start_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/cmd"
	. "github.com/synnaxlabs/x/testutil"
)

// coreEnv makes the test binary run as the Core instead of running the specs.
const coreEnv = "SYNNAX_START_TEST_CORE"

// TestMain runs the Core when coreEnv is set, so specs can start a real Core process
// from this binary without building one.
func TestMain(m *testing.M) {
	if os.Getenv(coreEnv) != "" {
		cmd.RunMain()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

func TestStart(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Start Suite")
}
