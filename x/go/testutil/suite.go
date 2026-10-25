// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
)

// SuiteConfig returns the configuration every suite passes to RunSpecs: the one parsed
// from the command line, with all specs in random order and pending specs failing the
// suite. Flags would disable the go test result cache, so the suite sets these itself.
// Call it from the Test function, after the flags are parsed.
func SuiteConfig() types.SuiteConfig {
	cfg, _ := ginkgo.GinkgoConfiguration()
	cfg.RandomizeAllSpecs = true
	cfg.FailOnPending = true
	return cfg
}
