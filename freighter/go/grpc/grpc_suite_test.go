// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGRPC(t *testing.T) {
	RegisterFailHandler(Fail)
	// The suites here share a connection pool that lazily dials the server on the
	// first RPC and caches the connection for the rest of the suite (the pool is
	// closed in each container's teardown). The spec that first dials therefore
	// registers a pooled connection that outlives it by design, so per-spec
	// goroutine-leak checking is not applicable. Container-level leak checks in each
	// BeforeAll still verify the pool and server are fully shut down at teardown.
	//nolint:leaklint
	RunSpecs(t, "GRPC Suite")
}
