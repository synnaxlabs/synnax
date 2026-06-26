// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTP(t *testing.T) {
	RegisterFailHandler(Fail)
	// freighter's HTTP transport keeps net/http idle keep-alive connections and
	// fasthttp server-worker goroutines alive past a spec's end; they are not
	// guaranteed to drain within the per-spec window, so per-spec goroutine-leak
	// checking is not applicable to this suite.
	//nolint:leak
	RunSpecs(t, "HTTP Suite")
}
