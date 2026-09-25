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
	It("Should stop when closed before its branches begin serving", func() {
		b := &lateBranch{release: make(chan struct{})}
		s := MustSucceed(server.Serve(server.Config{
			Security:  server.SecurityConfig{Insecure: new(true)},
			Listeners: []server.Listener{{Address: "localhost:0"}},
			Branches:  []server.Branch{b},
		}))
		closed := make(chan error, 1)
		go func() {
			defer GinkgoRecover()
			closed <- s.Close()
		}()
		close(b.release)
		Eventually(closed, 10*time.Second).Should(Receive(BeNil()))
	})
})

// lateBranch holds its routine short of Serve until the spec releases it, so Stop
// always lands first and only the Server can free the listener.
type lateBranch struct{ release chan struct{} }

var _ server.Branch = (*lateBranch)(nil)

func (*lateBranch) Key() string { return "late" }

func (*lateBranch) Routing() server.BranchRouting {
	return server.BranchRouting{
		Policy:   server.RoutingPolicyServeAlwaysPreferSecure,
		Matchers: []cmux.Matcher{cmux.Any()},
	}
}

func (*lateBranch) Init(server.BranchContext) {}

func (b *lateBranch) Serve(ctx server.BranchContext) error {
	<-b.release
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

func (*lateBranch) Stop() {}
