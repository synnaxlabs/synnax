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
	It("Should let each Branch close the listener that the Branches share", func() {
		branches := []*closingBranch{newClosingBranch("a"), newClosingBranch("b")}
		s := MustSucceed(server.Serve(server.Config{
			Security:  server.SecurityConfig{Insecure: new(true)},
			Listeners: []server.Listener{{Address: "localhost:0"}},
			Branches:  []server.Branch{branches[0], branches[1]},
		}))
		for _, b := range branches {
			Eventually(b.serving).Should(BeClosed())
		}
		Expect(s.Close()).To(Succeed())
		for _, b := range branches {
			Expect(b.closeErr).To(Succeed())
		}
	})
})

// closingBranch closes its listener on Stop, as an http.Server does on Shutdown.
type closingBranch struct {
	key      string
	serving  chan struct{}
	lis      net.Listener
	closeErr error
}

var _ server.Branch = (*closingBranch)(nil)

func newClosingBranch(key string) *closingBranch {
	return &closingBranch{key: key, serving: make(chan struct{})}
}

func (b *closingBranch) Key() string { return b.key }

func (*closingBranch) Routing() server.BranchRouting {
	return server.BranchRouting{
		Policy:   server.RoutingPolicyServeAlwaysPreferSecure,
		Matchers: []cmux.Matcher{cmux.Any()},
	}
}

func (*closingBranch) Init(server.BranchContext) {}

func (b *closingBranch) Serve(ctx server.BranchContext) error {
	b.lis = ctx.Lis
	close(b.serving)
	_, err := ctx.Lis.Accept()
	return err
}

func (b *closingBranch) Stop() { b.closeErr = b.lis.Close() }

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
