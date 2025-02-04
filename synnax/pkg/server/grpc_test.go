// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/server"
	"github.com/synnaxlabs/x/config"
	. "github.com/synnaxlabs/x/testutil"
	"sync"
	"time"
)

var _ = Describe("Grpc", func() {
	It("Should start a grpc server", func() {
		b := MustSucceed(server.New(server.Config{
			ListenAddress: "localhost:26260",
			Security: server.SecurityConfig{
				Insecure: config.Bool(true),
			},
			Debug: config.Bool(true),
			Branches: []server.Branch{
				&server.GRPCBranch{},
			},
		}))
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			Expect(b.Serve()).To(Succeed())
			wg.Done()
		}()
		time.Sleep(10 * time.Millisecond)
		b.Stop()
		wg.Wait()
	})
})
