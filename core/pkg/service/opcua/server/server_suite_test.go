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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/opcua/server"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var db *gorp.DB

var _ = BeforeSuite(func() {
	ShouldNotLeakGoroutines()
	db = DeferClose(gorp.Wrap(memkv.New()))
	StartRun(db.Run)
})

func TestServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Service OPC UA Server Suite", SuiteConfig())
}

var _ = ShouldNotLeakGoroutinesPerSpec()

// newConnection returns a connection to endpoint with the schema defaults.
func newConnection(endpoint string) server.Connection {
	var c server.Connection
	c.SetDefaults()
	c.Endpoint = endpoint
	return c
}
