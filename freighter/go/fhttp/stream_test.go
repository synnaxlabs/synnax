// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package fhttp_test

import (
	"github.com/gofiber/fiber/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/fhttp"
	. "github.com/synnaxlabs/freighter/testutil"
	"github.com/synnaxlabs/x/address"
	. "github.com/synnaxlabs/x/testutil"
)

type streamImplementation struct{ app *fiber.App }

var _ StreamImplementation = (*streamImplementation)(nil)

func (i *streamImplementation) Start(
	host address.Address, ins alamos.Instrumentation,
) (StreamServer, StreamClient) {
	i.app = fiber.New(fiber.Config{DisableStartupMessage: true})
	router := MustSucceed(fhttp.NewRouter(fhttp.RouterConfig{
		Instrumentation:     ins,
		StreamWriteDeadline: StreamWriteDeadline,
	}))
	client := MustSucceed(fhttp.NewStreamClient[Request, Response]())
	server := fhttp.NewStreamServer[Request, Response](router, "/")
	router.BindTo(i.app)
	go func() {
		defer GinkgoRecover()
		Expect(i.app.Listen(host.PortString())).To(Succeed())
	}()
	return server, client
}

func (i *streamImplementation) Stop() error { return i.app.Shutdown() }

var _ = Describe("Stream", func() {
	AssertStream(&streamImplementation{})
})
