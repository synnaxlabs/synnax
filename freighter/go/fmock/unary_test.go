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
	"github.com/synnaxlabs/freighter/fmock"
	. "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
)

type unaryImplementation struct{}

var _ UnaryImplementation = (*unaryImplementation)(nil)

func (i *unaryImplementation) Start(addr address.Address) (UnaryServer, UnaryClient) {
	net := fmock.NewNetwork[Request, Response]()
	server := net.UnaryServer(addr)
	client := net.UnaryClient()
	return server, client
}

func (i *unaryImplementation) Stop() error { return nil }

var _ = Describe("Unary", func() {
	AssertUnary(&unaryImplementation{})
})
