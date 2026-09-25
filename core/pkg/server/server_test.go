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
	"net"
	"time"

	"github.com/cockroachdb/cmux"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/server"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Server", func() {
	Describe("Addresses", func() {
		It("Should keep the configured host and resolve a port bound on 0", func() {
			s := MustOpen(server.Serve(server.Config{
				Security:  server.SecurityConfig{Insecure: new(true)},
				Listeners: []server.Listener{{Address: "localhost:0"}},
				Branches: []server.Branch{
					&server.SecureHTTPBranch{
						MaxIdleWorkerDuration: 100 * time.Millisecond,
					},
				},
			}))
			Expect(s.Addresses()).To(HaveLen(1))
			addr := s.Addresses()[0]
			Expect(addr.Host()).To(Equal("localhost"))
			Expect(addr.Port()).ToNot(BeZero())
			Expect(MustSucceed(net.Dial("tcp", addr.String())).Close()).To(Succeed())
		})
	})
	It("Should stop a branch that Stop does not unblock", func() {
		s := MustSucceed(server.Serve(server.Config{
			Security:  server.SecurityConfig{Insecure: new(true)},
			Listeners: []server.Listener{{Address: "localhost:0"}},
			Branches:  []server.Branch{&stuckBranch{}},
		}))
		closed := make(chan error, 1)
		go func() {
			defer GinkgoRecover()
			closed <- s.Close()
		}()
		Eventually(closed, 10*time.Second).Should(Receive(BeNil()))
	})
})

// stuckBranch parks in Accept and keeps its listener out of Stop's reach, which is the
// state a Branch is in until it reaches Serve. Only the Server can free it.
type stuckBranch struct{}

var _ server.Branch = (*stuckBranch)(nil)

func (*stuckBranch) Key() string { return "stuck" }

func (*stuckBranch) Routing() server.BranchRouting {
	return server.BranchRouting{
		Policy:   server.RoutingPolicyServeAlwaysPreferSecure,
		Matchers: []cmux.Matcher{cmux.Any()},
	}
}

func (*stuckBranch) Init(server.BranchContext) {}

func (*stuckBranch) Serve(ctx server.BranchContext) error {
	for {
		conn, err := ctx.Lis.Accept()
		if err != nil {
			return err
		}
		if err = conn.Close(); err != nil {
			return err
		}
	}
}

func (*stuckBranch) Stop() {}
