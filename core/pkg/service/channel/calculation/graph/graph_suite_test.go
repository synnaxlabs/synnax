// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package graph_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	. "github.com/synnaxlabs/x/testutil"
)

func TestGraph(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service Channel Calculation Graph Suite")
}

// Specs that delete a calculated channel's dependency drive reactive status writes
// through the suite-level node DB, which can push Pebble to rotate its write-ahead log
// and spin up a fresh LogWriter.flushLoop mid-spec. That goroutine belongs to the
// shared DB (torn down at suite end), not to anything the spec opens, so it is ignored
// here rather than flagged as a per-spec leak.
var _ = ShouldNotLeakGoroutinesPerSpec(
	LeakIgnoring(gleak.IgnoringCreator("github.com/cockroachdb/pebble/v2/record.NewLogWriter")),
)
