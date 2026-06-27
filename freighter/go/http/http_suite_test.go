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
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHTTP(t *testing.T) {
	RegisterFailHandler(Fail)
	// Per-spec leak checking is not registered: several specs stand up their own
	// per-spec fiber server and exchange websocket streams, whose connections cannot be
	// guaranteed to drain within the per-spec window. The BeforeSuite and BeforeAll
	// container-level checks still verify that the suite-wide servers are fully torn
	// down at teardown.
	//nolint:leak
	RunSpecs(t, "HTTP Suite")
}

// pollHealth issues a GET to url and closes the response body, returning any error. The
// Eventually health-check loops use it so their polling connections are released rather
// than left pinned in net/http's idle pool.
func pollHealth(url string) error {
	res, err := http.Get(url)
	if err != nil {
		return err
	}
	return res.Body.Close()
}
