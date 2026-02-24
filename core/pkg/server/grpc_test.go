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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/server"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Grpc", func() {
	It("Should start a grpc server", func() {
		b := MustSucceed(server.Serve(server.Config{
			ListenAddress: "localhost:26260",
			Security: server.SecurityConfig{
				Insecure: new(true),
			},
			Debug: new(true),
			Branches: []server.Branch{
				&server.GRPCBranch{},
			},
		}))
		time.Sleep(10 * time.Millisecond)
		Expect(b.Close()).To(Succeed())
	})
})
