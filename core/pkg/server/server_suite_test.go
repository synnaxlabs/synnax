// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	"testing"

	"github.com/gofiber/contrib/v3/monitor"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/synnaxlabs/x/testutil"
)

// The gofiber monitor middleware (mounted on /metrics when a server runs in debug
// mode) starts a process-global stats-refresh goroutine, guarded by a package-level
// sync.Once, that can never be stopped. Trigger it at package init so it is part of
// every spec's goroutine-leak baseline instead of being reported as a leak by the
// first spec that enables debug mode.
var _ = monitor.New()

var _ = ShouldNotLeakGoroutinesPerSpec()

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Server Suite")
}
