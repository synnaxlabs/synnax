// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/mock"
	"github.com/synnaxlabs/freighter/test"
	"github.com/synnaxlabs/x/address"
)

var _ = Describe("Stream", Ordered, Serial, func() {
	var (
		server *mock.StreamServer[test.Request, test.Response]
		client *mock.StreamClient[test.Request, test.Response]
	)

	BeforeAll(func() {
		server, client = mock.NewStreamPair[test.Request, test.Response](11, 11)
	})

	test.StreamSuite(func() (
		freighter.StreamServer[test.Request, test.Response],
		freighter.StreamClient[test.Request, test.Response],
		address.Address,
	) {
		return server, client, "localhost:0"
	})
})
