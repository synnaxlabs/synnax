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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/synnaxlabs/synnax/cmd/exit"
)

var _ = Describe("Data directory lock", func() {
	It("Should exit with the locked code when another Core holds the data", func() {
		data := GinkgoT().TempDir()
		startCore("--mem=false", "--data", data)
		second, _, _ := launchCore("--mem=false", "--data", data)
		Eventually(second, 30*time.Second).Should(gexec.Exit(exit.DataLocked))
		Expect(second.Out).To(gbytes.Say("Failed to acquire lock"))
	})
})
