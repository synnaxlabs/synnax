// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fmock_test

import (
	. "github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/fmock"
	. "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
)

type streamImplementation struct{}

var _ StreamImplementation = (*streamImplementation)(nil)

func (i *streamImplementation) Start(
	address.Address,
	alamos.Instrumentation,
) (StreamServer, StreamClient) {
	return fmock.NewStreamPair[Request, Response]()
}

func (i *streamImplementation) Stop() error { return nil }

var _ = Describe("Stream", func() {
	AssertStream(&streamImplementation{})
})
